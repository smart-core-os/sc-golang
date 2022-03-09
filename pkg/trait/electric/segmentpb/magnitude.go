package segmentpb

import (
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/protobuf/types/known/durationpb"
)

// MaxMagnitude returns the largest segment magnitude of all non-zero length segments.
func MaxMagnitude(segments ...*traits.ElectricMode_Segment) (max float32) {
	for _, segment := range segments {
		if segment.Length != nil && !durationPositive(segment.Length) {
			continue // don't count zero length segments
		}
		if segment.Magnitude > max {
			max = segment.Magnitude
		}
	}
	return max
}

// SumMagnitude sums the magnitude of all the given segments.
func SumMagnitude(segments ...*traits.ElectricMode_Segment) (sum float32) {
	for _, segment := range segments {
		sum += segment.Magnitude
	}
	return sum
}

// MagnitudeAt returns the magnitude of the segment active at d.
// If there is no segment at d, ok will be false.
func MagnitudeAt(d time.Duration, segments ...*traits.ElectricMode_Segment) (level float32, ok bool) {
	s, _ := ActiveAt(d, segments...)
	if s == nil {
		return 0, false
	}
	return s.Magnitude, true
}

func durationPositive(d *durationpb.Duration) bool {
	return d.AsDuration() > 0
}
