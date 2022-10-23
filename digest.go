package scuttlebutt

import (
	"go.uber.org/zap/zapcore"
)

type peerDigest struct {
	Addr    string `json:"addr,omitempty"`
	Version uint64 `json:"version,omitempty"`
}

func (p peerDigest) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("addr", p.Addr)
	enc.AddUint64("version", p.Version)
	return nil
}

type digest map[string]peerDigest

func (d digest) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for peerID, peerDigest := range d {
		enc.AddObject(peerID, peerDigest)
	}
	return nil
}
