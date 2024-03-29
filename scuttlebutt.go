package scuttlebutt

import (
	"sync"
	"time"

	"github.com/andydunstall/scuttlebutt/internal"
	"go.uber.org/zap"
)

const (
	failureDetectorSampleSize = 1000
)

// Scuttlebutt handles cluster membership using the scuttlebutt protocol.
// This is thread safe.
type Scuttlebutt struct {
	gossiper       *internal.Gossiper
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
func Create(addr string, options ...Option) (*Scuttlebutt, error) {
	opts := defaultOptions()
	for _, opt := range options {
		opt(opts)
	}

	g, err := newScuttlebutt(addr, opts)
	if err != nil {
		return nil, err
	}
	g.schedule()
	return g, nil
}

// Addrs returns the addresses of the peers known by this node (including
// ourselves).
func (s *Scuttlebutt) Addrs() []string {
	return s.gossiper.Addrs(true)
}

// Lookup looks up the given key in the known state of the peer with the
// address. Since the cluster state is eventually consistent, this isn't
// guaranteed to be up to date with the actual state of the peer, though should
// converge quickly.
func (s *Scuttlebutt) Lookup(addr string, key string) (string, bool) {
	return s.gossiper.Lookup(addr, key)
}

// UpdateLocal updates this nodes state with the given key-value pair. This will
// be propagated to the other nodes in the cluster.
func (s *Scuttlebutt) UpdateLocal(key string, value string) {
	s.gossiper.UpdateLocal(key, value)
}

// BindAddr returns the address the transport listener is bound to. Note
// this may be different from the configured bind addr if the system chooses
// the addr (such as using a port of 0).
func (s *Scuttlebutt) BindAddr() string {
	return s.gossiper.BindAddr()
}

// Shutdown closes all background networking and stops gossiping its state to
// the cluster.
func (s *Scuttlebutt) Shutdown() error {
	s.logger.Debug("shutdown")

	// Note must close transport first or could block writing to packetCh.
	err := s.gossiper.Close()
	close(s.done)
	s.wg.Wait()
	return err
}

func newScuttlebutt(addr string, opts *Options) (*Scuttlebutt, error) {
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
		// Note use transport bind addr not configured bind addr as these
		// may be different if the system assigns the port.
		transport.BindAddr(),
		opts.OnJoin,
		opts.OnLeave,
		opts.OnUpdate,
		opts.Logger,
	)
	gossip.gossiper = internal.NewGossiper(
		peerMap,
		transport,
		internal.NewFailureDetector(
			uint64(opts.Interval.Nanoseconds()),
			failureDetectorSampleSize,
			opts.ConvictionThreshold,
		),
		opts.MaxMessageSize,
		opts.Logger,
	)

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
	s.gossipToUpPeer()
	s.gossiper.CheckLiveness()
	s.gossipToDownPeer()
}

func (s *Scuttlebutt) gossipToUpPeer() {
	addr, ok := s.gossiper.RandomUpPeer()
	if !ok {
		// If we don't know about any other peers in the cluster re-seed.
		s.seed()
		return
	}
	s.gossiper.SendDigestRequest(addr)
}

func (s *Scuttlebutt) gossipToDownPeer() {
	addr, ok := s.gossiper.RandomDownPeer()
	if !ok {
		return
	}
	s.gossiper.SendDigestRequest(addr)
}

func (s *Scuttlebutt) seed() {
	if s.seedCB == nil {
		s.logger.Debug("no seed cb; skipping")
		return
	}

	s.gossiper.Seed(s.seedCB())
}

func (s *Scuttlebutt) onPacket(p *internal.Packet) {
	s.gossiper.OnMessage(p.Buf, p.From.String())
}
