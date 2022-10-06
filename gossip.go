package scuttlebutt

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	multierror "github.com/hashicorp/go-multierror"
)

// Gossip handles cluster membership using the scuttlebutt protocol.
// This is thread safe.
type Gossip struct {
	peerMap        *peerMap
	gossipInterval time.Duration
	transport      Transport
	done           chan struct{}
	wg             sync.WaitGroup
	logger         *log.Logger
}

// Create will create a new Gossip using the given configuration.
// This will not connect to any other node (see Join) yet, but will start
// all the listeners to allow other nodes to join this memberlist.
// After creating a Gossip, the configuration given should not be
// modified by the user anymore.
func Create(conf *Config) (*Gossip, error) {
	g, err := newGossip(conf)
	if err != nil {
		return nil, err
	}
	g.schedule()
	return g, nil
}

// Join attempts to join the cluster by syncing with the given seed node
// addresses.
//
// Note this does not wait for the sync to complete.
//
// This may be called multiple times, such as if all known nodes leave and so
// the node needs to bootstrap again.
func (g *Gossip) Join(seeds []string) error {
	var errs error
	for _, addr := range seeds {
		// Ignore ourselves.
		if addr == g.BindAddr() {
			continue
		}

		if err := g.sendDigest(addr, true); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs
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
	// Note must close transport first or could block writing to packetCh.
	err := g.transport.Shutdown()
	close(g.done)
	g.wg.Wait()
	return err
}

func newGossip(conf *Config) (*Gossip, error) {
	if conf.ID == "" {
		return nil, fmt.Errorf("config missing a node ID")
	}

	if conf.BindAddr == "" {
		return nil, fmt.Errorf("config missing a bind addr")
	}

	// By default gossip every 500ms.
	gossipInterval := conf.GossipInterval
	if gossipInterval == 0 {
		gossipInterval = time.Millisecond * 500
	}

	logger := conf.Logger
	if logger == nil {
		logger = log.New(os.Stderr, "", log.LstdFlags)
	}

	transport := conf.Transport
	if transport == nil {
		var err error
		transport, err = NewUDPTransport(conf.BindAddr, logger)
		if err != nil {
			return nil, err
		}
	}

	return &Gossip{
		peerMap: newPeerMap(
			conf.ID,
			// Note use transport bind addr not configured bind addr as these
			// may be different if the system assigns the port.
			transport.BindAddr(),
			conf.NodeSubscriber,
			conf.StateSubscriber,
		),
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
			g.handleMessage(packet)
		case <-ticker.C:
			g.gossip()
		case <-g.done:
			return
		}
	}
}

func (g *Gossip) gossip() {
	if len(g.peerMap.Peers()) == 0 {
		return
	}

	peers := g.peerMap.Peers()
	peer := peers[rand.Intn(len(peers))]
	addr, ok := g.peerMap.Addr(peer)
	if !ok {
		return
	}
	g.sendDigest(addr, true)
}

func (g *Gossip) handleMessage(packet *Packet) {
	var m message
	if err := json.Unmarshal(packet.Buf, &m); err != nil {
		g.logger.Println("[WARN] scuttlebutt: invalid message received")
		return
	}

	switch m.Type {
	case "digest":
		g.handleDigest(m.Digest, packet.From.String(), m.Request)
	case "delta":
		g.handleDelta(m.Delta)
	default:
		g.logger.Println("[WARN] scuttlebutt: unrecognised message type:", m.Type)
	}
}

func (g *Gossip) handleDigest(digest *digest, addr string, request bool) {
	// Add any peers we didn't know existed to the peer map.
	g.peerMap.ApplyDigest(*digest)

	delta := g.peerMap.Deltas(*digest)
	// Only send the delta if it is not empty. Note we don't care about sending
	// to prove liveness given we send our own digest immediately anyway.
	if len(delta) > 0 {
		g.sendDelta(addr, delta)
	}

	// Only respond with our own digest if the peers digest was a request.
	// Otherwise we get stuck in a loop sending digests back and forth.
	//
	// Note we respond with a digest even if our digests are the same, since
	// the peer uses the response to check liveness.
	if request {
		g.sendDigest(addr, false)
	}
}

func (g *Gossip) handleDelta(delta *delta) {
	g.peerMap.ApplyDeltas(*delta)
}

func (g *Gossip) sendDigest(addr string, request bool) error {
	digest := g.peerMap.Digest()
	m := message{
		Type:    "digest",
		Request: request,
		Digest:  &digest,
	}
	b, err := json.Marshal(&m)
	if err != nil {
		g.logger.Println("[WARN] scuttlebutt: failed to encode digest:", err)
		return fmt.Errorf("failed to encode digest: %v", err)
	}
	_, err = g.transport.WriteTo(b, addr)
	if err != nil {
		g.logger.Println("[ERR] scuttlebutt: failed to write to transport:", err)
		return fmt.Errorf("failed to write to transport %s: %v", addr, err)
	}
	return nil
}

func (g *Gossip) sendDelta(addr string, delta delta) error {
	m := message{
		Type:    "delta",
		Request: true,
		Delta:   &delta,
	}

	b, err := json.Marshal(&m)
	if err != nil {
		g.logger.Println("[WARN] scuttlebutt: failed to encode delta:", err)
		return fmt.Errorf("failed to encode delta: %v", err)
	}

	if _, err = g.transport.WriteTo(b, addr); err != nil {
		g.logger.Println("[ERR] scuttlebutt: failed to write to transport:", err)
		return fmt.Errorf("failed to write to transport %s: %v", addr, err)
	}
	return nil
}
