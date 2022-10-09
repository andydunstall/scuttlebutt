package cluster

import (
	"context"
	"math/rand"
	"time"

	"github.com/andydunstall/scuttlebutt"
	"github.com/google/uuid"
	multierror "github.com/hashicorp/go-multierror"
)

type Node struct {
	ID       string
	Gossiper *scuttlebutt.Gossip
}

func (n *Node) KnownPeers() int {
	// Add one to include itself.
	return len(n.Gossiper.Peers()) + 1
}

func (n *Node) DiscoveredNode(nodeID string) bool {
	if nodeID == n.ID {
		return true
	}

	for _, id := range n.Gossiper.Peers() {
		if nodeID == id {
			return true
		}
	}
	return false
}

// Cluster manages a local cluster used for testing and evaluation.
type Cluster struct {
	nodes map[string]*Node
}

func NewCluster() *Cluster {
	return &Cluster{
		nodes: make(map[string]*Node),
	}
}

func (c *Cluster) AddNode() (*Node, error) {
	id := uuid.New().String()
	conf := &scuttlebutt.Config{
		ID: id,
		// Use a port of 0 to let the system assigned a free port.
		BindAddr:       "127.0.0.1:0",
		GossipInterval: time.Millisecond * 100,
	}
	gossiper, err := scuttlebutt.Create(conf)
	if err != nil {
		return nil, err
	}
	node := &Node{
		ID:       id,
		Gossiper: gossiper,
	}
	c.nodes[node.ID] = node
	if err = node.Gossiper.Seed(c.seeds(3)); err != nil {
		return nil, err
	}
	return node, nil
}

func (c *Cluster) AddNodes(n int) error {
	var errs error
	for i := 0; i < n; i++ {
		if _, err := c.AddNode(); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}

// WaitForHealthy waits for all nodes to discovery each other.
func (c *Cluster) WaitForHealthy(ctx context.Context) error {
	// TODO(AD) for now just poll - later subscribe to each gossip - and
	// add another subscriber to fire once discovered whole custer
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			healthyNodes := 0
			for _, node := range c.nodes {
				if node.KnownPeers() == len(c.nodes) {
					healthyNodes += 1
				}
			}
			if healthyNodes == len(c.nodes) {
				return nil
			}
		}
	}
}

// WaitToDiscover waits for all nodes to be notified about the node with the
// given ID joining the cluster.
func (c *Cluster) WaitToDiscover(ctx context.Context, nodeID string) error {
	// TODO(AD) for now just poll - later subscribe to each gossip - and
	// add another subscriber to fire once discovered the given node
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			healthyNodes := 0
			for _, node := range c.nodes {
				if node.DiscoveredNode(nodeID) {
					healthyNodes += 1
				}
			}
			if healthyNodes == len(c.nodes) {
				return nil
			}
		}
	}

}

func (c *Cluster) seeds(n int) []string {
	seeds := []string{}
	for _, node := range c.nodes {
		seeds = append(seeds, node.Gossiper.BindAddr())
	}
	rand.Shuffle(len(seeds), func(i, j int) {
		seeds[i], seeds[j] = seeds[j], seeds[i]
	})

	if len(seeds) < n {
		return seeds
	}
	return seeds[:n]
}
