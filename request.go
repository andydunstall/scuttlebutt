package scuttlebutt

type Request struct {
	Type   string  `json:"type,omitempty"`
	Delta  *Delta  `json:"delta,omitempty"`
	Digest *Digest `json:"digest,omitempty"`
}
