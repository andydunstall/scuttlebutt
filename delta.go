package scuttlebutt

type deltaEntry struct {
	Key     string `json:"key,omitempty"`
	Value   string `json:"value,omitempty"`
	Version uint64 `json:"version,omitempty"`
}

type peerDelta struct {
	Addr   string       `json:"addr,omitempty"`
	Deltas []deltaEntry `json:"deltas,omitempty"`
}

type delta map[string]peerDelta
