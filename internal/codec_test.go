package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodec_EncodeAndDecodeDigestRequest(t *testing.T) {
	c := newCodec()

	orig := &Digest{
		"peer-1": PeerDigest{
			Addr:    "10.26.104.52:1001",
			Version: 14,
		},
		"peer-2": PeerDigest{
			Addr:    "10.26.104.52:1003",
			Version: 15,
		},
		"peer-3": PeerDigest{
			Addr:    "10.26.104.52:1004",
			Version: 2,
		},
	}
	b, err := c.Encode(typeDigestRequest, orig)
	assert.Nil(t, err)

	mType, err := c.DecodeType(b)
	assert.Nil(t, err)
	assert.Equal(t, typeDigestRequest, mType)

	var decoded Digest
	assert.Nil(t, c.Decode(b, &decoded))

	assert.Equal(t, orig, &decoded)
}

func TestCodec_EncodeAndDecodeDigestResponse(t *testing.T) {
	c := newCodec()

	orig := &Digest{
		"peer-1": PeerDigest{
			Addr:    "10.26.104.52:1001",
			Version: 14,
		},
		"peer-2": PeerDigest{
			Addr:    "10.26.104.52:1003",
			Version: 15,
		},
		"peer-3": PeerDigest{
			Addr:    "10.26.104.52:1004",
			Version: 2,
		},
	}
	b, err := c.Encode(typeDigestResponse, orig)
	assert.Nil(t, err)

	mType, err := c.DecodeType(b)
	assert.Nil(t, err)
	assert.Equal(t, typeDigestResponse, mType)

	var decoded Digest
	assert.Nil(t, c.Decode(b, &decoded))

	assert.Equal(t, orig, &decoded)
}

func TestCodec_EncodeAndDecodeDelta(t *testing.T) {
	c := newCodec()

	orig := &Delta{
		"peer-1": PeerDelta{
			Addr: "10.26.104.52:1001",
			Deltas: []DeltaEntry{
				{Key: "a", Value: "1", Version: 12},
				{Key: "b", Value: "2", Version: 14},
			},
		},
		"peer-2": PeerDelta{
			Addr: "10.26.104.52:1003",
			Deltas: []DeltaEntry{
				{Key: "c", Value: "3", Version: 15},
			},
		},
	}
	b, err := c.Encode(typeDelta, orig)
	assert.Nil(t, err)

	mType, err := c.DecodeType(b)
	assert.Nil(t, err)
	assert.Equal(t, typeDelta, mType)

	var decoded Delta
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
