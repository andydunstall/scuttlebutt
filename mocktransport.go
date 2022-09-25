package scuttlebutt

import (
	"fmt"
	"net"
	"time"
)

// MockNetwork is used as a factory that produces MockTransport instances which
// are uniquely addressed and wired up to talk to each other.
type MockNetwork struct {
	transports map[string]*MockTransport
	nextPort   int
}

func NewMockNetwork() *MockNetwork {
	return &MockNetwork{
		transports: make(map[string]*MockTransport),
		nextPort:   20000,
	}
}

func (n *MockNetwork) NewTransport() *MockTransport {
	addr := fmt.Sprintf("127.0.0.1:%d", n.nextPort)
	n.nextPort++
	transport := &MockTransport{
		net:      n,
		bindAddr: addr,
		// Add a small buffer so sending doesn't block.
		packetCh: make(chan *Packet, 64),
	}
	n.transports[addr] = transport
	return transport
}

// MockAddress is a wrapper which adds the net.Addr interface to our mock
// address scheme.
type MockAddress struct {
	addr string
}

func (a *MockAddress) Network() string {
	return "mock"
}

func (a *MockAddress) String() string {
	return a.addr
}

type MockTransport struct {
	net      *MockNetwork
	packetCh chan *Packet
	bindAddr string
}

func (t *MockTransport) WriteTo(b []byte, addr string) (time.Time, error) {
	dest, err := t.getPeer(addr)
	if err != nil {
		return time.Time{}, err
	}

	now := time.Now()
	dest.packetCh <- &Packet{
		Buf:       b,
		From:      t.from(),
		Timestamp: now,
	}
	return now, nil
}

func (t *MockTransport) PacketCh() <-chan *Packet {
	return t.packetCh
}

func (t *MockTransport) BindAddr() string {
	return t.bindAddr
}

func (t *MockTransport) Shutdown() error {
	return nil
}

func (t *MockTransport) getPeer(addr string) (*MockTransport, error) {
	dest, ok := t.net.transports[addr]
	if !ok {
		return nil, fmt.Errorf("No route to %s", addr)
	}
	return dest, nil
}

func (t *MockTransport) from() net.Addr {
	return &MockAddress{
		addr: t.bindAddr,
	}
}
