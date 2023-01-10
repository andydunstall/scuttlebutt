package internal

import (
	"fmt"

	"go.uber.org/zap"
)

type Protocol struct {
	peerMap *PeerMap
	codec   *codec
	logger  *zap.Logger
}

func NewProtocol(peerMap *PeerMap, logger *zap.Logger) *Protocol {
	return &Protocol{
		peerMap: peerMap,
		codec:   newCodec(),
		logger:  logger,
	}
}

func (p *Protocol) DigestRequest() ([]byte, error) {
	digest := p.peerMap.Digest()
	return p.codec.Encode(typeDigestRequest, &digest)
}

func (p *Protocol) OnMessage(b []byte) ([][]byte, error) {
	mType, err := p.codec.DecodeType(b)
	if err != nil {
		return nil, err
	}

	switch mType {
	case typeDigestRequest:
		var d Digest
		if err = p.codec.Decode(b, &d); err != nil {
			p.logger.Error("invalid digest request", zap.Error(err))
			return nil, err
		}
		return p.handleDigest(&d, true)
	case typeDigestResponse:
		var d Digest
		if err = p.codec.Decode(b, &d); err != nil {
			p.logger.Error("invalid digest response", zap.Error(err))
			return nil, err
		}
		return p.handleDigest(&d, false)
	case typeDelta:
		var d Delta
		if err = p.codec.Decode(b, &d); err != nil {
			p.logger.Error("invalid delta response", zap.Error(err))
			return nil, err
		}
		return p.handleDelta(&d)
	default:
		p.logger.Error("unrecognised message type", zap.Uint8("type", uint8(mType)))
		return nil, fmt.Errorf("unrecognised message type: %v", mType)
	}
}

func (p *Protocol) handleDigest(digest *Digest, request bool) ([][]byte, error) {
	responses := [][]byte{}

	// Add any peers we didn't know existed to the peer map.
	p.peerMap.ApplyDigest(*digest)

	delta := p.peerMap.Deltas(*digest)
	// Only send the delta if it is not empty. Note we don't care about sending
	// to prove liveness given we send our own digest immediately anyway.
	if len(delta) > 0 {
		resp, err := p.deltaResponse(&delta)
		if err != nil {
			return nil, err
		}
		if resp != nil {
			responses = append(responses, resp)
		}
	}

	// Only respond with our own digest if the peers digest was a request.
	// Otherwise we get stuck in a loop sending digests back and forth.
	//
	// Note we respond with a digest even if our digests are the same, since
	// the peer uses the response to check liveness.
	if request {
		resp, err := p.digestResponse()
		if err != nil {
			return nil, err
		}
		if resp != nil {
			responses = append(responses, resp)
		}
	}

	return responses, nil
}

func (p *Protocol) handleDelta(delta *Delta) ([][]byte, error) {
	p.peerMap.ApplyDeltas(*delta)
	return [][]byte{}, nil
}

func (p *Protocol) deltaResponse(delta *Delta) ([]byte, error) {
	return p.codec.Encode(typeDelta, delta)
}

func (p *Protocol) digestResponse() ([]byte, error) {
	digest := p.peerMap.Digest()
	return p.codec.Encode(typeDigestResponse, &digest)
}
