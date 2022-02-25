package resource

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// serverTimestamp returns a timestamppb.Now() but is a var so it can be overridden for tests
var serverTimestamp = func() *timestamppb.Timestamp {
	return timestamppb.Now()
}

// Clock defines all time related features required by this package.
type Clock interface {
	Now() time.Time
}

// WallClock returns a Clock backed by the time package.
func WallClock() Clock {
	return realClock{}
}

type realClock struct{}

func (r realClock) Now() time.Time {
	return time.Now()
}
