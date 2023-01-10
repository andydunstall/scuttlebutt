package internal

import (
	"go.uber.org/zap/zapcore"
)

type PeerDigest struct {
	Addr    string `json:"addr,omitempty"`
	Version uint64 `json:"version,omitempty"`
}

func (p PeerDigest) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("addr", p.Addr)
	enc.AddUint64("version", p.Version)
	return nil
}

type Digest map[string]PeerDigest

func (d Digest) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for peerID, peerDigest := range d {
		enc.AddObject(peerID, peerDigest)
	}
	return nil
}
