package scuttlebutt

type Message struct {
	// Type of message.
	Type string `json:"type,omitempty"`
	// Request is a flag indicating this is a request rather than replying to
	// an earlier request. Used to avoid getting into a request/reply loop.
	Request bool    `json:"request,omitempty"`
	Delta   *Delta  `json:"delta,omitempty"`
	Digest  *Digest `json:"digest,omitempty"`
}