package scuttlebutt

type message struct {
	// Type of message.
	Type   string  `json:"type,omitempty"`
	Delta  *delta  `json:"delta,omitempty"`
	Digest *digest `json:"digest,omitempty"`
}
