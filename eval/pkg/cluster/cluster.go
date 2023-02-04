package cluster

import (
	"context"
	"math/rand"
	"time"

	"github.com/andydunstall/scuttlebutt"
	multierror "github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)

type Node struct {
	Gossiper *scuttlebutt.Scuttlebutt
}

func (n *Node) KnownPeers() int {
	return len(n.Gossiper.Addrs())
}

func (n *Node) DiscoveredNode(nodeAddr string) bool {
	if nodeAddr == n.Gossiper.BindAddr() {
		return true
	}

	for _, addr := range n.Gossiper.Addrs() {
		if nodeAddr == addr {
			return true
		}
	}
	return false
}

func (n *Node) ReceivedUpdate(nodeAddr string, key string, value string) bool {
	val, ok := n.Gossiper.Lookup(nodeAddr, key)
	if ok && val == value {
		return true
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
	logger, _ := zap.NewDevelopment()

	gossiper, err := scuttlebutt.Create(
		"127.0.0.1:0",
		scuttlebutt.WithSeedCB(func() []string {
			return c.seeds(3)
		}),
		scuttlebutt.WithInterval(time.Millisecond*100),
		scuttlebutt.WithLogger(logger),
	)
	if err != nil {
		return nil, err
	}
	node := &Node{
		Gossiper: gossiper,
	}
	c.nodes[gossiper.BindAddr()] = node
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
// given addr joining the cluster.
func (c *Cluster) WaitToDiscover(ctx context.Context, nodeAddr string) error {
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
				if node.DiscoveredNode(nodeAddr) {
					healthyNodes += 1
				}
			}
			if healthyNodes == len(c.nodes) {
				return nil
			}
		}
	}

}

// WaitToUpdate waits for all nodes to be notified about the given update.
func (c *Cluster) WaitToUpdate(ctx context.Context, nodeAddr string, key string, value string) error {
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
				if node.ReceivedUpdate(nodeAddr, key, value) {
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
