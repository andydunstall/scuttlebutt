package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGossip_PropagateUpdate(t *testing.T) {
	cluster := NewCluster()
	defer cluster.Shutdown()

	sub := NewChannelStateSubscriber()

	node1, err := cluster.AddNode("node-1", nil, sub)
	assert.Nil(t, err)
	node2, err := cluster.AddNode("node-2", nil, nil)
	assert.Nil(t, err)

	err = node1.Join(cluster.Seeds())
	assert.Nil(t, err)

	node2.UpdateLocal("foo", "bar")

	update, ok := sub.WaitPeerUpdatedWithTimeout(3 * time.Second)
	assert.True(t, ok)
	assert.Equal(t, "node-2", update.PeerID)
	assert.Equal(t, "foo", update.Key)
	assert.Equal(t, "bar", update.Value)

	val, ok := node1.Lookup("node-2", "foo")
	assert.True(t, ok)
	assert.Equal(t, "bar", val)
}

func TestGossip_PeerDiscovery(t *testing.T) {
	cluster := NewCluster()
	defer cluster.Shutdown()

	sub := NewChannelStateSubscriber()

	node1, err := cluster.AddNode("node-1", sub, nil)
	assert.Nil(t, err)
	_, err = cluster.AddNode("node-2", nil, nil)
	assert.Nil(t, err)
	_, err = cluster.AddNode("node-3", nil, nil)
	assert.Nil(t, err)

	err = node1.Join(cluster.Seeds())
	assert.Nil(t, err)

	// Wait to discovery the other nodes.
	_, ok := sub.WaitPeerJoinedWithTimeout(time.Second)
	assert.True(t, ok)
	_, ok = sub.WaitPeerJoinedWithTimeout(time.Second)
	assert.True(t, ok)
}
