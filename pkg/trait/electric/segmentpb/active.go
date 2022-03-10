package segmentpb

import (
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
)

// ActiveAt returns the segment active at d or nil if there is no such segment.
// Returns 0 if d is before all segments, and len(segments) if d is after all segments, otherwise returns the index of
// the segment in segments.
func ActiveAt(d time.Duration, segments ...*traits.ElectricMode_Segment) (segment *traits.ElectricMode_Segment, i int) {
	if d < 0 {
		return nil, 0
	}

	var cur time.Duration
	for i, segment = range segments {
		if segment.Length == nil {
			return segment, i
		}

		cur += segment.Length.AsDuration()
		if cur > d {
			return segment, i
		}
	}

	return nil, len(segments)
}
