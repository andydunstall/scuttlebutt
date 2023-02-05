package internal

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// PeerMap contains this nodes view of all known peers in the cluster.
//
// Note this is thread safe.
type PeerMap struct {
	// localAddr is the address of the local node.
	localAddr string
	// peers contains the set of known peers indexed by address.
	peers map[string]*Peer
	// mu protects all above fields. Using a RWMutex since expect the workload to be
	// quite read heavy (calculating deltas and digests).
	mu sync.RWMutex

	logger *zap.Logger

	// Note must not hold mu when invoking callback as it may call back to
	// peerMap.
	onJoin   func(addr string)
	onLeave  func(addr string)
	onUpdate func(addr string, key string, value string)
}

func NewPeerMap(
	localAddr string,
	onJoin func(addr string),
	onLeave func(addr string),
	onUpdate func(addr string, key string, value string),
	logger *zap.Logger,
) *PeerMap {
	peers := map[string]*Peer{
		localAddr: NewPeer(localAddr),
	}
	return &PeerMap{
		localAddr: localAddr,
		peers:     peers,
		mu:        sync.RWMutex{},
		onJoin:    onJoin,
		onLeave:   onLeave,
		onUpdate:  onUpdate,
		logger:    logger,
	}
}

// Addrs returns the addresses of the up peers known by this node. If
// includeLocal is true the local node is included, otherwise it isn't.
func (m *PeerMap) Addrs(includeLocal bool) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peers := make([]string, 0, len(m.peers))
	for addr, peer := range m.peers {
		if peer.Status() != PeerStatusUp {
			continue
		}

		if includeLocal || addr != m.localAddr {
			peers = append(peers, addr)
		}
	}
	return peers
}

func (m *PeerMap) DownPeers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peers := make([]string, 0, len(m.peers))
	for addr, peer := range m.peers {
		if peer.Status() != PeerStatusDown {
			continue
		}
		peers = append(peers, addr)
	}
	return peers
}

func (m *PeerMap) Lookup(addr string, key string) (PeerEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if peer, ok := m.peers[addr]; ok {
		return peer.Lookup(key)
	}
	return PeerEntry{}, false
}

func (m *PeerMap) Version(addr string) uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peer, ok := m.peers[addr]
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

	m.peers[m.localAddr].UpdateLocal(key, value)
}

func (m *PeerMap) SetStatusUp(addr string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	peer, ok := m.peers[addr]
	if !ok {
		return
	}

	// Check if the status has changed to avoid sending duplicate notifications.
	if peer.Status() == PeerStatusUp {
		return
	}

	peer.SetStatusUp()

	if m.onJoin != nil {
		m.onJoin(addr)
	}
}

func (m *PeerMap) SetStatusDown(addr string, expiry time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// The local peers state is always active.
	if addr == m.localAddr {
		return
	}

	peer, ok := m.peers[addr]
	if !ok {
		return
	}

	// Check if the status has changed to avoid sending duplicate notifications.
	if peer.Status() == PeerStatusDown {
		return
	}

	peer.SetStatusDown(expiry)

	if m.onLeave != nil {
		m.onLeave(addr)
	}
}

func (m *PeerMap) Digest(addr string) Digest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peer, ok := m.peers[addr]
	if !ok {
		return Digest{}
	}
	return peer.Digest()
}

func (m *PeerMap) Deltas(addr string, version uint64) []Delta {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peer, ok := m.peers[addr]
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

	peer, ok := m.peers[digest.Addr]
	if !ok {
		m.logger.Info("node joined", zap.String("joined", digest.Addr))

		if m.onJoin != nil {
			m.mu.Unlock()
			m.onJoin(digest.Addr)
			m.mu.Lock()
		}

		// Add the peer with a version of 0 given we don't have any state
		// for the peer yet.
		peer = NewPeer(digest.Addr)
		m.peers[digest.Addr] = peer
	}
}

func (m *PeerMap) ApplyDelta(delta Delta) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if delta.Addr == m.localAddr {
		m.logger.Error("received delta update about local peer")
		return
	}

	peer, ok := m.peers[delta.Addr]
	if !ok {
		// This should never happen. We only receive digest entries for
		// the peers we requested.
		return
	}

	m.logger.Debug(
		"apply delta",
		zap.String("addr", delta.Addr),
		zap.String("key", delta.Key),
		zap.String("value", delta.Value),
		zap.Uint64("version", delta.Version),
	)

	peer.UpdateRemote(delta.Key, delta.Value, delta.Version)

	if m.onUpdate != nil {
		m.mu.Unlock()
		m.onUpdate(delta.Addr, delta.Key, delta.Value)
		m.mu.Lock()
	}
}

func (m *PeerMap) RemoveExpiredPeers() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	expired := []string{}
	for addr, peer := range m.peers {
		if peer.Expiry().After(time.Now()) {
			expired = append(expired, addr)
		}
	}

	for _, addr := range expired {
		delete(m.peers, addr)
	}

	return expired
}
