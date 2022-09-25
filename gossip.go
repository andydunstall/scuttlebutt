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

type Gossip struct {
	peerMap   *PeerMap
	transport Transport
	done      chan struct{}
	wg        sync.WaitGroup
	logger    *log.Logger
}

// Create will create a new Gossip using the given configuration.
// This will not connect to any other node (see Join) yet, but will start
// all the listeners to allow other nodes to join this memberlist.
// After creating a Gossip, the configuration given should not be
// modified by the user anymore.
func Create(conf *Config) (*Gossip, error) {
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

	g := &Gossip{
		peerMap: NewPeerMap(
			conf.ID,
			// Note use transport bind addr not configured bind addr as these
			// may be different if the system assigns the port.
			transport.BindAddr(),
			conf.NodeSubscriber,
			conf.EventSubscriber,
		),
		transport: transport,
		done:      make(chan struct{}),
		wg:        sync.WaitGroup{},
		logger:    logger,
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

		if err := g.syncNode(addr); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}

func (g *Gossip) Lookup(peerID string, key string) (string, bool) {
	e, ok := g.peerMap.Lookup(peerID, key)
	if !ok {
		return "", false
	}
	return e.Value, true
}

func (g *Gossip) UpdateLocal(key string, value string) {
	g.peerMap.UpdateLocal(key, value)
}

// BindAddr returns the address the transport listener is bound to. Note
// this may be different from the configured bind addr if the system chooses
// the addr (such as using a port of 0).
func (g *Gossip) BindAddr() string {
	return g.transport.BindAddr()
}

func (g *Gossip) Shutdown() error {
	// Note must close transport first or could block writing to packetCh.
	err := g.transport.Shutdown()
	close(g.done)
	g.wg.Wait()
	return err
}

func (g *Gossip) schedule() {
	g.wg.Add(1)
	go g.gossipLoop()
}

func (g *Gossip) syncNode(addr string) error {
	digest := g.peerMap.Digest()
	req := Request{
		Type:   "digest",
		Digest: &digest,
	}
	b, err := json.Marshal(&req)
	if err != nil {
		return fmt.Errorf("failed to encode digest: %v", err)
	}
	_, err = g.transport.WriteTo(b, addr)
	if err != nil {
		return fmt.Errorf("failed to write to peer %s: %v", addr, err)
	}
	return nil
}

func (g *Gossip) gossipLoop() {
	defer g.wg.Done()

	ticker := time.NewTicker(time.Millisecond * 100)
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
	g.syncNode(addr)
}

func (g *Gossip) handleMessage(packet *Packet) {
	var req Request
	if err := json.Unmarshal(packet.Buf, &req); err != nil {
		g.logger.Println("[WARN] scuttlebutt: invalid request received")
		return
	}

	switch req.Type {
	case "digest":
		g.handleDigest(req.Digest, packet.From.String())
	case "delta":
		g.handleDelta(req.Delta)
	default:
		g.logger.Println("[WARN] scuttlebutt: unrecognised request type:", req.Type)
	}
}

func (g *Gossip) handleDigest(digest *Digest, addr string) {
	g.peerMap.ApplyDigest(*digest)

	delta := g.peerMap.Deltas(*digest)
	req := Request{
		Type:  "delta",
		Delta: &delta,
	}
	b, err := json.Marshal(&req)
	if err != nil {
		g.logger.Println("[WARN] scuttlebutt: failed to encode delta:", err)
		return
	}

	if _, err = g.transport.WriteTo(b, addr); err != nil {
		g.logger.Println("[ERR] scuttlebutt: failed to write to transport:", err)
		return
	}
}

func (g *Gossip) handleDelta(delta *Delta) {
	g.peerMap.ApplyDeltas(*delta)
}
