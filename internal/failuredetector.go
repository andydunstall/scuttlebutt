package internal

import (
	"sync"
	"time"
)

type PeerStatus int

const (
	PeerStatusUp   = PeerStatus(1)
	PeerStatusDown = PeerStatus(2)
)

// FailureDetector is a failure detector that detects down nodes based on
// incoming heartbeats.
//
// This implements the paper "The Phi Accrual Failure Detector".
type FailureDetector struct {
	// mu is a mutex protecting the below fields
	mu sync.Mutex

	windows map[string]*ArrivalWindow

	sampleSize     int
	gossipInterval uint64

	convictThreshold float64
}

func NewFailureDetector(gossipInterval uint64, sampleSize int, convictThreshold float64) *FailureDetector {
	return &FailureDetector{
		windows:          make(map[string]*ArrivalWindow),
		sampleSize:       sampleSize,
		gossipInterval:   gossipInterval,
		convictThreshold: convictThreshold,
	}
}

func (fd *FailureDetector) PeerStatus(endpoint string) PeerStatus {
	return fd.PeerStatusAtTimestamp(endpoint, uint64(time.Now().UnixNano()))
}

func (fd *FailureDetector) PeerStatusAtTimestamp(endpoint string, timestampNano uint64) PeerStatus {
	fd.mu.Lock()
	defer fd.mu.Unlock()

	window, ok := fd.windows[endpoint]
	if !ok {
		// If we have never received any heartbeats from the node, start by
		// assuming it is alive, though add an initial bootstrap interval so we
		// can eventually detect the node as down if we never receive any
		// heartbeats.
		window = NewArrivalWindow(fd.gossipInterval, fd.sampleSize)
		window.Add(timestampNano)
		fd.windows[endpoint] = window
	}

	phi := window.Phi(timestampNano)
	if phi > fd.convictThreshold {
		return PeerStatusDown
	}
	return PeerStatusUp
}

func (fd *FailureDetector) Report(endpoint string) {
	fd.ReportWithTimestamp(endpoint, uint64(time.Now().UnixNano()))
}

func (fd *FailureDetector) ReportWithTimestamp(endpoint string, timestampNano uint64) {
	fd.mu.Lock()
	defer fd.mu.Unlock()

	window, ok := fd.windows[endpoint]
	if !ok {
		window = NewArrivalWindow(fd.gossipInterval, fd.sampleSize)
		fd.windows[endpoint] = window
	}

	window.Add(timestampNano)
}

func (fd *FailureDetector) RemovePeer(endpoint string) {
	fd.mu.Lock()
	defer fd.mu.Unlock()

	delete(fd.windows, endpoint)
}
