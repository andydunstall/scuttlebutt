package scuttlebutt

import (
	"sort"
)

type peerEntry struct {
	Version uint64
	Value   string
}

// Peer represents the state of a peer.
type Peer struct {
	peerID string
	addr   string
	// version is the highest version of all the peers entries. This is used to
	// compare versions between nodes to check for missing updates.
	version uint64
	// entries contains the peer state to be propagated around the cluster.
	entries map[string]peerEntry
}

// NewPeer returns a new peer with the given ID, with a version of 0 to indicate
// this hasn't had any updates.
func NewPeer(peerID string, addr string) *Peer {
	return &Peer{
		peerID:  peerID,
		addr:    addr,
		version: 0,
		entries: make(map[string]peerEntry),
	}
}

func (p *Peer) Addr() string {
	return p.addr
}

func (p *Peer) Version() uint64 {
	return p.version
}

func (p *Peer) Lookup(key string) (peerEntry, bool) {
	if entry, ok := p.entries[key]; ok {
		return entry, true
	}
	return peerEntry{}, false
}

// UpdateLocal updates the peer when it is owned by the local node. This
// increments the peers version so it is propagated around the cluster.
// If the value is unchanged, the version isn't updated (to avoid propagating
// redundant data).
func (p *Peer) UpdateLocal(key string, value string) {
	if entry, ok := p.entries[key]; ok {
		if entry.Value == value {
			return
		}
	}

	p.version++
	p.entries[key] = peerEntry{
		Version: p.version,
		Value:   value,
	}
}

// UpdateRemote updates the peer from an update from a remote node. If the
// local version of that entry is greater than the new version, the update is
// discarded.
func (p *Peer) UpdateRemote(key string, value string, version uint64) {
	// Ignore updates with a smaller version than the current entry.
	if entry, ok := p.entries[key]; ok {
		if version <= entry.Version {
			return
		}
	}

	p.entries[key] = peerEntry{
		Version: version,
		Value:   value,
	}
	if version > p.version {
		p.version = version
	}
}

func (p *Peer) Digest() PeerDigest {
	return PeerDigest{
		Addr:    p.addr,
		Version: p.version,
	}
}

// Deltas returns all entries whos versions exceed the given version, ordered
// by version.
//
// Note ordering deltas by version (per peer) is important since the full
// delta may not be sent - and we can't have gaps in versions.
func (p *Peer) Deltas(version uint64) PeerDelta {
	deltas := []DeltaEntry{}
	for key, entry := range p.entries {
		if entry.Version <= version {
			continue
		}

		deltas = append(deltas, DeltaEntry{
			Key:     key,
			Value:   entry.Value,
			Version: entry.Version,
		})
	}

	// Sort by version. There may be a more efficient way to store this but
	// for now sorting is fine.
	sort.Slice(deltas, func(i, j int) bool {
		return deltas[i].Version < deltas[j].Version
	})

	return PeerDelta{
		Addr:   p.addr,
		Deltas: deltas,
	}
}
