package internal

import (
	"go.uber.org/zap/zapcore"
)

type Delta struct {
	ID      string
	Key     string
	Value   string
	Version uint64
}

func (e Delta) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("id", e.ID)
	enc.AddString("key", e.Key)
	enc.AddString("value", e.Value)
	enc.AddUint64("version", e.Version)
	return nil
}
