package scuttlebutt

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/andydunstall/scuttlebutt/internal"
	"go.uber.org/zap"
)

// Scuttlebutt handles cluster membership using the scuttlebutt protocol.
// This is thread safe.
type Scuttlebutt struct {
	peerMap        *internal.PeerMap
	protocol       *internal.Protocol
	seedCB         func() []string
	gossipInterval time.Duration
	transport      internal.Transport
	done           chan struct{}
	wg             sync.WaitGroup
	logger         *zap.Logger
}

// Create will create a new Scuttlebutt using the given configuration.
// This will start listening on the network to allow other nodes to join, though
// will not attempt to join the cluster itself unless contacted by another
// node.
// After this the given configuration should not be modified again.
func Create(id string, addr string, options ...Option) (*Scuttlebutt, error) {
	opts := defaultOptions()
	for _, opt := range options {
		opt(opts)
	}

	g, err := newScuttlebutt(id, addr, opts)
	if err != nil {
		return nil, err
	}
	g.schedule()
	return g, nil
}

// Peers returns the peer IDs of the peers known by this node (excluding
// ourselves).
func (s *Scuttlebutt) Peers() []string {
	return s.peerMap.Peers()
}

// Lookup looks up the given key in the known state of the peer with the given
// ID. Since the cluster state is eventually consistent, this isn't guaranteed
// to be up to date with the actual state of the peer, though should converge
// quickly.
func (s *Scuttlebutt) Lookup(peerID string, key string) (string, bool) {
	e, ok := s.peerMap.Lookup(peerID, key)
	if !ok {
		return "", false
	}
	return e.Value, true
}

// UpdateLocal updates this nodes state with the given key-value pair. This will
// be propagated to the other nodes in the cluster.
func (s *Scuttlebutt) UpdateLocal(key string, value string) {
	s.peerMap.UpdateLocal(key, value)
}

// BindAddr returns the address the transport listener is bound to. Note
// this may be different from the configured bind addr if the system chooses
// the addr (such as using a port of 0).
func (s *Scuttlebutt) BindAddr() string {
	return s.transport.BindAddr()
}

// Shutdown closes all background networking and stops gossiping its state to
// the cluster.
func (s *Scuttlebutt) Shutdown() error {
	s.logger.Debug("shutdown")

	// Note must close transport first or could block writing to packetCh.
	err := s.transport.Shutdown()
	close(s.done)
	s.wg.Wait()
	return err
}

func newScuttlebutt(id string, addr string, opts *Options) (*Scuttlebutt, error) {
	// Limit the size of the node ID size this is encoded with a 1 byte size
	// prefix.
	if len(id) > internal.MaxNodeIDSize {
		return nil, fmt.Errorf("node id too large (cannot exceed 256 bytes)")
	}

	gossip := &Scuttlebutt{
		seedCB:         opts.SeedCB,
		gossipInterval: opts.Interval,
		done:           make(chan struct{}),
		wg:             sync.WaitGroup{},
		logger:         opts.Logger,
	}

	transport, err := internal.NewUDPTransport(addr, gossip.onPacket, opts.Logger)
	if err != nil {
		opts.Logger.Error("failed to start transport", zap.Error(err))
		return nil, err
	}
	gossip.transport = transport

	opts.Logger.Debug("transport started", zap.String("addr", transport.BindAddr()))

	peerMap := internal.NewPeerMap(
		id,
		// Note use transport bind addr not configured bind addr as these
		// may be different if the system assigns the port.
		transport.BindAddr(),
		opts.OnJoin,
		opts.OnLeave,
		opts.OnUpdate,
		opts.Logger,
	)
	gossip.peerMap = peerMap
	gossip.protocol = internal.NewProtocol(peerMap, opts.Logger)

	return gossip, nil
}

func (s *Scuttlebutt) schedule() {
	s.wg.Add(1)
	go s.gossipLoop()
}

func (s *Scuttlebutt) gossipLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.gossipInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.round()
		case <-s.done:
			return
		}
	}
}

func (s *Scuttlebutt) round() {
	if len(s.peerMap.Peers()) == 0 {
		// If we don't know about any other peers in the cluster re-seed.
		s.seed()
		return
	}

	// Scuttlebutt with a random peer.
	peers := s.peerMap.Peers()
	peer := peers[rand.Intn(len(peers))]
	addr, ok := s.peerMap.Addr(peer)
	if !ok {
		return
	}
	s.gossip(peer, addr)
}

func (s *Scuttlebutt) seed() {
	if s.seedCB == nil {
		s.logger.Debug("no seed cb; skipping")
		return
	}

	seeds := s.seedCB()

	s.logger.Debug("seeding gossiper", zap.Strings("seeds", seeds))

	for _, addr := range seeds {
		// Ignore ourselves.
		if addr == s.BindAddr() {
			continue
		}
		s.gossip("seed", addr)
	}
}

func (s *Scuttlebutt) onPacket(p *internal.Packet) {
	responses, err := s.protocol.OnMessage(p.Buf)
	if err != nil {
		return
	}
	for _, b := range responses {
		_, err := s.transport.WriteTo(b, p.From.String())
		if err != nil {
			s.logger.Error("failed to write to transport", zap.Error(err))
			return
		}
	}
}

func (s *Scuttlebutt) gossip(id string, addr string) error {
	s.logger.Debug(
		"gossip with peer",
		zap.String("id", id),
		zap.String("addr", addr),
	)

	b, err := s.protocol.DigestRequest()
	if err != nil {
		s.logger.Error("failed to get digest reqeust", zap.Error(err))
		return err
	}

	_, err = s.transport.WriteTo(b, addr)
	if err != nil {
		s.logger.Error("failed to write to transport", zap.Error(err))
		return fmt.Errorf("failed to write to transport %s: %v", addr, err)
	}

	return nil
}
