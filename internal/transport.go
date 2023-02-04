package internal

import (
	"net"
)

// Packet is used to provide some metadata about incoming packets from peers
// over a packet connection, as well as the packet payload.
type Packet struct {
	// Buf has the raw contents of the packet.
	Buf []byte

	// From has the address of the peer. This is an actual net.Addr so we
	// can expose some concrete details about incoming packets.
	From net.Addr
}

// Transport is an interface for a best-effort packet oriented transport.
type Transport interface {
	// WriteTo is a packet-oriented interface that fires off the given
	// payload to the given address in a connectionless fashion.
	WriteTo(b []byte, addr string) error

	// BindAddr returns the address the transport listener is bound to. Note
	// this may be different from the configured bind addr if the system chooses
	// the addr (such as using a port of 0).
	BindAddr() string

	// Shutdown is called when gossip is shutting down; this gives the
	// transport a chance to clean up any listeners.
	Shutdown() error
}
