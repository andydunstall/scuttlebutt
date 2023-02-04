package internal

import (
	"fmt"
	"math/rand"
	"sort"

	"go.uber.org/zap"
)

type peerVersionDelta struct {
	PeerAddr string
	Delta    uint64
	Version  uint64
}

type Gossiper struct {
	peerMap        *PeerMap
	transport      Transport
	maxMessageSize int
	logger         *zap.Logger
}

func NewGossiper(peerMap *PeerMap, transport Transport, maxMessageSize int, logger *zap.Logger) *Gossiper {
	return &Gossiper{
		peerMap:        peerMap,
		transport:      transport,
		maxMessageSize: maxMessageSize,
		logger:         logger,
	}
}

func (g *Gossiper) Addrs(includeLocal bool) []string {
	return g.peerMap.Addrs(includeLocal)
}

func (g *Gossiper) Lookup(addr string, key string) (string, bool) {
	e, ok := g.peerMap.Lookup(addr, key)
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

func (g *Gossiper) SendDigestRequest(addr string) error {
	g.logger.Debug(
		"sending digest request",
		zap.String("addr", addr),
	)

	return g.sendDigestSync(addr, true)
}

func (g *Gossiper) OnMessage(b []byte, fromAddr string) error {
	if len(b) == 0 {
		return fmt.Errorf("invalid message; message is empty")
	}

	switch messageType(b[0]) {
	case typeDigestRequest:
		g.logger.Debug(
			"received digest request",
			zap.String("addr", fromAddr),
		)
		return g.onDigestRequest(decodeDigestSync(b[1:]), fromAddr)
	case typeDigestResponse:
		g.logger.Debug(
			"received digest response",
			zap.String("addr", fromAddr),
		)
		return g.onDigestResponse(decodeDigestSync(b[1:]), fromAddr)
	case typeDelta:
		g.logger.Debug(
			"received delta",
			zap.String("addr", fromAddr),
		)
		return g.onDelta(decodeDeltaSync(b[1:]), fromAddr)
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
		g.SendDigestRequest(addr)
	}
}

func (g *Gossiper) RandomPeer() (string, bool) {
	if len(g.peerMap.Addrs(false)) == 0 {
		return "", false
	}

	// Scuttlebutt with a random peer (excluding ourselves).
	addrs := g.peerMap.Addrs(false)
	return addrs[rand.Intn(len(addrs))], true
}

func (g *Gossiper) Close() error {
	return g.transport.Shutdown()
}

func (g *Gossiper) sendDigestResponse(addr string) error {
	g.logger.Debug(
		"sending digest response",
		zap.String("addr", addr),
	)

	return g.sendDigestSync(addr, false)
}

func (g *Gossiper) sendDigestSync(addr string, request bool) error {
	peerAddrs := g.peerMap.Addrs(true)
	shuffle(peerAddrs)

	messageType := typeDigestRequest
	if !request {
		messageType = typeDigestResponse
	}

	req := []byte{byte(messageType)}
	for _, addr := range peerAddrs {
		digest := g.peerMap.Digest(addr)
		digestEnc := encodeDigest(digest)
		if len(req)+len(digestEnc) > g.maxMessageSize {
			break
		}

		req = append(req, digestEnc...)
	}

	_, err := g.transport.WriteTo(req, addr)
	if err != nil {
		g.logger.Error("failed to write to transport", zap.Error(err))
		return fmt.Errorf("failed to write to transport %s: %v", addr, err)
	}

	return nil
}

func (g *Gossiper) sendDelta(sync []Digest, addr string) error {
	resp := []byte{byte(typeDelta)}
	peerVersionDeltas := g.peerVersionDeltas(sync)
	for _, entry := range peerVersionDeltas {
		deltas := g.peerMap.Deltas(entry.PeerAddr, entry.Version)
		for _, delta := range deltas {
			deltaEnc := encodeDelta(delta)
			if len(sync)+len(deltaEnc) > g.maxMessageSize {
				break
			}

			resp = append(resp, deltaEnc...)
		}
	}

	// Only send the delta response if it is not empty.
	if len(resp) > 1 {
		g.logger.Debug(
			"sending delta",
			zap.String("addr", addr),
		)

		_, err := g.transport.WriteTo(resp, addr)
		if err != nil {
			g.logger.Error("failed to write to transport", zap.Error(err))
			return fmt.Errorf("failed to write to transport %s: %v", addr, err)
		}
	}

	return nil
}

func (g *Gossiper) onDigestRequest(req []Digest, fromAddr string) error {
	return g.onDigestSync(req, fromAddr, true)
}

func (g *Gossiper) onDigestResponse(resp []Digest, fromAddr string) error {
	return g.onDigestSync(resp, fromAddr, false)
}

func (g *Gossiper) onDigestSync(sync []Digest, fromAddr string, sendDigestResponse bool) error {
	for _, digest := range sync {
		g.peerMap.ApplyDigest(digest)
	}

	if err := g.sendDelta(sync, fromAddr); err != nil {
		return err
	}

	if sendDigestResponse {
		return g.sendDigestResponse(fromAddr)
	}

	return nil
}

func (g *Gossiper) onDelta(sync []Delta, fromAddr string) error {
	for _, delta := range sync {
		g.peerMap.ApplyDelta(delta)
	}
	return nil
}

// peerVersionDeltas returns the difference between the versions in each digest
// and the known versions, sorted with the largest delta first. It only includes
// peers where the digest includes a version greater than the local known
// version.
func (g *Gossiper) peerVersionDeltas(sync []Digest) []peerVersionDelta {
	peerVersionDeltas := []peerVersionDelta{}
	for _, digest := range sync {
		knownVersion := g.peerMap.Version(digest.Addr)
		if digest.Version < knownVersion {
			peerVersionDeltas = append(peerVersionDeltas, peerVersionDelta{
				PeerAddr: digest.Addr,
				Delta:    knownVersion - digest.Version,
				Version:  digest.Version,
			})
		}
	}
	sort.Slice(peerVersionDeltas, func(i, j int) bool {
		return peerVersionDeltas[i].Delta > peerVersionDeltas[j].Delta
	})
	return peerVersionDeltas
}
