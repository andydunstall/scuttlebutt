package internal

import (
	"sync"

	"go.uber.org/zap"
)

// PeerMap contains this nodes view of all known peers in the cluster.
//
// Note this is thread safe.
type PeerMap struct {
	// peerID is the ID of the local peer.
	peerID string
	// peers contains the set of known peers.
	peers map[string]*Peer
	// mu protects all above fields. Using a RWMutex since expect the workload to be
	// quite read heavy (calculating deltas and digests).
	mu sync.RWMutex

	logger *zap.Logger

	// Note must not hold mu when invoking callback as it may call back to
	// peerMap.
	onJoin   func(peerID string)
	onLeave  func(peerID string)
	onUpdate func(peerID string, key string, value string)
}

func NewPeerMap(
	peerID string,
	peerAddr string,
	onJoin func(peerID string),
	onLeave func(peerID string),
	onUpdate func(peerID string, key string, value string),
	logger *zap.Logger,
) *PeerMap {
	peers := map[string]*Peer{
		peerID: NewPeer(peerID, peerAddr),
	}
	return &PeerMap{
		peerID:   peerID,
		peers:    peers,
		mu:       sync.RWMutex{},
		onJoin:   onJoin,
		onLeave:  onLeave,
		onUpdate: onUpdate,
		logger:   logger,
	}
}

// PeerIDs returns the IDs of the peers known by this node. If includeLocal
// is true the local node is included, otherwise it isn't.
func (m *PeerMap) PeerIDs(includeLocal bool) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peers := make([]string, 0, len(m.peers))
	for peerID, _ := range m.peers {
		if includeLocal || peerID != m.peerID {
			peers = append(peers, peerID)
		}
	}
	return peers
}

func (m *PeerMap) Lookup(peerID string, key string) (PeerEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if peer, ok := m.peers[peerID]; ok {
		return peer.Lookup(key)
	}
	return PeerEntry{}, false
}

func (m *PeerMap) Addr(peerID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if peer, ok := m.peers[peerID]; ok {
		return peer.Addr(), true
	}
	return "", false
}

func (m *PeerMap) Version(peerID string) uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peer, ok := m.peers[peerID]
	if !ok {
		// If we haven't see the peer the version is always 0.
		return 0
	}
	return peer.Version()
}

func (m *PeerMap) PeersEqual(o *PeerMap) bool {
	if len(m.peers) != len(o.peers) {
		return false
	}
	for k, v := range m.peers {
		w, ok := o.peers[k]
		if !ok {
			return false
		}
		if !v.Equal(w) {
			return false
		}
	}
	return true
}

// UpdateLocal updates an entery in this nodes local peer.
func (m *PeerMap) UpdateLocal(key string, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Debug("update local", zap.String("key", key), zap.String("value", value))

	m.peers[m.peerID].UpdateLocal(key, value)
}

func (m *PeerMap) Digest(peerID string) Digest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peer, ok := m.peers[peerID]
	if !ok {
		return Digest{}
	}
	return peer.Digest()
}

func (m *PeerMap) Deltas(peerID string, version uint64) []Delta {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peer, ok := m.peers[peerID]
	if !ok {
		return []Delta{}
	}
	return peer.Deltas(version)
}

func (m *PeerMap) ApplyDigest(digest Digest) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Debug(
		"apply digest",
		zap.Object("digest", digest),
	)

	peer, ok := m.peers[digest.ID]
	if !ok {
		m.logger.Info("node joined", zap.String("joined", digest.ID))

		if m.onJoin != nil {
			m.mu.Unlock()
			m.onJoin(digest.ID)
			m.mu.Lock()
		}

		// Add the peer with a version of 0 given we don't have any state
		// for the peer yet.
		peer = NewPeer(digest.ID, digest.Addr)
		m.peers[digest.ID] = peer
	}
}

func (m *PeerMap) ApplyDelta(delta Delta) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if delta.ID == m.peerID {
		m.logger.Error("received delta update about local peer")
		return
	}

	peer, ok := m.peers[delta.ID]
	if !ok {
		// This should never happen. We only receive digest entries for
		// the peers we requested.
		return
	}

	m.logger.Debug(
		"apply delta",
		zap.String("id", delta.ID),
		zap.String("key", delta.Key),
		zap.String("value", delta.Value),
		zap.Uint64("version", delta.Version),
	)

	peer.UpdateRemote(delta.Key, delta.Value, delta.Version)

	if m.onUpdate != nil {
		m.mu.Unlock()
		m.onUpdate(delta.ID, delta.Key, delta.Value)
		m.mu.Lock()
	}
}
