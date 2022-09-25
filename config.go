package scuttlebutt

import (
	"log"
)

type Config struct {
	// The ID of this node. This must be unique in the cluster.
	ID string

	// BindAddr is the address this node listens on.
	BindAddr string

	// Transport used to communicate with other nodes. If unset gossip
	// uses UDPTransport listening on BindAddr.
	Transport Transport

	// NodeSubscriber subscribes to events relating to nodes joining and leaving
	// the cluster.
	NodeSubscriber NodeSubscriber

	// EventSubscriber subscribes to events relating to peers state being
	// updated.
	EventSubscriber EventSubscriber

	// Logger is a custom logger. If not set no logs are output to stderr.
	Logger *log.Logger
}
