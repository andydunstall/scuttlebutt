package scuttlebutt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMockTransport_SendAndRecv(t *testing.T) {
	net := NewMockNetwork()

	t1 := net.NewTransport()
	t2 := net.NewTransport()

	_, err := t1.WriteTo([]byte("foo"), t2.BindAddr())
	assert.Nil(t, err)

	select {
	case packet := <-t2.PacketCh():
		assert.Equal(t, t1.BindAddr(), packet.From.String())
		assert.Equal(t, "foo", string(packet.Buf))
	case <-time.After(time.Second):
		assert.True(t, false)
	}
}
