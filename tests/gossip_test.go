package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGossip_PropagateUpdate(t *testing.T) {
	cluster := NewCluster()
	defer cluster.Shutdown()

	sub := NewNodeSubscriber()

	node1, err := cluster.AddNode(sub)
	assert.Nil(t, err)
	node2, err := cluster.AddNode(nil)
	assert.Nil(t, err)

	node2.UpdateLocal("foo", "bar")

	update, ok := sub.WaitPeerUpdatedWithTimeout(3 * time.Second)
	assert.True(t, ok)
	assert.Equal(t, node2.BindAddr(), update.Addr)
	assert.Equal(t, "foo", update.Key)
	assert.Equal(t, "bar", update.Value)

	val, ok := node1.Lookup(node2.BindAddr(), "foo")
	assert.True(t, ok)
	assert.Equal(t, "bar", val)
}

func TestGossip_PeerDiscovery(t *testing.T) {
	cluster := NewCluster()
	defer cluster.Shutdown()

	sub := NewNodeSubscriber()

	_, err := cluster.AddNode(sub)
	assert.Nil(t, err)
	_, err = cluster.AddNode(nil)
	assert.Nil(t, err)
	_, err = cluster.AddNode(nil)
	assert.Nil(t, err)

	// Wait to discovery the other nodes.
	_, ok := sub.WaitPeerJoinedWithTimeout(time.Second)
	assert.True(t, ok)
	_, ok = sub.WaitPeerJoinedWithTimeout(time.Second)
	assert.True(t, ok)
}

func TestGossip_DetectFailedNode(t *testing.T) {
	cluster := NewCluster()
	defer cluster.Shutdown()

	sub := NewNodeSubscriber()

	_, err := cluster.AddNode(sub)
	assert.Nil(t, err)
	_, err = cluster.AddNode(nil)
	assert.Nil(t, err)
	node3, err := cluster.AddNode(nil)
	assert.Nil(t, err)

	// Wait to discovery the other nodes.
	_, ok := sub.WaitPeerJoinedWithTimeout(time.Second)
	assert.True(t, ok)
	_, ok = sub.WaitPeerJoinedWithTimeout(time.Second)
	assert.True(t, ok)

	node3.Shutdown()
	cluster.RemoveNode(node3.BindAddr())

	_, ok = sub.WaitPeerLeftWithTimeout(time.Second * 10)
	assert.True(t, ok)
}
