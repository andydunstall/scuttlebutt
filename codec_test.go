package scuttlebutt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodec_EncodeAndDecodeDigestRequest(t *testing.T) {
	c := newCodec()

	orig := &digest{
		"peer-1": peerDigest{
			Addr:    "10.26.104.52:1001",
			Version: 14,
		},
		"peer-2": peerDigest{
			Addr:    "10.26.104.52:1003",
			Version: 15,
		},
		"peer-3": peerDigest{
			Addr:    "10.26.104.52:1004",
			Version: 2,
		},
	}
	b, err := c.Encode(typeDigestRequest, orig)
	assert.Nil(t, err)

	mType, err := c.DecodeType(b)
	assert.Nil(t, err)
	assert.Equal(t, typeDigestRequest, mType)

	var decoded digest
	assert.Nil(t, c.Decode(b, &decoded))

	assert.Equal(t, orig, &decoded)
}

func TestCodec_EncodeAndDecodeDigestResponse(t *testing.T) {
	c := newCodec()

	orig := &digest{
		"peer-1": peerDigest{
			Addr:    "10.26.104.52:1001",
			Version: 14,
		},
		"peer-2": peerDigest{
			Addr:    "10.26.104.52:1003",
			Version: 15,
		},
		"peer-3": peerDigest{
			Addr:    "10.26.104.52:1004",
			Version: 2,
		},
	}
	b, err := c.Encode(typeDigestResponse, orig)
	assert.Nil(t, err)

	mType, err := c.DecodeType(b)
	assert.Nil(t, err)
	assert.Equal(t, typeDigestResponse, mType)

	var decoded digest
	assert.Nil(t, c.Decode(b, &decoded))

	assert.Equal(t, orig, &decoded)
}

func TestCodec_EncodeAndDecodeDelta(t *testing.T) {
	c := newCodec()

	orig := &delta{
		"peer-1": peerDelta{
			Addr: "10.26.104.52:1001",
			Deltas: []deltaEntry{
				{Key: "a", Value: "1", Version: 12},
				{Key: "b", Value: "2", Version: 14},
			},
		},
		"peer-2": peerDelta{
			Addr: "10.26.104.52:1003",
			Deltas: []deltaEntry{
				{Key: "c", Value: "3", Version: 15},
			},
		},
	}
	b, err := c.Encode(typeDelta, orig)
	assert.Nil(t, err)

	mType, err := c.DecodeType(b)
	assert.Nil(t, err)
	assert.Equal(t, typeDelta, mType)

	var decoded delta
	assert.Nil(t, c.Decode(b, &decoded))

	assert.Equal(t, orig, &decoded)
}

func TestCodec_EncodeUnknownType(t *testing.T) {
	c := newCodec()
	_, err := c.Encode(5, nil)
	assert.NotNil(t, err)
}

func TestCodec_DecodeInvalidEncoding(t *testing.T) {
	c := newCodec()
	err := c.Decode([]byte("invalid"), nil)
	assert.NotNil(t, err)
}
