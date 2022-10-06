package scuttlebutt

type peerDigest struct {
	Addr    string `json:"addr,omitempty"`
	Version uint64 `json:"version,omitempty"`
}

type digest map[string]peerDigest
