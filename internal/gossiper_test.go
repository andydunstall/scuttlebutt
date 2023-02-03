package internal

import (
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type fakeTransport struct {
	target *Gossiper
}

func newFakeTransport(target *Gossiper) Transport {
	return &fakeTransport{
		target: target,
	}
}

func (t *fakeTransport) WriteTo(b []byte, addr string) (time.Time, error) {
	t.target.OnMessage(b, "")
	return time.Time{}, nil
}

func (t *fakeTransport) BindAddr() string {
	return ""
}

func (t *fakeTransport) Shutdown() error {
	return nil
}

// Tests multiple gossip round between two gossipers end up with the same known
// state about the cluster.
func TestGossiper_SyncState(t *testing.T) {
	// Test multiple max message sizes.
	for maxMessageSize := 200; maxMessageSize != 1000; maxMessageSize += 50 {
		name := fmt.Sprintf("max-msg-size-%d", maxMessageSize)
		t.Run(name, func(t *testing.T) {
			map1 := randomPeerMap(10, 5)
			map2 := randomPeerMap(10, 5)

			gossiper1 := NewGossiper(
				map1,
				nil,
				maxMessageSize,
				zap.NewNop(),
			)
			gossiper2 := NewGossiper(
				map2,
				nil,
				maxMessageSize,
				zap.NewNop(),
			)
			gossiper1.transport = newFakeTransport(gossiper2)
			gossiper2.transport = newFakeTransport(gossiper1)

			// Keep exchanging messages. Note give plenty of rounds, given the digests
			// can be randomised if they don't fit in the message.
			for i := 0; i != 50; i++ {
				assert.Nil(t, gossiper1.SendDigestRequest(""))
				assert.Nil(t, gossiper2.SendDigestRequest(""))
				if map1.PeersEqual(map2) {
					return
				}
			}
			assert.True(t, map1.PeersEqual(map2))
		})
	}
}

func randomPeerMap(numPeers int, numValues int) *PeerMap {
	localPeerID := fmt.Sprintf("peer-%d", rand.Int())
	peerMap := NewPeerMap(localPeerID, randomAddr(), nil, nil, nil, zap.NewNop())
	for j := 0; j != numValues; j++ {
		peerMap.UpdateLocal(
			fmt.Sprintf("%s-key-%d", localPeerID, rand.Int()),
			fmt.Sprintf("%s-value-%d", localPeerID, rand.Int()),
		)
	}

	for i := 1; i != numValues+1; i++ {
		peerID := fmt.Sprintf("peer-%d", rand.Int())
		peerMap.ApplyDigest(Digest{
			ID:      peerID,
			Addr:    "10.26.104.56:9110",
			Version: 0,
		})
		for j := 0; j != numPeers; j++ {
			peerMap.ApplyDelta(Delta{
				ID:      peerID,
				Key:     fmt.Sprintf("%s-key-%d", peerID, rand.Int()),
				Value:   fmt.Sprintf("%s-value-%d", peerID, rand.Int()),
				Version: uint64(randomUint16()),
			})
		}
	}
	return peerMap
}

func randomAddr() string {
	return fmt.Sprintf("%s:%d", net.IPv4(randomByte(), randomByte(), randomByte(), randomByte()).String(), randomUint16())
}

func randomUint16() uint16 {
	return uint16(rand.Intn(0xffff))
}

func randomByte() byte {
	return byte(rand.Intn(0xff))
}