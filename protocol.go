package scuttlebutt

import (
	"encoding/json"
	"fmt"
)

type protocol struct {
	peerMap *peerMap
}

func newProtocol(peerMap *peerMap) *protocol {
	return &protocol{
		peerMap: peerMap,
	}
}

func (p *protocol) DigestRequest() ([]byte, error) {
	return p.digestMessage(true)
}

func (p *protocol) OnMessage(b []byte) ([][]byte, error) {
	var m message
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("failed to decode message: %v", err)
	}

	switch m.Type {
	case "digest":
		return p.handleDigest(m.Digest, m.Request)
	case "delta":
		return p.handleDelta(m.Delta)
	default:
		return nil, fmt.Errorf("unrecognised message type: %s", m.Type)
	}
}

func (p *protocol) handleDigest(digest *digest, request bool) ([][]byte, error) {
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

func (p *protocol) handleDelta(delta *delta) ([][]byte, error) {
	p.peerMap.ApplyDeltas(*delta)
	return [][]byte{}, nil
}

func (p *protocol) deltaResponse(delta *delta) ([]byte, error) {
	m := message{
		Type:    "delta",
		Request: true,
		Delta:   delta,
	}

	b, err := json.Marshal(&m)
	if err != nil {
		return nil, fmt.Errorf("failed to encode delta: %v", err)
	}

	return b, nil
}

func (p *protocol) digestResponse() ([]byte, error) {
	return p.digestMessage(false)
}

func (p *protocol) digestMessage(request bool) ([]byte, error) {
	digest := p.peerMap.Digest()
	m := message{
		Type:    "digest",
		Request: request,
		Digest:  &digest,
	}
	b, err := json.Marshal(&m)
	if err != nil {
		return nil, fmt.Errorf("failed to encode digest: %v", err)
	}
	return b, nil
}
