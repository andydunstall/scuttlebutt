package internal

import (
	"sort"
	"time"
)

type PeerEntry struct {
	Version uint64
	Value   string
}

// Peer represents the state of a peer.
type Peer struct {
	addr string
	// version is the highest version of all the peers entries. This is used to
	// compare versions between nodes to check for missing updates.
	version uint64
	// entries contains the peer state to be propagated around the cluster.
	entries map[string]PeerEntry
	// status indicates whether the peer is considered up or down.
	status PeerStatus
	// expiry is the time the peer should be removed if it is still down.
	expiry time.Time
}

// NewPeer returns a new peer with the given address, with a version of 0 to
// indicate it has no known state.
func NewPeer(addr string) *Peer {
	return &Peer{
		addr:    addr,
		version: 0,
		entries: make(map[string]PeerEntry),
		status:  PeerStatusUp,
	}
}

func (p *Peer) Addr() string {
	return p.addr
}

func (p *Peer) Version() uint64 {
	return p.version
}

func (p *Peer) Status() PeerStatus {
	return p.status
}

func (p *Peer) Expiry() time.Time {
	return p.expiry
}

func (p *Peer) SetStatusUp() {
	p.status = PeerStatusUp
	p.expiry = time.Time{}
}

// SetStatusDown sets the status to down and sets the expiry of when the
// peer should be removed if it hasen't come up.
func (p *Peer) SetStatusDown(expiry time.Time) {
	// Check if the peer is already down to avoid resetting the expiry.
	if p.status == PeerStatusDown {
		return
	}

	p.status = PeerStatusDown
	p.expiry = expiry
}

func (p *Peer) Lookup(key string) (PeerEntry, bool) {
	if entry, ok := p.entries[key]; ok {
		return entry, true
	}
	return PeerEntry{}, false
}

func (p *Peer) Equal(o *Peer) bool {
	if p.addr != o.addr {
		return false
	}
	if p.version != o.version {
		return false
	}

	if len(p.entries) != len(o.entries) {
		return false
	}
	for k, v := range p.entries {
		w, ok := o.entries[k]
		if !ok {
			return false
		}
		if v.Version != w.Version {
			return false
		}
		if v.Value != w.Value {
			return false
		}
	}

	return true
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
	p.entries[key] = PeerEntry{
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

	p.entries[key] = PeerEntry{
		Version: version,
		Value:   value,
	}
	if version > p.version {
		p.version = version
	}
}

func (p *Peer) Digest() Digest {
	return Digest{
		Addr:    p.addr,
		Version: p.version,
	}
}

// Deltas returns all entries whos versions exceed the given version, ordered
// by version.
//
// Note the deltas are ordered by version since the full all deltas may not be
// sent and we can't have gaps in versions.
func (p *Peer) Deltas(version uint64) []Delta {
	deltas := []Delta{}
	for key, entry := range p.entries {
		if entry.Version <= version {
			continue
		}

		deltas = append(deltas, Delta{
			Addr:    p.addr,
			Key:     key,
			Value:   entry.Value,
			Version: entry.Version,
		})
	}

	// Sort by version.
	sort.Slice(deltas, func(i, j int) bool {
		return deltas[i].Version < deltas[j].Version
	})

	return deltas
}
