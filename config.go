package scuttlebutt

import (
	"log"
	"time"
)

type Config struct {
	// The ID of this node. This must be unique in the cluster.
	ID string

	// BindAddr is the address this node listens on.
	BindAddr string

	// Transport used to communicate with other nodes. If unset gossip
	// uses UDPTransport listening on BindAddr.
	Transport Transport

	// GossipInterval is the time between gossip rounds, when the node selects
	// a random peer to sync with.
	// If not set defaults to 500ms.
	GossipInterval time.Duration

	// NodeSubscriber subscribes to events relating to nodes joining and leaving
	// the cluster.
	NodeSubscriber NodeSubscriber

	// StateSubscriber subscribes to events relating to peers state being
	// updated.
	StateSubscriber StateSubscriber

	// Logger is a custom logger. If not set no logs are output to stderr.
	Logger *log.Logger
}
