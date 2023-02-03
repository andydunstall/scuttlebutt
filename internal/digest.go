package internal

import (
	"go.uber.org/zap/zapcore"
)

type Digest struct {
	ID      string
	Addr    string
	Version uint64
}

func (p Digest) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("id", p.ID)
	enc.AddString("addr", p.Addr)
	enc.AddUint64("version", p.Version)
	return nil
}
