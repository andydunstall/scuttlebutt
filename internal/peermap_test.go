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

func TestPeerMap_PeersIDs(t *testing.T) {
	pm := NewPeerMap("local-peer", "", nil, nil, nil, zap.NewNop())

	pm.ApplyDigest(Digest{
		ID:      "peer-1",
		Addr:    "10.26.104.11:8119",
		Version: 12,
	})
	pm.ApplyDigest(Digest{
		ID:      "peer-3",
		Addr:    "10.26.104.12:8119",
		Version: 15,
	})
	pm.ApplyDigest(Digest{
		ID:      "peer-4",
		Addr:    "10.26.104.13:8119",
		Version: 2,
	})

	allPeers := pm.PeerIDs(true)
	// Sort to make comparison easier.
	sort.Strings(allPeers)
	// Should include the local peer.
	assert.Equal(t, []string{
		"local-peer", "peer-1", "peer-3", "peer-4",
	}, allPeers)

	remotePeers := pm.PeerIDs(false)
	// Sort to make comparison easier.
	sort.Strings(remotePeers)
	// Should not include the local peer.
	assert.Equal(t, []string{
		"peer-1", "peer-3", "peer-4",
	}, remotePeers)
}

// Tests two random peer maps that exchange digests and deltas should have the
// same peer state.
func TestPeerMap_SyncState(t *testing.T) {
	map1 := randomPeerMap(5, 3)
	map2 := randomPeerMap(5, 3)

	assert.False(t, map1.PeersEqual(map2))

	for _, peerID := range map1.PeerIDs(true) {
		map2.ApplyDigest(map1.Digest(peerID))
	}
	for _, peerID := range map2.PeerIDs(true) {
		map1.ApplyDigest(map2.Digest(peerID))
	}

	for _, peerID := range map1.PeerIDs(true) {
		deltas := map1.Deltas(peerID, map2.Version(peerID))
		for _, delta := range deltas {
			map2.ApplyDelta(delta)
		}
	}

	for _, peerID := range map2.PeerIDs(true) {
		deltas := map2.Deltas(peerID, map1.Version(peerID))
		for _, delta := range deltas {
			map1.ApplyDelta(delta)
		}
	}

	assert.True(t, map1.PeersEqual(map2))
}
