package internal

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type peerUpdate struct {
	PeerID string
	Key    string
	Value  string
}

func TestPeerMap_UpdateLocal(t *testing.T) {
	pm := NewPeerMap("local-peer", "", nil, nil, nil, zap.NewNop())

	pm.UpdateLocal("foo", "bar")
	e, ok := pm.Lookup("local-peer", "foo")
	assert.True(t, ok)
	assert.Equal(t, "bar", e.Value)
	assert.Equal(t, uint64(1), e.Version)
}

func TestPeerMap_Peers(t *testing.T) {
	pm := NewPeerMap("local-peer", "", nil, nil, nil, zap.NewNop())

	pm.ApplyDeltas(Delta{
		"peer-1": PeerDelta{
			Addr: "10.26.104.52:1001",
			Deltas: []DeltaEntry{
				{Key: "a", Value: "1", Version: 12},
				{Key: "b", Value: "2", Version: 14},
			},
		},
		"peer-3": PeerDelta{
			Addr: "10.26.104.52:1003",
			Deltas: []DeltaEntry{
				{Key: "c", Value: "3", Version: 15},
			},
		},
		"peer-4": PeerDelta{
			Addr: "10.26.104.52:1004",
			Deltas: []DeltaEntry{
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
	pm := NewPeerMap("local-peer", "10.26.104.52:1000", nil, nil, nil, zap.NewNop())

	pm.ApplyDeltas(Delta{
		"peer-1": PeerDelta{
			Addr: "10.26.104.52:1001",
			Deltas: []DeltaEntry{
				{Key: "a", Value: "1", Version: 12},
				{Key: "b", Value: "2", Version: 14},
			},
		},
		"peer-3": PeerDelta{
			Addr: "10.26.104.52:1003",
			Deltas: []DeltaEntry{
				{Key: "c", Value: "3", Version: 15},
			},
		},
		"peer-4": PeerDelta{
			Addr: "10.26.104.52:1004",
			Deltas: []DeltaEntry{
				{Key: "d", Value: "4", Version: 2},
			},
		},
	})

	expected := Digest{
		"local-peer": PeerDigest{
			Addr:    "10.26.104.52:1000",
			Version: 0,
		},
		"peer-1": PeerDigest{
			Addr:    "10.26.104.52:1001",
			Version: 14,
		},
		"peer-3": PeerDigest{
			Addr:    "10.26.104.52:1003",
			Version: 15,
		},
		"peer-4": PeerDigest{
			Addr:    "10.26.104.52:1004",
			Version: 2,
		},
	}

	actual := pm.Digest()
	assert.Equal(t, expected, actual)
}

func TestPeerMap_Deltas(t *testing.T) {
	pm := NewPeerMap("local-peer", "10.26.104.52:1000", nil, nil, nil, zap.NewNop())

	pm.ApplyDeltas(Delta{
		"peer-1": PeerDelta{
			Addr: "10.26.104.52:1001",
			Deltas: []DeltaEntry{
				{Key: "a", Value: "1", Version: 12},
				{Key: "b", Value: "2", Version: 14},
			},
		},
		"peer-2": PeerDelta{
			Addr: "10.26.104.52:1002",
			Deltas: []DeltaEntry{
				{Key: "c", Value: "3", Version: 5},
				{Key: "d", Value: "4", Version: 21},
			},
		},
		"peer-3": PeerDelta{
			Addr: "10.26.104.52:1003",
			Deltas: []DeltaEntry{
				{Key: "e", Value: "5", Version: 13},
				{Key: "f", Value: "6", Version: 15},
			},
		},
		"peer-4": PeerDelta{
			Addr: "10.26.104.52:1004",
			Deltas: []DeltaEntry{
				{Key: "g", Value: "7", Version: 2},
			},
		},
	})

	actual := pm.Deltas(Digest{
		// Version lower than all peer 1's entries.
		"peer-1": PeerDigest{
			Addr:    "10.26.104.52:1001",
			Version: 10,
		},
		// Version higher than all peer 2's entries.
		"peer-2": PeerDigest{
			Addr:    "10.26.104.52:1002",
			Version: 33,
		},
		// Version higher than half peer 3's entries.
		"peer-3": PeerDigest{
			Addr:    "10.26.104.52:1003",
			Version: 14,
		},
		// Not including peer-4 so should be implicitly treated as 0 and
		// include all entries.
	})

	expected := Delta{
		"peer-1": PeerDelta{
			Addr: "10.26.104.52:1001",
			Deltas: []DeltaEntry{
				{Key: "a", Value: "1", Version: 12},
				{Key: "b", Value: "2", Version: 14},
			},
		},
		"peer-3": PeerDelta{
			Addr: "10.26.104.52:1003",
			Deltas: []DeltaEntry{
				{Key: "f", Value: "6", Version: 15},
			},
		},
		"peer-4": PeerDelta{
			Addr: "10.26.104.52:1004",
			Deltas: []DeltaEntry{
				{Key: "g", Value: "7", Version: 2},
			},
		},
	}
	assert.Equal(t, expected, actual)
}

func TestPeerMap_ApplyDigest(t *testing.T) {
	pm := NewPeerMap("local-peer", "10.26.104.52:1000", nil, nil, nil, zap.NewNop())

	// Add peers and check the callback is fired.
	pm.ApplyDigest(Digest{
		"peer-1": PeerDigest{
			Addr:    "10.26.104.52:1001",
			Version: 14,
		},
		"peer-2": PeerDigest{
			Addr:    "10.26.104.52:1002",
			Version: 15,
		},
		"peer-3": PeerDigest{
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
	pm := NewPeerMap("local-peer", "", nil, nil, nil, zap.NewNop())

	pm.ApplyDeltas(Delta{
		"peer-1": PeerDelta{
			Addr: "10.26.104.52:1001",
			Deltas: []DeltaEntry{
				{Key: "a", Value: "1", Version: 12},
				{Key: "b", Value: "2", Version: 14},
			},
		},
		"peer-3": PeerDelta{
			Addr: "10.26.104.52:1003",
			Deltas: []DeltaEntry{
				{Key: "c", Value: "3", Version: 15},
			},
		},
		"peer-4": PeerDelta{
			Addr: "10.26.104.52:1004",
			Deltas: []DeltaEntry{
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
	pm := NewPeerMap("local-peer", "10.26.104.52:1000", nil, nil, nil, zap.NewNop())

	pm.ApplyDeltas(Delta{
		"local-peer": PeerDelta{
			Addr: "10.26.104.52:1000",
			Deltas: []DeltaEntry{
				{Key: "foo", Value: "bar", Version: 12},
			},
		},
	})

	// The entry should not be found as should not have been updated.
	_, ok := pm.Lookup("local-peer", "foo")
	assert.False(t, ok)
}

func TestPeerMap_SubscribeToNodeJoinedFromDigest(t *testing.T) {
	nodesJoined := []string{}

	onJoin := func(peerID string) {
		nodesJoined = append(nodesJoined, peerID)
	}
	pm := NewPeerMap("local-peer", "10.26.104.52:1000", onJoin, nil, nil, zap.NewNop())

	// Add peers and check the callback is fired.
	pm.ApplyDigest(Digest{
		"peer-1": PeerDigest{
			Addr:    "10.26.104.52:1001",
			Version: 14,
		},
		"peer-2": PeerDigest{
			Addr:    "10.26.104.52:1002",
			Version: 15,
		},
		"peer-3": PeerDigest{
			Addr:    "10.26.104.52:1003",
			Version: 2,
		},
	})
	// Nodes could be processed in any order so sort first.
	sort.Strings(nodesJoined)
	assert.Equal(t, []string{"peer-1", "peer-2", "peer-3"}, nodesJoined)
}

func TestPeerMap_SubscribeToNodeJoinedFromDelta(t *testing.T) {
	nodesJoined := []string{}

	onJoin := func(peerID string) {
		nodesJoined = append(nodesJoined, peerID)
	}
	pm := NewPeerMap("local-peer", "10.26.104.52:1000", onJoin, nil, nil, zap.NewNop())

	// Add peers and check the callback is fired.
	pm.ApplyDeltas(Delta{
		"peer-1": PeerDelta{
			Addr: "10.26.104.52:1001",
			Deltas: []DeltaEntry{
				{Key: "a", Value: "1", Version: 12},
				{Key: "b", Value: "2", Version: 14},
			},
		},
		"peer-2": PeerDelta{
			Addr: "10.26.104.52:1002",
			Deltas: []DeltaEntry{
				{Key: "c", Value: "3", Version: 15},
			},
		},
		"peer-3": PeerDelta{
			Addr: "10.26.104.52:1003",
			Deltas: []DeltaEntry{
				{Key: "d", Value: "4", Version: 2},
			},
		},
	})
	// Nodes could be processed in any order so sort first.
	sort.Strings(nodesJoined)
	assert.Equal(t, []string{"peer-1", "peer-2", "peer-3"}, nodesJoined)
}

func TestPeerMap_SubscribeToPeerUpdated(t *testing.T) {
	peerUpdates := []peerUpdate{}

	onUpdate := func(peerID string, key string, value string) {
		peerUpdates = append(peerUpdates, peerUpdate{
			PeerID: peerID,
			Key:    key,
			Value:  value,
		})
	}
	pm := NewPeerMap("local-peer", "10.26.104.52:1000", nil, nil, onUpdate, zap.NewNop())

	// Add peers and check the callback is fired.
	pm.ApplyDeltas(Delta{
		"peer-1": PeerDelta{
			Addr: "10.26.104.52:1001",
			Deltas: []DeltaEntry{
				{Key: "a", Value: "1", Version: 12},
				{Key: "b", Value: "2", Version: 14},
			},
		},
		"peer-2": PeerDelta{
			Addr: "10.26.104.52:1002",
			Deltas: []DeltaEntry{
				{Key: "c", Value: "3", Version: 15},
			},
		},
		"peer-3": PeerDelta{
			Addr: "10.26.104.52:1003",
			Deltas: []DeltaEntry{
				{Key: "d", Value: "4", Version: 2},
			},
		},
	})
	// Nodes could be processed in any order so sort first.
	sort.Slice(peerUpdates, func(i, j int) bool {
		return peerUpdates[i].PeerID < peerUpdates[j].PeerID
	})
	assert.Equal(t, []peerUpdate{
		peerUpdate{PeerID: "peer-1", Key: "a", Value: "1"},
		peerUpdate{PeerID: "peer-1", Key: "b", Value: "2"},
		peerUpdate{PeerID: "peer-2", Key: "c", Value: "3"},
		peerUpdate{PeerID: "peer-3", Key: "d", Value: "4"},
	}, peerUpdates)
}
