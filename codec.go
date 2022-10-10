package scuttlebutt

import (
	"encoding/json"
	"fmt"
)

type messageType uint8

const (
	typeDigestRequest  messageType = 0
	typeDigestResponse messageType = 1
	typeDelta          messageType = 2
)

type codec struct{}

func newCodec() *codec {
	return &codec{}
}

func (c *codec) Encode(mType messageType, v interface{}) ([]byte, error) {
	switch mType {
	case typeDigestRequest:
		d, ok := v.(*digest)
		if !ok {
			return nil, fmt.Errorf("digest expected")
		}
		return c.encodeDigest("digest-request", d)
	case typeDigestResponse:
		d, ok := v.(*digest)
		if !ok {
			return nil, fmt.Errorf("digest expected")
		}
		return c.encodeDigest("digest-response", d)
	case typeDelta:
		d, ok := v.(*delta)
		if !ok {
			return nil, fmt.Errorf("delta expected")
		}
		return c.encodeDelta(d)
	default:
		return nil, fmt.Errorf("unknown type")
	}
}

func (c *codec) DecodeType(b []byte) (messageType, error) {
	var m message
	if err := json.Unmarshal(b, &m); err != nil {
		return 0, fmt.Errorf("failed to decode message type: %v", err)
	}

	switch m.Type {
	case "digest-request":
		return typeDigestRequest, nil
	case "digest-response":
		return typeDigestResponse, nil
	case "delta":
		return typeDelta, nil
	default:
		return 0, fmt.Errorf("failed to decode message type: unrecognised type: %s", m.Type)
	}
}

func (c *codec) Decode(b []byte, v interface{}) error {
	t, err := c.DecodeType(b)
	if err != nil {
		return err
	}

	var m message
	if err := json.Unmarshal(b, &m); err != nil {
		return fmt.Errorf("failed to decode message: invalid format: %v", err)
	}

	switch t {
	case typeDigestRequest:
		d, ok := v.(*digest)
		if !ok {
			return fmt.Errorf("failed to decode message: digest expected")
		}
		*d = *m.Digest
		return nil
	case typeDigestResponse:
		d, ok := v.(*digest)
		if !ok {
			return fmt.Errorf("failed to decode message: digest expected")
		}
		*d = *m.Digest
		return nil
	case typeDelta:
		d, ok := v.(*delta)
		if !ok {
			return fmt.Errorf("failed to decode message: delta expected")
		}
		*d = *m.Delta
		return nil
	default:
		return fmt.Errorf("failed to decode message: unrecognised type: %d", t)
	}
}

func (c *codec) encodeDigest(mType string, d *digest) ([]byte, error) {
	m := message{
		Type:   mType,
		Digest: d,
	}
	b, err := json.Marshal(&m)
	if err != nil {
		return nil, fmt.Errorf("failed to encode digest: %v", err)
	}
	return b, nil
}

func (c *codec) encodeDelta(d *delta) ([]byte, error) {
	m := message{
		Type:  "delta",
		Delta: d,
	}
	b, err := json.Marshal(&m)
	if err != nil {
		return nil, fmt.Errorf("failed to encode delta: %v", err)
	}
	return b, nil
}
