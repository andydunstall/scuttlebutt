package scuttlebutt

import (
	"time"

	"go.uber.org/zap"
)

const (
	DefaultInterval = time.Millisecond * 500
)

type Options struct {
	// SeedCB is a callback that returns a list of seed addresses to use to
	// join the cluster. This will be called whenever the node does not know
	// about any other nodes in the cluster. If nil the node will not attempt
	// to seed and must wait for the other nodes to contact it instead.
	SeedCB func() []string

	// OnJoin is invoked when a peer joins the cluster.
	OnJoin func(peerID string)

	// OnLeave is invoked when a peer joins the cluster.
	OnLeave func(peerID string)

	// OnUpdate is invoked when a peers state is updated.
	OnUpdate func(peerID string, key string, value string)

	// Transport used to communicate with other nodes. If unset gossip
	// uses UDPTransport listening on BindAddr.
	Transport Transport

	// Interval is the time between gossip rounds, when the node selects
	// a random peer to sync with.
	// If not set defaults to 500ms.
	Interval time.Duration

	Logger *zap.Logger
}

type Option func(*Options)

func WithSeedCB(seedCB func() []string) Option {
	return func(opts *Options) {
		opts.SeedCB = seedCB
	}
}

func WithOnJoin(cb func(peerID string)) Option {
	return func(opts *Options) {
		opts.OnJoin = cb
	}
}

func WithOnLeave(cb func(peerID string)) Option {
	return func(opts *Options) {
		opts.OnLeave = cb
	}
}

func WithOnUpdate(cb func(peerID string, key string, value string)) Option {
	return func(opts *Options) {
		opts.OnUpdate = cb
	}
}

func WithTransport(transport Transport) Option {
	return func(opts *Options) {
		opts.Transport = transport
	}
}

func WithInterval(interval time.Duration) Option {
	return func(opts *Options) {
		opts.Interval = interval
	}
}

func WithLogger(logger *zap.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}

func defaultOptions() *Options {
	return &Options{
		SeedCB:    nil,
		OnJoin:    nil,
		OnLeave:   nil,
		OnUpdate:  nil,
		Transport: nil,
		Interval:  DefaultInterval,
		Logger:    zap.NewNop(),
	}
}
