package scuttlebutt

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Gossip handles cluster membership using the scuttlebutt protocol.
// This is thread safe.
type Gossip struct {
	peerMap        *peerMap
	protocol       *protocol
	seedCB         func() []string
	gossipInterval time.Duration
	transport      Transport
	done           chan struct{}
	wg             sync.WaitGroup
	logger         *zap.Logger
}

// Create will create a new Gossip using the given configuration.
// This will start listening on the network to allow other nodes to join, though
// will not attempt to join the cluster itself unless contacted by another
// node.
// After this the given configuration should not be modified again.
func Create(conf *Config) (*Gossip, error) {
	g, err := newGossip(conf)
	if err != nil {
		return nil, err
	}
	g.schedule()
	return g, nil
}

// Peers returns the peer IDs of the peers known by this node (excluding
// ourselves).
func (g *Gossip) Peers() []string {
	return g.peerMap.Peers()
}

// Lookup looks up the given key in the known state of the peer with the given
// ID. Since the cluster state is eventually consistent, this isn't guaranteed
// to be up to date with the actual state of the peer, though should converge
// quickly.
func (g *Gossip) Lookup(peerID string, key string) (string, bool) {
	e, ok := g.peerMap.Lookup(peerID, key)
	if !ok {
		return "", false
	}
	return e.Value, true
}

// UpdateLocal updates this nodes state with the given key-value pair. This will
// be propagated to the other nodes in the cluster.
func (g *Gossip) UpdateLocal(key string, value string) {
	g.peerMap.UpdateLocal(key, value)
}

// BindAddr returns the address the transport listener is bound to. Note
// this may be different from the configured bind addr if the system chooses
// the addr (such as using a port of 0).
func (g *Gossip) BindAddr() string {
	return g.transport.BindAddr()
}

// Shutdown closes all background networking and stops gossiping its state to
// the cluster.
func (g *Gossip) Shutdown() error {
	g.logger.Debug("shutdown")

	// Note must close transport first or could block writing to packetCh.
	err := g.transport.Shutdown()
	close(g.done)
	g.wg.Wait()
	return err
}

func newGossip(conf *Config) (*Gossip, error) {
	logger := conf.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	if conf.ID == "" {
		logger.Error("config missing a node ID")
		return nil, fmt.Errorf("config missing a node ID")
	}

	if conf.BindAddr == "" {
		logger.Error("config missing a bind addr")
		return nil, fmt.Errorf("config missing a bind addr")
	}

	// By default gossip every 500ms.
	gossipInterval := conf.GossipInterval
	if gossipInterval == 0 {
		gossipInterval = time.Millisecond * 500
	}

	transport := conf.Transport
	if transport == nil {
		var err error
		transport, err = NewUDPTransport(conf.BindAddr, logger)
		if err != nil {
			logger.Error("failed to start transport", zap.Error(err))
			return nil, err
		}
	}

	logger.Debug("transport started", zap.String("addr", transport.BindAddr()))

	peerMap := newPeerMap(
		conf.ID,
		// Note use transport bind addr not configured bind addr as these
		// may be different if the system assigns the port.
		transport.BindAddr(),
		conf.OnJoin,
		conf.OnLeave,
		conf.OnUpdate,
		logger,
	)

	return &Gossip{
		peerMap:        peerMap,
		protocol:       newProtocol(peerMap, logger),
		seedCB:         conf.SeedCB,
		gossipInterval: gossipInterval,
		transport:      transport,
		done:           make(chan struct{}),
		wg:             sync.WaitGroup{},
		logger:         logger,
	}, nil
}

func (g *Gossip) schedule() {
	g.wg.Add(1)
	go g.gossipLoop()
}

func (g *Gossip) gossipLoop() {
	defer g.wg.Done()

	ticker := time.NewTicker(g.gossipInterval)
	defer ticker.Stop()

	for {
		select {
		case packet := <-g.transport.PacketCh():
			g.onPacket(packet)
		case <-ticker.C:
			g.round()
		case <-g.done:
			return
		}
	}
}

func (g *Gossip) round() {
	if len(g.peerMap.Peers()) == 0 {
		// If we don't know about any other peers in the cluster re-seed.
		g.foo.Seed()
		return
	}

	// Gossip with a random peer.
	peers := g.peerMap.Peers()
	peer := peers[rand.Intn(len(peers))]
	addr, ok := g.peerMap.Addr(peer)
	if !ok {
		return
	}
	g.foo.Gossip(peer, addr)
}

// func (g *Gossip) seed() {
// 	if g.seedCB == nil {
// 		g.logger.Debug("no seed cb; skipping")
// 		return
// 	}
// 
// 	seeds := g.seedCB()
// 
// 	g.logger.Debug("seeding gossiper", zap.Strings("seeds", seeds))
// 
// 	for _, addr := range seeds {
// 		// Ignore ourselves.
// 		if addr == g.BindAddr() {
// 			continue
// 		}
// 		g.gossip("seed", addr)
// 	}
// }

func (g *Gossip) onPacket(p *Packet) {
	f.foo.OnMessage(p.Buf)

	// responses, err := g.protocol.OnMessage(p.Buf)
	// if err != nil {
	// 	return
	// }
	// for _, b := range responses {
	// 	_, err := g.transport.WriteTo(b, p.From.String())
	// 	if err != nil {
	// 		g.logger.Error("failed to write to transport", zap.Error(err))
	// 		return
	// 	}
	// }
}

// func (g *Gossip) gossip(id string, addr string) error {
// 	g.logger.Debug(
// 		"gossip with peer",
// 		zap.String("id", id),
// 		zap.String("addr", addr),
// 	)
// 
// 	b, err := g.protocol.DigestRequest()
// 	if err != nil {
// 		g.logger.Error("failed to get digest reqeust", zap.Error(err))
// 		return err
// 	}
// 
// 	_, err = g.transport.WriteTo(b, addr)
// 	if err != nil {
// 		g.logger.Error("failed to write to transport", zap.Error(err))
// 		return fmt.Errorf("failed to write to transport %s: %v", addr, err)
// 	}
// 
// 	return nil
// }
