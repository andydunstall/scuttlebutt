package scuttlebutt

type PeerDigest struct {
	Addr    string `json:"addr,omitempty"`
	Version uint64 `json:"version,omitempty"`
}

type Digest map[string]PeerDigest
