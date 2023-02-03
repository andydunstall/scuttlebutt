package internal

import (
	"encoding/binary"
)

type messageType uint8

const (
	MaxNodeIDSize = 0xff

	typeDigestRequest  messageType = 1
	typeDigestResponse messageType = 2
	typeDelta          messageType = 3

	uint8Len  = 1
	uint64Len = 8
)

func encodeUint8(buf []byte, offset int, n uint8) int {
	if len(buf) < offset+uint8Len {
		panic("buf too small; cannot encode uint8")
	}

	buf[offset] = byte(n)
	return offset + uint8Len
}

func encodeUint64(buf []byte, offset int, n uint64) int {
	if len(buf) < offset+uint64Len {
		panic("buf too small; cannot encode uint64")
	}

	binary.BigEndian.PutUint64(buf[offset:offset+uint64Len], n)
	return offset + uint64Len
}

func encodeString(buf []byte, offset int, s string) int {
	if len(buf) < offset+len(s) {
		panic("buf too small; cannot encode bytes")
	}
	if len(s) > 0xff {
		panic("string too large; cannot exceed 256 bytes")
	}

	offset = encodeUint8(buf, offset, uint8(len(s)))
	for i := 0; i != len(s); i++ {
		buf[offset+i] = s[i]
	}
	offset += len(s)
	return offset
}

func encodeDigest(d Digest) []byte {
	payloadLen := uint8Len + len(d.ID) + uint8Len + len(d.Addr) + uint64Len

	b := make([]byte, payloadLen)
	offset := encodeString(b, 0, d.ID)
	offset = encodeString(b, offset, d.Addr)
	encodeUint64(b, offset, d.Version)

	return b
}

func encodeDelta(d Delta) []byte {
	payloadLen := uint8Len + len(d.ID) + uint8Len + len(d.Key) + uint8Len + len(d.Value) + uint64Len

	b := make([]byte, payloadLen)
	offset := encodeString(b, 0, d.ID)
	offset = encodeString(b, offset, d.Key)
	offset = encodeString(b, offset, d.Value)
	encodeUint64(b, offset, d.Version)

	return b
}

func decodeUint8(buf []byte, offset int) (uint8, int) {
	if len(buf) < offset+uint8Len {
		panic("buf too small; cannot decode uint8")
	}

	n := uint8(buf[offset])
	return n, offset + uint8Len
}

func decodeUint64(buf []byte, offset int) (uint64, int) {
	if len(buf) < offset+uint64Len {
		panic("buf too small; cannot decode uint64")
	}

	n := binary.BigEndian.Uint64(buf[offset : offset+uint64Len])
	return n, offset + uint64Len
}

func decodeString(buf []byte, offset int) (string, int) {
	n, offset := decodeUint8(buf, offset)
	if len(buf) < offset+int(n) {
		panic("buf too small; cannot decode string")
	}
	return string(buf[offset : offset+int(n)]), offset + int(n)
}

func decodeDigest(b []byte, offset int) (Digest, int) {
	id, offset := decodeString(b, offset)
	addr, offset := decodeString(b, offset)
	version, offset := decodeUint64(b, offset)
	return Digest{
		ID:      id,
		Addr:    addr,
		Version: version,
	}, offset
}

func decodeDigestSync(b []byte) []Digest {
	sync := []Digest{}
	offset := 0
	for offset < len(b) {
		var digest Digest
		digest, offset = decodeDigest(b, offset)
		sync = append(sync, digest)
	}
	return sync
}

func decodeDelta(b []byte, offset int) (Delta, int) {
	id, offset := decodeString(b, offset)
	key, offset := decodeString(b, offset)
	value, offset := decodeString(b, offset)
	version, offset := decodeUint64(b, offset)
	return Delta{
		ID:      id,
		Key:     key,
		Value:   value,
		Version: version,
	}, offset
}

func decodeDeltaSync(b []byte) []Delta {
	sync := []Delta{}
	offset := 0
	for offset < len(b) {
		var delta Delta
		delta, offset = decodeDelta(b, offset)
		sync = append(sync, delta)
	}
	return sync
}
