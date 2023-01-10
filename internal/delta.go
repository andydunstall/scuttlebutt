package internal

import (
	"go.uber.org/zap/zapcore"
)

type DeltaEntry struct {
	Key     string `json:"key,omitempty"`
	Value   string `json:"value,omitempty"`
	Version uint64 `json:"version,omitempty"`
}

func (e DeltaEntry) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("key", e.Key)
	enc.AddString("value", e.Value)
	enc.AddUint64("version", e.Version)
	return nil
}

type DeltaEntries []DeltaEntry

func (d DeltaEntries) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	for _, e := range d {
		enc.AppendObject(e)
	}
	return nil
}

type PeerDelta struct {
	Addr   string       `json:"addr,omitempty"`
	Deltas DeltaEntries `json:"deltas,omitempty"`
}

func (p PeerDelta) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("addr", p.Addr)
	enc.AddArray("deltas", p.Deltas)
	return nil
}

type Delta map[string]PeerDelta

func (d Delta) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for peerID, peerDelta := range d {
		enc.AddObject(peerID, peerDelta)
	}
	return nil
}
