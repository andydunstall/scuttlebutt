package scuttlebutt

import (
	"sync"
)

// peerMap contains this nodes view of all known peers in the cluster.
//
// Note this is thread safe.
type peerMap struct {
	// peerID is the ID of the local peer.
	peerID string
	// peers contains the set of known peers.
	peers map[string]*peer
	// mu protects all above fields. Using a RWMutex since expect the workload to be
	// quite read heavy (calculating deltas and digests).
	mu sync.RWMutex

	// Note must not hold mu when notifying a subscriber as it may call back to
	// peerMap.
	nodeSubscriber  NodeSubscriber
	eventSubscriber StateSubscriber
}

func newPeerMap(peerID string, peerAddr string, nodeSubscriber NodeSubscriber, eventSubscriber StateSubscriber) *peerMap {
	peers := map[string]*peer{
		peerID: newPeer(peerID, peerAddr),
	}
	return &peerMap{
		peerID:          peerID,
		peers:           peers,
		mu:              sync.RWMutex{},
		nodeSubscriber:  nodeSubscriber,
		eventSubscriber: eventSubscriber,
	}
}

// Peers returns the peer IDs of the peers known by this node (excluding
// ourselves).
func (m *peerMap) Peers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peers := []string{}
	for peerID, _ := range m.peers {
		if peerID != m.peerID {
			peers = append(peers, peerID)
		}
	}
	return peers
}

func (m *peerMap) Lookup(peerID string, key string) (peerEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if peer, ok := m.peers[peerID]; ok {
		return peer.Lookup(key)
	}
	return peerEntry{}, false
}

func (m *peerMap) Addr(peerID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if peer, ok := m.peers[peerID]; ok {
		return peer.Addr(), true
	}
	return "", false
}

// UpdateLocal updates an entery in this nodes local peer.
func (m *peerMap) UpdateLocal(key string, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.peers[m.peerID].UpdateLocal(key, value)
}

// Digest all known peers and their versions. This is used to check for missing
// entries when comparing with another nodes state.
func (m *peerMap) Digest() digest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	digest := digest{}
	for peerID, peer := range m.peers {
		digest[peerID] = peer.Digest()
	}
	return digest
}

// Deltas returns all peer entries whose version exceeds the corresponding peer
// entry in the digest. A peer we know about that is not in the digest returns
// all entries for that peer. The deltas are ordered by version per peer as they
// may be truncated by the transport and we can't have gaps in versions.
func (m *peerMap) Deltas(digest digest) delta {
	m.mu.RLock()
	defer m.mu.RUnlock()

	delta := delta{}
	for peerID, peer := range m.peers {
		entry, ok := digest[peerID]
		// If we know about a peer that is not in the digest, use a version of
		// 0 to send all entries.
		if !ok {
			entry.Version = 0
		}

		if peer.Version() > entry.Version {
			delta[peerID] = peer.Deltas(entry.Version)
		}
	}

	return delta
}

func (m *peerMap) ApplyDigest(digest digest) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for peerID, peerDigest := range digest {
		peer, ok := m.peers[peerID]
		if !ok {
			if m.nodeSubscriber != nil {
				m.mu.Unlock()
				m.nodeSubscriber.NotifyJoin(peerID)
				m.mu.Lock()
			}

			peer = newPeer(peerID, peerDigest.Addr)
			m.peers[peerID] = peer
		}
	}
}

func (m *peerMap) ApplyDeltas(delta delta) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for peerID, peerDelta := range delta {
		// Ignore updates about our own peer.
		if peerID == m.peerID {
			return
		}

		peer, ok := m.peers[peerID]
		if !ok {
			if m.nodeSubscriber != nil {
				m.mu.Unlock()
				m.nodeSubscriber.NotifyJoin(peerID)
				m.mu.Lock()
			}

			peer = newPeer(peerID, peerDelta.Addr)
			m.peers[peerID] = peer
		}

		for _, entry := range peerDelta.Deltas {
			peer.UpdateRemote(entry.Key, entry.Value, entry.Version)

			if m.eventSubscriber != nil {
				m.mu.Unlock()
				m.eventSubscriber.NotifyUpdate(peerID, entry.Key, entry.Value)
				m.mu.Lock()
			}
		}
	}
}
