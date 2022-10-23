package scuttlebutt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type fakePeer struct {
	ID       string
	Addr     string
	PeerMap  *peerMap
	Protocol *protocol
}

func newFakePeer(id string, addr string) *fakePeer {
	peerMap := newPeerMap(id, addr, nil, nil, zap.NewNop())
	return &fakePeer{
		ID:       id,
		Addr:     addr,
		PeerMap:  peerMap,
		Protocol: newProtocol(peerMap, zap.NewNop()),
	}
}

// Tests a round of gossip between two peers
func TestProtocol_PeerGossip(t *testing.T) {
	// Create two peers and add state to each.
	p1 := newFakePeer("peer-1", "10.26.104.24:8119")
	p1.PeerMap.UpdateLocal("foo", "bar")
	p2 := newFakePeer("peer-2", "10.26.104.83:9322")
	p2.PeerMap.UpdateLocal("baz", "car")

	// Complete a round of gossip between each peer.

	// Send an initial digest to start the sync.
	b, err := p1.Protocol.DigestRequest()
	assert.Nil(t, err)
	responsesP2, err := p2.Protocol.OnMessage(b)
	assert.Nil(t, err)
	// Expect the receiver to respond with both a digest of its state (so the
	// sender can calculate a delta response) and a delta with the entries
	// the sender is missing.
	assert.Equal(t, 2, len(responsesP2))

	// Forward the responses to the sender, which should respond with its own
	// delta to get the receiver up to date.
	responsesP1 := [][]byte{}
	for _, resp := range responsesP2 {
		responses, err := p1.Protocol.OnMessage(resp)
		assert.Nil(t, err)
		responsesP1 = append(responsesP1, responses...)
	}
	assert.Equal(t, 1, len(responsesP1))

	// Forward the responses. The round is done so expect no more messages.
	responsesP2 = [][]byte{}
	for _, resp := range responsesP1 {
		responses, err := p2.Protocol.OnMessage(resp)
		assert.Nil(t, err)
		responsesP1 = append(responsesP2, responses...)
	}
	assert.Equal(t, 0, len(responsesP2))

	// Check the two peers state is now equal.
	assert.True(t, p1.PeerMap.PeersEqual(p2.PeerMap))
}

// Tests a round of gossip between two peers syncs their states.
func TestProtocol_GossipRoundSyncsPeers(t *testing.T) {
	// Create two peers and add state to each.
	p1 := newFakePeer("peer-1", "10.26.104.24:8119")
	p1.PeerMap.UpdateLocal("foo", "bar")
	p2 := newFakePeer("peer-2", "10.26.104.83:9322")
	p2.PeerMap.UpdateLocal("baz", "car")

	// Complete a round of gossip between each peer.

	// Send an initial digest to start the sync.
	b, err := p1.Protocol.DigestRequest()
	assert.Nil(t, err)
	responsesP2, err := p2.Protocol.OnMessage(b)
	assert.Nil(t, err)
	// Expect the receiver to respond with both a digest of its state (so the
	// sender can calculate a delta response) and a delta with the entries
	// the sender is missing.
	assert.Equal(t, 2, len(responsesP2))

	// Forward the responses to the sender, which should respond with its own
	// delta to get the receiver up to date.
	responsesP1 := [][]byte{}
	for _, resp := range responsesP2 {
		responses, err := p1.Protocol.OnMessage(resp)
		assert.Nil(t, err)
		responsesP1 = append(responsesP1, responses...)
	}
	assert.Equal(t, 1, len(responsesP1))

	// Forward the responses. The round is done so expect no more messages.
	responsesP2 = [][]byte{}
	for _, resp := range responsesP1 {
		responses, err := p2.Protocol.OnMessage(resp)
		assert.Nil(t, err)
		responsesP1 = append(responsesP2, responses...)
	}
	assert.Equal(t, 0, len(responsesP2))

	// Check the two peers state is now equal.
	assert.True(t, p1.PeerMap.PeersEqual(p2.PeerMap))
}

func TestProtocol_DoesntSendEmptyDelta(t *testing.T) {
	// Create two peers with no state.
	p1 := newFakePeer("peer-1", "10.26.104.24:8119")
	p2 := newFakePeer("peer-2", "10.26.104.83:9322")

	// Send an initial digest to start the sync.
	b, err := p1.Protocol.DigestRequest()
	assert.Nil(t, err)
	responsesP2, err := p2.Protocol.OnMessage(b)
	assert.Nil(t, err)
	// Expect the receiver to respond with only a digest, as the delta should
	// be empty.
	assert.Equal(t, 1, len(responsesP2))

	// Forward the responses to the sender, which should not respond as the
	// delta should be empty.
	responsesP1 := [][]byte{}
	for _, resp := range responsesP2 {
		responses, err := p1.Protocol.OnMessage(resp)
		assert.Nil(t, err)
		responsesP1 = append(responsesP1, responses...)
	}
	assert.Equal(t, 0, len(responsesP1))
}
