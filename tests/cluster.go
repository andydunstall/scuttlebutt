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

type ChannelStateSubscriber struct {
	PeerJoinedCh  chan string
	PeerUpdatedCh chan peerUpdate
}

func NewChannelStateSubscriber() *ChannelStateSubscriber {
	return &ChannelStateSubscriber{
		PeerJoinedCh:  make(chan string, 64),
		PeerUpdatedCh: make(chan peerUpdate, 64),
	}
}

func (e *ChannelStateSubscriber) NotifyJoin(peerID string) {
	e.PeerJoinedCh <- peerID
}

func (e *ChannelStateSubscriber) NotifyLeave(peerID string) {}

func (e *ChannelStateSubscriber) NotifyUpdate(peerID string, key string, value string) {
	e.PeerUpdatedCh <- peerUpdate{
		PeerID: peerID,
		Key:    key,
		Value:  value,
	}
}

func (s *ChannelStateSubscriber) WaitPeerUpdatedWithTimeout(t time.Duration) (peerUpdate, bool) {
	select {
	case update := <-s.PeerUpdatedCh:
		return update, true
	case <-time.After(t):
		return peerUpdate{}, false
	}
}

func (s *ChannelStateSubscriber) WaitPeerJoinedWithTimeout(t time.Duration) (string, bool) {
	select {
	case peerID := <-s.PeerJoinedCh:
		return peerID, true
	case <-time.After(t):
		return "", false
	}
}

type Cluster struct {
	nodes map[string]*scuttlebutt.Gossip
}

func NewCluster() *Cluster {
	return &Cluster{
		nodes: make(map[string]*scuttlebutt.Gossip),
	}
}

func (c *Cluster) AddNode(peerID string, nodeSub scuttlebutt.NodeSubscriber, eventSub scuttlebutt.StateSubscriber) (*scuttlebutt.Gossip, error) {
	conf := &scuttlebutt.Config{
		ID: peerID,
		// Use a port of 0 to let the system assigned a free port.
		BindAddr:        "127.0.0.1:0",
		GossipInterval:  time.Millisecond * 100,
		NodeSubscriber:  nodeSub,
		StateSubscriber: eventSub,
	}

	node, err := scuttlebutt.Create(conf)
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
