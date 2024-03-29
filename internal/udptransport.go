package internal

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
)

const (
	// udpPacketBufSize is used to buffer incoming packets during read
	// operations.
	udpPacketBufSize = 65536
)

// UDPTransport is a Transport implementation using UDP.
type UDPTransport struct {
	udpListener *net.UDPConn
	onPacket    func(p *Packet)
	wg          sync.WaitGroup
	shutdown    int32
	logger      *zap.Logger
}

// NewUDPTransport returns a new UDP transport listening on the given addr.
func NewUDPTransport(bindAddr string, onPacket func(p *Packet), logger *zap.Logger) (Transport, error) {
	udpListener, err := udpListen(bindAddr)
	if err != nil {
		return nil, err
	}

	t := &UDPTransport{
		udpListener: udpListener,
		onPacket:    onPacket,
		wg:          sync.WaitGroup{},
		shutdown:    0,
		logger:      logger,
	}

	t.wg.Add(1)
	go t.udpReadLoop(udpListener)

	return t, nil
}

func (t *UDPTransport) WriteTo(b []byte, addr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	_, err = t.udpListener.WriteTo(b, udpAddr)
	// If we've been shutdown ignore the error.
	if s := atomic.LoadInt32(&t.shutdown); s == 1 {
		return nil
	}
	return err
}

func (t *UDPTransport) BindAddr() string {
	return t.udpListener.LocalAddr().String()
}

func (t *UDPTransport) Shutdown() error {
	// This will avoid log spam about errors when we shut down.
	atomic.StoreInt32(&t.shutdown, 1)

	// Close the listener, which will stop the read loop.
	t.udpListener.Close()

	// Block until all the listener threads have died.
	t.wg.Wait()
	return nil
}

// udpReadLoop is a long running goroutine that accepts incoming UDP packets and
// hands them off to the packet channel.
func (t *UDPTransport) udpReadLoop(lis *net.UDPConn) {
	defer t.wg.Done()
	for {
		// Do a blocking read into a fresh buffer. Grab a time stamp as
		// close as possible to the I/O.
		buf := make([]byte, udpPacketBufSize)
		n, addr, err := lis.ReadFrom(buf)
		if err != nil {
			if s := atomic.LoadInt32(&t.shutdown); s == 1 {
				break
			}

			t.logger.Error("failed to read from transport", zap.Error(err))
			continue
		}

		// Check the length - it needs to have at least one byte to be a
		// proper message.
		if n < 1 {
			t.logger.Error("8eceived packet too small")
			continue
		}

		t.onPacket(&Packet{
			Buf:  buf[:n],
			From: addr,
		})
	}
}

func udpListen(bindAddr string) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp4", bindAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to start UDP listener on %s: %v", bindAddr, err)
	}
	listener, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to start UDP listener on %s: %v", bindAddr, err)
	}
	return listener, nil
}
