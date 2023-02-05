package internal

import (
	"math"
)

var (
	phiFactor = float64(1.0 / math.Log10(10.0))
)

type ArrivalWindow struct {
	lastTimestampNano uint64
	intervals         *ArrivalIntervals
	bootstrapInterval uint64
}

func NewArrivalWindow(gossipInterval uint64, sampleSize int) *ArrivalWindow {
	return &ArrivalWindow{
		lastTimestampNano: 0,
		intervals:         NewArrivalIntervals(sampleSize),
		bootstrapInterval: gossipInterval * 2,
	}
}

func (w *ArrivalWindow) Phi(timestampNano uint64) float64 {
	if !(w.lastTimestampNano > 0 && w.intervals.Mean() > 0.0) {
		panic("cannot sample phi before any samples arrived")
	}

	deltaSinceLast := timestampNano - w.lastTimestampNano
	return (float64(deltaSinceLast) / w.intervals.Mean()) * phiFactor
}

func (w *ArrivalWindow) Add(timestampNano uint64) {
	if w.lastTimestampNano > 0 {
		w.intervals.Add(timestampNano - w.lastTimestampNano)
	} else {
		// If this is the first interval, use a high interval to avoid false
		// positives when we don't have many samples.
		w.intervals.Add(w.bootstrapInterval)
	}
	w.lastTimestampNano = timestampNano
}

// ArrivalIntervals tracks the intervals in a circular buffer.
type ArrivalIntervals struct {
	intervals []uint64
	// index points to the next entry to add an interval. Since intervals is
	// a circular buffer this wraps around.
	index  int
	isFull bool

	sum  uint64
	mean float64
}

func NewArrivalIntervals(sampleSize int) *ArrivalIntervals {
	return &ArrivalIntervals{
		intervals: make([]uint64, sampleSize),
		index:     0,
		isFull:    false,
		sum:       0,
	}
}

func (ai *ArrivalIntervals) Mean() float64 {
	return ai.mean
}

func (ai *ArrivalIntervals) Add(interval uint64) {
	// If the index is at the end of the buffer wrap around.
	if ai.index == len(ai.intervals) {
		ai.index = 0
		ai.isFull = true
	}
	if ai.isFull {
		ai.sum = ai.sum - ai.intervals[ai.index]
	}

	ai.intervals[ai.index] = interval
	ai.index++
	ai.sum += interval
	ai.mean = float64(ai.sum) / float64(ai.size())
}

func (ai *ArrivalIntervals) size() int {
	if ai.isFull {
		return len(ai.intervals)
	}
	return ai.index
}
