package tests

import (
	"time"

	"github.com/andydunstall/scuttlebutt"
	multierror "github.com/hashicorp/go-multierror"
)

type peerUpdate struct {
	PeerID string
	Key    string
	Value  string
}

type NodeSubscriber struct {
	PeerJoinedCh  chan string
	PeerLeftCh    chan string
	PeerUpdatedCh chan peerUpdate
}

func NewNodeSubscriber() *NodeSubscriber {
	return &NodeSubscriber{
		PeerJoinedCh:  make(chan string, 64),
		PeerLeftCh:    make(chan string, 64),
		PeerUpdatedCh: make(chan peerUpdate, 64),
	}
}

func (e *NodeSubscriber) OnJoin(peerID string) {
	e.PeerJoinedCh <- peerID
}

func (e *NodeSubscriber) OnLeave(peerID string) {}

func (e *NodeSubscriber) OnUpdate(peerID string, key string, value string) {
	e.PeerUpdatedCh <- peerUpdate{
		PeerID: peerID,
		Key:    key,
		Value:  value,
	}
}

func (s *NodeSubscriber) WaitPeerUpdatedWithTimeout(t time.Duration) (peerUpdate, bool) {
	select {
	case update := <-s.PeerUpdatedCh:
		return update, true
	case <-time.After(t):
		return peerUpdate{}, false
	}
}

func (s *NodeSubscriber) WaitPeerJoinedWithTimeout(t time.Duration) (string, bool) {
	select {
	case peerID := <-s.PeerJoinedCh:
		return peerID, true
	case <-time.After(t):
		return "", false
	}
}

type Cluster struct {
	nodes map[string]*scuttlebutt.Scuttlebutt
}

func NewCluster() *Cluster {
	return &Cluster{
		nodes: make(map[string]*scuttlebutt.Scuttlebutt),
	}
}

func (c *Cluster) AddNode(peerID string, nodeSub *NodeSubscriber) (*scuttlebutt.Scuttlebutt, error) {
	opts := []scuttlebutt.Option{
		scuttlebutt.WithSeedCB(func() []string {
			return c.Seeds()
		}),
		scuttlebutt.WithInterval(time.Millisecond * 100),
	}
	if nodeSub != nil {
		opts = append(opts, scuttlebutt.WithOnJoin(nodeSub.OnJoin))
		opts = append(opts, scuttlebutt.WithOnLeave(nodeSub.OnLeave))
		opts = append(opts, scuttlebutt.WithOnUpdate(nodeSub.OnUpdate))
	}

	node, err := scuttlebutt.Create(peerID, "127.0.0.1:0", opts...)
	if err != nil {
		return nil, err
	}
	c.nodes[peerID] = node
	return node, nil
}

func (c *Cluster) Shutdown() error {
	var errs error
	for _, node := range c.nodes {
		if err := node.Shutdown(); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}

func (c *Cluster) Seeds() []string {
	seeds := []string{}
	for _, node := range c.nodes {
		seeds = append(seeds, node.BindAddr())
	}
	return seeds
}
