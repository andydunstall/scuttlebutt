package scuttlebutt

import (
	"go.uber.org/zap/zapcore"
)

type deltaEntry struct {
	Key     string `json:"key,omitempty"`
	Value   string `json:"value,omitempty"`
	Version uint64 `json:"version,omitempty"`
}

func (e deltaEntry) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("key", e.Key)
	enc.AddString("value", e.Value)
	enc.AddUint64("version", e.Version)
	return nil
}

type deltaEntries []deltaEntry

func (d deltaEntries) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	for _, e := range d {
		enc.AppendObject(e)
	}
	return nil
}

type peerDelta struct {
	Addr   string       `json:"addr,omitempty"`
	Deltas deltaEntries `json:"deltas,omitempty"`
}

func (p peerDelta) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("addr", p.Addr)
	enc.AddArray("deltas", p.Deltas)
	return nil
}

type delta map[string]peerDelta

func (d delta) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for peerID, peerDelta := range d {
		enc.AddObject(peerID, peerDelta)
	}
	return nil
}
