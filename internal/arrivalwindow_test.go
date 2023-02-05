package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArrivalWindow(t *testing.T) {
	tests := []struct {
		Name        string
		ExpectedPhi float64
		Timestamps  []uint64
		Now         uint64
		SampleSize  int
	}{
		{
			Name:        "bootstrap phi",
			ExpectedPhi: 0.05,
			Timestamps:  []uint64{100},
			Now:         200,
			SampleSize:  10,
		},
		{
			Name:        "low phi",
			ExpectedPhi: 1.0,
			Timestamps:  []uint64{100, 200, 300, 400, 500, 600},
			Now:         700,
			SampleSize:  5,
		},
		{
			Name:        "high phi",
			ExpectedPhi: 14.0,
			Timestamps:  []uint64{100, 200, 300, 400, 500, 600},
			Now:         2000,
			SampleSize:  5,
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			window := NewArrivalWindow(1000, test.SampleSize)
			for _, ts := range test.Timestamps {
				window.Add(ts)
			}

			assert.InEpsilon(t, test.ExpectedPhi, window.Phi(test.Now), 0.01)
		})
	}
}
