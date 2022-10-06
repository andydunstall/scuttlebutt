package scuttlebutt

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeNodeSubscriber struct {
	nodesJoined []string
	nodesLeft   []string
}

func NewFakeNodeSubscriber() *fakeNodeSubscriber {
	return &fakeNodeSubscriber{
		nodesJoined: []string{},
		nodesLeft:   []string{},
	}
}

func (s *fakeNodeSubscriber) NotifyJoin(peerID string) {
	s.nodesJoined = append(s.nodesJoined, peerID)
}

func (s *fakeNodeSubscriber) NotifyLeave(peerID string) {
	s.nodesLeft = append(s.nodesLeft, peerID)
}

type peerUpdate struct {
	PeerID string
	Key    string
	Value  string
}

type fakeEventSubscriber struct {
	updates []peerUpdate
}

func NewFakeEventSubscriber() *fakeEventSubscriber {
	return &fakeEventSubscriber{
		updates: []peerUpdate{},
	}
}

func (s *fakeEventSubscriber) NotifyUpdate(peerID string, key string, value string) {
	s.updates = append(s.updates, peerUpdate{
		PeerID: peerID,
		Key:    key,
		Value:  value,
	})
}

func TestPeerMap_UpdateLocal(t *testing.T) {
	pm := newPeerMap("local-peer", "", nil, nil)

	pm.UpdateLocal("foo", "bar")
	e, ok := pm.Lookup("local-peer", "foo")
	assert.True(t, ok)
	assert.Equal(t, "bar", e.Value)
	assert.Equal(t, uint64(1), e.Version)
}

func TestPeerMap_Peers(t *testing.T) {
	pm := newPeerMap("local-peer", "", nil, nil)

	pm.ApplyDeltas(delta{
		"peer-1": peerDelta{
			Addr: "10.26.104.52:1001",
			Deltas: []deltaEntry{
				{Key: "a", Value: "1", Version: 12},
				{Key: "b", Value: "2", Version: 14},
			},
		},
		"peer-3": peerDelta{
			Addr: "10.26.104.52:1003",
			Deltas: []deltaEntry{
				{Key: "c", Value: "3", Version: 15},
			},
		},
		"peer-4": peerDelta{
			Addr: "10.26.104.52:1004",
			Deltas: []deltaEntry{
				{Key: "d", Value: "4", Version: 2},
			},
		},
	})

	// Sort to make comparison easier.
	peers := pm.Peers()
	sort.Strings(peers)

	// Should not include out local peer.
	assert.Equal(t, []string{
		"peer-1", "peer-3", "peer-4",
	}, peers)
}

func TestPeerMap_Digest(t *testing.T) {
	pm := newPeerMap("local-peer", "10.26.104.52:1000", nil, nil)

	pm.ApplyDeltas(delta{
		"peer-1": peerDelta{
			Addr: "10.26.104.52:1001",
			Deltas: []deltaEntry{
				{Key: "a", Value: "1", Version: 12},
				{Key: "b", Value: "2", Version: 14},
			},
		},
		"peer-3": peerDelta{
			Addr: "10.26.104.52:1003",
			Deltas: []deltaEntry{
				{Key: "c", Value: "3", Version: 15},
			},
		},
		"peer-4": peerDelta{
			Addr: "10.26.104.52:1004",
			Deltas: []deltaEntry{
				{Key: "d", Value: "4", Version: 2},
			},
		},
	})

	expected := digest{
		"local-peer": peerDigest{
			Addr:    "10.26.104.52:1000",
			Version: 0,
		},
		"peer-1": peerDigest{
			Addr:    "10.26.104.52:1001",
			Version: 14,
		},
		"peer-3": peerDigest{
			Addr:    "10.26.104.52:1003",
			Version: 15,
		},
		"peer-4": peerDigest{
			Addr:    "10.26.104.52:1004",
			Version: 2,
		},
	}

	actual := pm.Digest()
	assert.Equal(t, expected, actual)
}

func TestPeerMap_Deltas(t *testing.T) {
	pm := newPeerMap("local-peer", "10.26.104.52:1000", nil, nil)

	pm.ApplyDeltas(delta{
		"peer-1": peerDelta{
			Addr: "10.26.104.52:1001",
			Deltas: []deltaEntry{
				{Key: "a", Value: "1", Version: 12},
				{Key: "b", Value: "2", Version: 14},
			},
		},
		"peer-2": peerDelta{
			Addr: "10.26.104.52:1002",
			Deltas: []deltaEntry{
				{Key: "c", Value: "3", Version: 5},
				{Key: "d", Value: "4", Version: 21},
			},
		},
		"peer-3": peerDelta{
			Addr: "10.26.104.52:1003",
			Deltas: []deltaEntry{
				{Key: "e", Value: "5", Version: 13},
				{Key: "f", Value: "6", Version: 15},
			},
		},
		"peer-4": peerDelta{
			Addr: "10.26.104.52:1004",
			Deltas: []deltaEntry{
				{Key: "g", Value: "7", Version: 2},
			},
		},
	})

	actual := pm.Deltas(digest{
		// Version lower than all peer 1's entries.
		"peer-1": peerDigest{
			Addr:    "10.26.104.52:1001",
			Version: 10,
		},
		// Version higher than all peer 2's entries.
		"peer-2": peerDigest{
			Addr:    "10.26.104.52:1002",
			Version: 33,
		},
		// Version higher than half peer 3's entries.
		"peer-3": peerDigest{
			Addr:    "10.26.104.52:1003",
			Version: 14,
		},
		// Not including peer-4 so should be implicitly treated as 0 and
		// include all entries.
	})

	expected := delta{
		"peer-1": peerDelta{
			Addr: "10.26.104.52:1001",
			Deltas: []deltaEntry{
				{Key: "a", Value: "1", Version: 12},
				{Key: "b", Value: "2", Version: 14},
			},
		},
		"peer-3": peerDelta{
			Addr: "10.26.104.52:1003",
			Deltas: []deltaEntry{
				{Key: "f", Value: "6", Version: 15},
			},
		},
		"peer-4": peerDelta{
			Addr: "10.26.104.52:1004",
			Deltas: []deltaEntry{
				{Key: "g", Value: "7", Version: 2},
			},
		},
	}
	assert.Equal(t, expected, actual)
}

func TestPeerMap_ApplyDigest(t *testing.T) {
	pm := newPeerMap("local-peer", "10.26.104.52:1000", nil, nil)

	// Add peers and check the callback is fired.
	pm.ApplyDigest(digest{
		"peer-1": peerDigest{
			Addr:    "10.26.104.52:1001",
			Version: 14,
		},
		"peer-2": peerDigest{
			Addr:    "10.26.104.52:1002",
			Version: 15,
		},
		"peer-3": peerDigest{
			Addr:    "10.26.104.52:1003",
			Version: 2,
		},
	})
	// Nodes could be processed in any order so sort first.
	peers := pm.Peers()
	sort.Strings(peers)
	assert.Equal(t, []string{"peer-1", "peer-2", "peer-3"}, peers)

	addr, ok := pm.Addr("peer-1")
	assert.True(t, ok)
	assert.Equal(t, "10.26.104.52:1001", addr)
}

func TestPeerMap_ApplyDeltasUpdateRemote(t *testing.T) {
	pm := newPeerMap("local-peer", "", nil, nil)

	pm.ApplyDeltas(delta{
		"peer-1": peerDelta{
			Addr: "10.26.104.52:1001",
			Deltas: []deltaEntry{
				{Key: "a", Value: "1", Version: 12},
				{Key: "b", Value: "2", Version: 14},
			},
		},
		"peer-3": peerDelta{
			Addr: "10.26.104.52:1003",
			Deltas: []deltaEntry{
				{Key: "c", Value: "3", Version: 15},
			},
		},
		"peer-4": peerDelta{
			Addr: "10.26.104.52:1004",
			Deltas: []deltaEntry{
				{Key: "d", Value: "4", Version: 2},
			},
		},
	})

	// Check the peers were updated.

	entry, ok := pm.Lookup("peer-1", "a")
	assert.True(t, ok)
	assert.Equal(t, "1", entry.Value)
	assert.Equal(t, uint64(12), entry.Version)

	entry, ok = pm.Lookup("peer-1", "b")
	assert.True(t, ok)
	assert.Equal(t, "2", entry.Value)
	assert.Equal(t, uint64(14), entry.Version)

	entry, ok = pm.Lookup("peer-3", "c")
	assert.True(t, ok)
	assert.Equal(t, "3", entry.Value)
	assert.Equal(t, uint64(15), entry.Version)

	entry, ok = pm.Lookup("peer-4", "d")
	assert.True(t, ok)
	assert.Equal(t, "4", entry.Value)
	assert.Equal(t, uint64(2), entry.Version)
}

func TestPeerMap_ApplyDeltasIgnoreUpdatesAboutLocalPeer(t *testing.T) {
	pm := newPeerMap("local-peer", "10.26.104.52:1000", nil, nil)

	pm.ApplyDeltas(delta{
		"local-peer": peerDelta{
			Addr: "10.26.104.52:1000",
			Deltas: []deltaEntry{
				{Key: "foo", Value: "bar", Version: 12},
			},
		},
	})

	// The entry should not be found as should not have been updated.
	_, ok := pm.Lookup("local-peer", "foo")
	assert.False(t, ok)
}

func TestPeerMap_SubscribeToNodeJoinedFromDigest(t *testing.T) {
	sub := NewFakeNodeSubscriber()
	pm := newPeerMap("local-peer", "10.26.104.52:1000", sub, nil)

	// Add peers and check the callback is fired.
	pm.ApplyDigest(digest{
		"peer-1": peerDigest{
			Addr:    "10.26.104.52:1001",
			Version: 14,
		},
		"peer-2": peerDigest{
			Addr:    "10.26.104.52:1002",
			Version: 15,
		},
		"peer-3": peerDigest{
			Addr:    "10.26.104.52:1003",
			Version: 2,
		},
	})
	// Nodes could be processed in any order so sort first.
	sort.Strings(sub.nodesJoined)
	assert.Equal(t, []string{"peer-1", "peer-2", "peer-3"}, sub.nodesJoined)
}

func TestPeerMap_SubscribeToNodeJoinedFromDelta(t *testing.T) {
	sub := NewFakeNodeSubscriber()
	pm := newPeerMap("local-peer", "10.26.104.52:1000", sub, nil)

	// Add peers and check the callback is fired.
	pm.ApplyDeltas(delta{
		"peer-1": peerDelta{
			Addr: "10.26.104.52:1001",
			Deltas: []deltaEntry{
				{Key: "a", Value: "1", Version: 12},
				{Key: "b", Value: "2", Version: 14},
			},
		},
		"peer-2": peerDelta{
			Addr: "10.26.104.52:1002",
			Deltas: []deltaEntry{
				{Key: "c", Value: "3", Version: 15},
			},
		},
		"peer-3": peerDelta{
			Addr: "10.26.104.52:1003",
			Deltas: []deltaEntry{
				{Key: "d", Value: "4", Version: 2},
			},
		},
	})
	// Nodes could be processed in any order so sort first.
	sort.Strings(sub.nodesJoined)
	assert.Equal(t, []string{"peer-1", "peer-2", "peer-3"}, sub.nodesJoined)
}

func TestPeerMap_SubscribeToPeerUpdated(t *testing.T) {
	sub := NewFakeEventSubscriber()
	pm := newPeerMap("local-peer", "10.26.104.52:1000", nil, sub)

	// Add peers and check the callback is fired.
	pm.ApplyDeltas(delta{
		"peer-1": peerDelta{
			Addr: "10.26.104.52:1001",
			Deltas: []deltaEntry{
				{Key: "a", Value: "1", Version: 12},
				{Key: "b", Value: "2", Version: 14},
			},
		},
		"peer-2": peerDelta{
			Addr: "10.26.104.52:1002",
			Deltas: []deltaEntry{
				{Key: "c", Value: "3", Version: 15},
			},
		},
		"peer-3": peerDelta{
			Addr: "10.26.104.52:1003",
			Deltas: []deltaEntry{
				{Key: "d", Value: "4", Version: 2},
			},
		},
	})
	// Nodes could be processed in any order so sort first.
	sort.Slice(sub.updates, func(i, j int) bool {
		return sub.updates[i].PeerID < sub.updates[j].PeerID
	})
	assert.Equal(t, []peerUpdate{
		peerUpdate{PeerID: "peer-1", Key: "a", Value: "1"},
		peerUpdate{PeerID: "peer-1", Key: "b", Value: "2"},
		peerUpdate{PeerID: "peer-2", Key: "c", Value: "3"},
		peerUpdate{PeerID: "peer-3", Key: "d", Value: "4"},
	}, sub.updates)
}
