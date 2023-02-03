package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodec_EncodeDigest(t *testing.T) {
	digest := Digest{
		ID:      "peer-id",
		Addr:    "10.26.104.56:8123",
		Version: 0xaabbccddeeff,
	}
	b := encodeDigest(digest)
	assert.Equal(t, []byte{
		0x7, 0x70, 0x65, 0x65, 0x72, 0x2d, 0x69, 0x64, // ID
		0x11, 0x31, 0x30, 0x2e, 0x32, 0x36, 0x2e, 0x31, 0x30, 0x34, 0x2e, 0x35, 0x36, 0x3a, 0x38, 0x31, 0x32, 0x33, // Addr
		0x0, 0x0, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, // Version
	}, b)
}

func TestCodec_EncodeDelta(t *testing.T) {
	digest := Delta{
		ID:      "peer-id",
		Key:     "key-123",
		Value:   "value-123",
		Version: 0xaabbccddeeff,
	}
	b := encodeDelta(digest)
	assert.Equal(t, []byte{
		0x7, 0x70, 0x65, 0x65, 0x72, 0x2d, 0x69, 0x64, // ID
		0x7, 0x6b, 0x65, 0x79, 0x2d, 0x31, 0x32, 0x33, // Key
		0x9, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x2d, 0x31, 0x32, 0x33, // Value
		0x0, 0x0, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, // Version
	}, b)
}

func TestCodec_DecodeDigestSync(t *testing.T) {
	sync := []Digest{
		{
			ID:      "peer-1",
			Addr:    "10.26.104.56:8123",
			Version: 0x10,
		},
		{
			ID:      "peer-2",
			Addr:    "10.26.104.82:9833",
			Version: 0x20,
		},
		{
			ID:      "peer-3",
			Addr:    "10.26.104.11:1211",
			Version: 0x30,
		},
	}

	syncEnc := []byte{}
	for _, digest := range sync {
		syncEnc = append(syncEnc, encodeDigest(digest)...)
	}

	assert.Equal(t, decodeDigestSync(syncEnc), sync)
}

func TestCodec_DecodeDeltaSync(t *testing.T) {
	sync := []Delta{
		{
			ID:      "peer-1",
			Key:     "key-1",
			Value:   "value-1",
			Version: 0x10,
		},
		{
			ID:      "peer-2",
			Key:     "key-2",
			Value:   "value-2",
			Version: 0x20,
		},
		{
			ID:      "peer-3",
			Key:     "key-3",
			Value:   "value-3",
			Version: 0x30,
		},
	}

	syncEnc := []byte{}
	for _, delta := range sync {
		syncEnc = append(syncEnc, encodeDelta(delta)...)
	}

	assert.Equal(t, decodeDeltaSync(syncEnc), sync)
}
