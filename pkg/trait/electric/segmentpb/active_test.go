package segmentpb

import (
	"reflect"
	"testing"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
)

func TestActiveAt(t *testing.T) {
	tests := []struct {
		name        string
		d           time.Duration
		segments    []*traits.ElectricMode_Segment
		wantSegment *traits.ElectricMode_Segment
		wantI       int
	}{
		{"empty", 0, segs(), nil, 0},
		{"start", 0, segs(s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), seg(s{1, 1}), 0},
		{"end1", 1, segs(s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), seg(s{2, 2}), 1},
		{"mid", 2, segs(s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), seg(s{2, 2}), 1},
		{"end2", 3, segs(s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), seg(s{3, 3}), 2},
		{"end-n", 11, segs(s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), nil, 4},
		{"after", 100, segs(s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), nil, 4},
		{"inf", 3, segs(s{1, 1}, s{2, 2}, s{3, 0}, s{5, 5}), seg(s{3, 0}), 2},
		{"inf-after", 8, segs(s{1, 1}, s{2, 2}, s{3, 0}, s{5, 5}), seg(s{3, 0}), 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSegment, gotI := ActiveAt(tt.d, tt.segments...)
			if !reflect.DeepEqual(gotSegment, tt.wantSegment) {
				t.Errorf("ActiveAt() gotSegment = %v, want %v", gotSegment, tt.wantSegment)
			}
			if gotI != tt.wantI {
				t.Errorf("ActiveAt() gotI = %v, want %v", gotI, tt.wantI)
			}
		})
	}
}
