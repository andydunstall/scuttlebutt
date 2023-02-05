package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFailureDetector(t *testing.T) {
	tests := []struct {
		Name           string
		ExpectedStatus PeerStatus
		Timestamps     []uint64
		Now            uint64
		SampleSize     int
	}{
		{
			Name:           "bootstrap status",
			ExpectedStatus: PeerStatusUp,
			Timestamps:     []uint64{100},
			Now:            200,
			SampleSize:     10,
		},
		{
			Name:           "peer up",
			ExpectedStatus: PeerStatusUp,
			Timestamps:     []uint64{100, 200, 300, 400, 500, 600},
			Now:            700,
			SampleSize:     5,
		},
		{
			Name:           "peer down",
			ExpectedStatus: PeerStatusDown,
			Timestamps:     []uint64{100, 200, 300, 400, 500, 600},
			Now:            2000,
			SampleSize:     5,
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			failureDetector := NewFailureDetector(1000, test.SampleSize, 8.0)
			for _, ts := range test.Timestamps {
				failureDetector.ReportWithTimestamp("my-endpoint", ts)
			}

			assert.Equal(
				t,
				test.ExpectedStatus,
				failureDetector.PeerStatusAtTimestamp("my-endpoint", test.Now),
			)
		})
	}
}
