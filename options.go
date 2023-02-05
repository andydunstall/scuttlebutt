package scuttlebutt

import (
	"time"

	"go.uber.org/zap"
)

const (
	DefaultMaxMessageSize      = 512
	DefaultConvictionThreshold = 8.0
	DefaultInterval            = time.Millisecond * 500
)

type Options struct {
	// SeedCB is a callback that returns a list of seed addresses to use to
	// join the cluster. This will be called whenever the node does not know
	// about any other nodes in the cluster. If nil the node will not attempt
	// to seed and must wait for the other nodes to contact it instead.
	SeedCB func() []string

	// OnJoin is invoked when a peer joins the cluster.
	OnJoin func(peerAddr string)

	// OnLeave is invoked when a peer leaves the cluster or is considered
	// inactive.
	OnLeave func(peerAddr string)

	// OnUpdate is invoked when a peers state is updated.
	OnUpdate func(peerAddr string, key string, value string)

	// MaxMessageSize is the maximum allowed UDP payload for gossip messages.
	// If the MTU is known this should be increased to the maximum size. If not
	// set default to 512 bytes.
	MaxMessageSize int

	// ConvictionThreshold is the value if phi in the failure detector to
	// consider a node down. If not set defaults to 8.0.
	ConvictionThreshold float64

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

func WithOnJoin(cb func(peerAddr string)) Option {
	return func(opts *Options) {
		opts.OnJoin = cb
	}
}

func WithOnLeave(cb func(peerAddr string)) Option {
	return func(opts *Options) {
		opts.OnLeave = cb
	}
}

func WithOnUpdate(cb func(peerAddr string, key string, value string)) Option {
	return func(opts *Options) {
		opts.OnUpdate = cb
	}
}

func WithMaxMessageSize(size int) Option {
	return func(opts *Options) {
		opts.MaxMessageSize = size
	}
}

func WithConvictionThreshold(convictionThreshold float64) Option {
	return func(opts *Options) {
		opts.ConvictionThreshold = convictionThreshold
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
	l, _ := zap.NewDevelopment()
	return &Options{
		SeedCB:              nil,
		OnJoin:              nil,
		OnLeave:             nil,
		OnUpdate:            nil,
		MaxMessageSize:      DefaultMaxMessageSize,
		ConvictionThreshold: DefaultConvictionThreshold,
		Interval:            DefaultInterval,
		Logger:              l,
	}
}
