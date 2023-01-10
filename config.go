package scuttlebutt

import (
	"time"

	"go.uber.org/zap"
)

type Config struct {
	// The ID of this node. This must be unique in the cluster.
	ID string

	// BindAddr is the address this node listens on.
	BindAddr string

	// SeedCB is a callback that returns a list of seed addresses to use to
	// join the cluster. This will be called whenever the node does not know
	// about any other nodes in the cluster. If nil the node will not attempt
	// to seed and must wait for the other nodes to contact it instead.
	SeedCB func() []string

	// Transport used to communicate with other nodes. If unset gossip
	// uses UDPTransport listening on BindAddr.
	Transport Transport

	// GossipInterval is the time between gossip rounds, when the node selects
	// a random peer to sync with.
	// If not set defaults to 500ms.
	GossipInterval time.Duration

	// OnJoin is invoked when a peer joins the cluster.
	OnJoin func(peerID string)

	// OnLeave is invoked when a peer joins the cluster.
	OnLeave func(peerID string)

	// OnUpdate is invoked when a peers state is updated.
	OnUpdate func(peerID string, key string, value string)

	Logger *zap.Logger
}
