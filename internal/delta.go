package internal

import (
	"go.uber.org/zap/zapcore"
)

type Delta struct {
	Addr    string
	Key     string
	Value   string
	Version uint64
}

func (e Delta) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("addr", e.Addr)
	enc.AddString("key", e.Key)
	enc.AddString("value", e.Value)
	enc.AddUint64("version", e.Version)
	return nil
}
