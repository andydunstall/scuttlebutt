package internal

import (
	"fmt"
	"math/rand"

	"go.uber.org/zap"
)

type Gossiper struct {
	peerMap   *PeerMap
	protocol  *Protocol
	transport Transport
	logger    *zap.Logger
}

func NewGossiper(peerMap *PeerMap, transport Transport, logger *zap.Logger) *Gossiper {
	return &Gossiper{
		peerMap:   peerMap,
		protocol:  NewProtocol(peerMap, logger),
		transport: transport,
		logger:    logger,
	}
}

func (g *Gossiper) PeerIDs(includeLocal bool) []string {
	return g.peerMap.PeerIDs(includeLocal)
}

func (g *Gossiper) Lookup(peerID string, key string) (string, bool) {
	e, ok := g.peerMap.Lookup(peerID, key)
	if !ok {
		return "", false
	}
	return e.Value, true
}

func (g *Gossiper) UpdateLocal(key string, value string) {
	g.peerMap.UpdateLocal(key, value)
}

func (g *Gossiper) BindAddr() string {
	return g.transport.BindAddr()
}

func (g *Gossiper) Gossip(peerID string, addr string) error {
	g.logger.Debug(
		"gossip with peer",
		zap.String("id", peerID),
		zap.String("addr", addr),
	)

	b, err := g.protocol.DigestRequest()
	if err != nil {
		g.logger.Error("failed to get digest reqeust", zap.Error(err))
		return err
	}

	_, err = g.transport.WriteTo(b, addr)
	if err != nil {
		g.logger.Error("failed to write to transport", zap.Error(err))
		return fmt.Errorf("failed to write to transport %s: %v", addr, err)
	}

	return nil
}

func (g *Gossiper) OnMessage(b []byte, fromAddr string) error {
	responses, err := g.protocol.OnMessage(b)
	if err != nil {
		return err
	}
	for _, b := range responses {
		_, err := g.transport.WriteTo(b, fromAddr)
		if err != nil {
			g.logger.Error("failed to write to transport", zap.Error(err))
			return err
		}
	}
	return nil
}

func (g *Gossiper) Seed(seeds []string) {
	g.logger.Debug("seeding gossiper", zap.Strings("seeds", seeds))

	for _, addr := range seeds {
		// Ignore ourselves.
		if addr == g.BindAddr() {
			continue
		}
		g.Gossip("seed", addr)
	}
}

func (g *Gossiper) RandomPeer() (string, string, bool) {
	if len(g.peerMap.PeerIDs(false)) == 0 {
		return "", "", false
	}

	// Scuttlebutt with a random peer (excluding ourselves).
	peerIDs := g.peerMap.PeerIDs(false)
	peerID := peerIDs[rand.Intn(len(peerIDs))]
	addr, ok := g.peerMap.Addr(peerID)
	return peerID, addr, ok
}

func (g *Gossiper) Close() error {
	return g.transport.Shutdown()
}
