package internal

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestPeerMap_UpdateLocal(t *testing.T) {
	pm := NewPeerMap("local:123", nil, nil, nil, zap.NewNop())

	pm.UpdateLocal("foo", "bar")
	e, ok := pm.Lookup("local:123", "foo")
	assert.True(t, ok)
	assert.Equal(t, "bar", e.Value)
	assert.Equal(t, uint64(1), e.Version)
}

func TestPeerMap_PeerAddrs(t *testing.T) {
	pm := NewPeerMap("local:123", nil, nil, nil, zap.NewNop())

	pm.ApplyDigest(Digest{
		Addr:    "10.26.104.11:8119",
		Version: 12,
	})
	pm.ApplyDigest(Digest{
		Addr:    "10.26.104.12:8119",
		Version: 15,
	})
	pm.ApplyDigest(Digest{
		Addr:    "10.26.104.13:8119",
		Version: 2,
	})

	// Add another peer but set its status to dead so it should not be included.
	pm.ApplyDigest(Digest{
		Addr:    "10.26.104.81:4431",
		Version: 6,
	})
	pm.SetStatusDown("10.26.104.81:4431", time.Now())

	allPeers := pm.Addrs(true)
	// Sort to make comparison easier.
	sort.Strings(allPeers)
	// Should include the local peer.
	assert.Equal(t, []string{
		"10.26.104.11:8119", "10.26.104.12:8119", "10.26.104.13:8119", "local:123",
	}, allPeers)

	remotePeers := pm.Addrs(false)
	// Sort to make comparison easier.
	sort.Strings(remotePeers)
	// Should not include the local peer.
	assert.Equal(t, []string{
		"10.26.104.11:8119", "10.26.104.12:8119", "10.26.104.13:8119",
	}, remotePeers)
}

// Tests two random peer maps that exchange digests and deltas should have the
// same peer state.
func TestPeerMap_SyncState(t *testing.T) {
	map1 := randomPeerMap(5, 3)
	map2 := randomPeerMap(5, 3)

	assert.False(t, map1.PeersEqual(map2))

	for _, peerAddr := range map1.Addrs(true) {
		map2.ApplyDigest(map1.Digest(peerAddr))
	}
	for _, peerAddr := range map2.Addrs(true) {
		map1.ApplyDigest(map2.Digest(peerAddr))
	}

	for _, peerAddr := range map1.Addrs(true) {
		deltas := map1.Deltas(peerAddr, map2.Version(peerAddr))
		for _, delta := range deltas {
			map2.ApplyDelta(delta)
		}
	}

	for _, peerAddr := range map2.Addrs(true) {
		deltas := map2.Deltas(peerAddr, map1.Version(peerAddr))
		for _, delta := range deltas {
			map1.ApplyDelta(delta)
		}
	}

	assert.True(t, map1.PeersEqual(map2))
}

func TestPeerMap_SetStatus(t *testing.T) {
	left := []string{}
	onLeave := func(addr string) {
		left = append(left, addr)
	}

	joined := []string{}
	onJoin := func(addr string) {
		joined = append(joined, addr)
	}

	pm := NewPeerMap("local:123", onJoin, onLeave, nil, zap.NewNop())

	// Add a peer and check notified about it joining.
	pm.ApplyDigest(Digest{
		Addr:    "10.26.104.11:8119",
		Version: 12,
	})
	assert.Equal(t, []string{"10.26.104.11:8119"}, joined)

	// Mark the peer as dead and check notified about it leaving.
	pm.SetStatusDown("10.26.104.11:8119", time.Now())
	assert.Equal(t, []string{"10.26.104.11:8119"}, left)

	// Mark the peer as alive and check notified about it re-joining.
	pm.SetStatusUp("10.26.104.11:8119")
	assert.Equal(t, []string{"10.26.104.11:8119", "10.26.104.11:8119"}, joined)
}

func TestPeerMap_RemoveExpiredPeers(t *testing.T) {
	pm := NewPeerMap("local:123", nil, nil, nil, zap.NewNop())

	// Add a down peer who has expired.
	pm.ApplyDigest(Digest{
		Addr:    "10.26.104.11:8119",
		Version: 12,
	})
	pm.SetStatusDown("10.26.104.11:8119", time.Now().Add(time.Minute))

	// Add a down peer who has not expired.
	pm.ApplyDigest(Digest{
		Addr:    "10.26.104.12:6823",
		Version: 21,
	})
	pm.SetStatusDown("10.26.104.12:6823", time.Now().Add(-time.Minute))

	assert.Equal(t, []string{"10.26.104.12:6823"}, pm.RemoveExpiredPeers())
	assert.Equal(t, []string{"10.26.104.11:8119"}, pm.DownPeers())
}
