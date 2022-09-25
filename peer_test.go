package scuttlebutt

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPeer_UpdateLocalThenLookup(t *testing.T) {
	p := NewPeer("", "")

	p.UpdateLocal("foo", "bar")

	e, ok := p.Lookup("foo")
	assert.True(t, ok)
	assert.Equal(t, "bar", e.Value)
}

func TestPeer_LookupNotFound(t *testing.T) {
	p := NewPeer("", "")
	p.UpdateLocal("foo", "bar")

	_, ok := p.Lookup("car")
	assert.False(t, ok)
}

func TestPeer_UpdateLocalIncrementsVersion(t *testing.T) {
	p := NewPeer("", "")

	// Peer version should start at 0.
	assert.Equal(t, uint64(0), p.Version())

	// Each updates should increase the version.
	for i := 0; i != 10; i++ {
		p.UpdateLocal(strconv.Itoa(i), strconv.Itoa(i))
		assert.Equal(t, uint64(i+1), p.Version())
	}
}

// Tests local updates don't increase the version when the value is unchanged.
func TestPeer_UpdateLocalDiscardsDuplicateUpdate(t *testing.T) {
	p := NewPeer("", "")

	// Peer version should start at 0.
	assert.Equal(t, uint64(0), p.Version())

	// Add an update to increase the version.
	p.UpdateLocal("foo", "bar")
	assert.Equal(t, uint64(1), p.Version())

	// Adding the same update again should not change the version.
	p.UpdateLocal("foo", "bar")
	assert.Equal(t, uint64(1), p.Version())
}

func TestPeer_UpdateRemoteUpdatesValue(t *testing.T) {
	p := NewPeer("", "")

	p.UpdateRemote("foo", "bar", 10)
	e, ok := p.Lookup("foo")
	assert.True(t, ok)
	assert.Equal(t, "bar", e.Value)

	p.UpdateRemote("foo", "car", 20)
	e, ok = p.Lookup("foo")
	assert.True(t, ok)
	assert.Equal(t, "car", e.Value)
}

func TestPeer_UpdateRemoteUpdatesPeerVersion(t *testing.T) {
	p := NewPeer("", "")

	p.UpdateRemote("foo", "bar", 10)
	assert.Equal(t, uint64(10), p.Version())

	p.UpdateRemote("foo", "car", 20)
	assert.Equal(t, uint64(20), p.Version())

	// A smaller peer version, even for an unseen entry, shouldn't update the peer
	// version.
	p.UpdateRemote("boo", "baz", 15)
	assert.Equal(t, uint64(20), p.Version())
}

func TestPeer_UpdateRemoteDiscardsOldVersion(t *testing.T) {
	p := NewPeer("", "")

	// Update and check value updated.
	p.UpdateRemote("foo", "bar", 10)
	e, ok := p.Lookup("foo")
	assert.True(t, ok)
	assert.Equal(t, "bar", e.Value)

	// Update again with a smaller version and check the update is ignored.
	p.UpdateRemote("foo", "car", 5)
	e, ok = p.Lookup("foo")
	assert.True(t, ok)
	assert.Equal(t, "bar", e.Value)
}

func TestPeer_Digest(t *testing.T) {
	p := NewPeer("my-peer", "10.26.104.52:8119")
	assert.Equal(t, PeerDigest{
		Addr:    "10.26.104.52:8119",
		Version: 0,
	}, p.Digest())
}

// Tests deltas returns all entries with a greater version in sorted order.
func TestPeer_Deltas(t *testing.T) {
	p := NewPeer("my-peer", "10.26.104.52:8119")

	p.UpdateLocal("a", "b")
	p.UpdateRemote("c", "d", 3)
	p.UpdateRemote("e", "f", 7)
	p.UpdateRemote("g", "h", 9)

	// Expect a version of 0 to include all entries.
	expectedSince0 := PeerDelta{
		Addr: "10.26.104.52:8119",
		Deltas: []DeltaEntry{
			{Key: "a", Value: "b", Version: 1},
			{Key: "c", Value: "d", Version: 3},
			{Key: "e", Value: "f", Version: 7},
			{Key: "g", Value: "h", Version: 9},
		},
	}
	assert.Equal(t, expectedSince0, p.Deltas(0))

	// A version of 3 should only returns entries with greater versions.
	expectedSince3 := PeerDelta{
		Addr: "10.26.104.52:8119",
		Deltas: []DeltaEntry{
			{Key: "e", Value: "f", Version: 7},
			{Key: "g", Value: "h", Version: 9},
		},
	}
	assert.Equal(t, expectedSince3, p.Deltas(3))

	// A version of 10 should return no entries as we have no entries with
	// a version greater than 10.
	expectedSince10 := PeerDelta{
		Addr:   "10.26.104.52:8119",
		Deltas: []DeltaEntry{},
	}
	assert.Equal(t, expectedSince10, p.Deltas(10))
}
