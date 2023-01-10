package internal

type message struct {
	// Type of message.
	Type   string  `json:"type,omitempty"`
	Delta  *Delta  `json:"delta,omitempty"`
	Digest *Digest `json:"digest,omitempty"`
}
