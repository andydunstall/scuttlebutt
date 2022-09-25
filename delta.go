package scuttlebutt

type DeltaEntry struct {
	Key     string `json:"key,omitempty"`
	Value   string `json:"value,omitempty"`
	Version uint64 `json:"version,omitempty"`
}

type PeerDelta struct {
	Addr   string       `json:"addr,omitempty"`
	Deltas []DeltaEntry `json:"deltas,omitempty"`
}

type Delta map[string]PeerDelta
