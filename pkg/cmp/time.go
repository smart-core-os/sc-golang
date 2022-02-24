package cmp

import (
	"time"

	pref "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TimeValueWithin(d time.Duration) Value {
	return func(fd pref.FieldDescriptor, x, y pref.Value) (equal bool, ok bool) {
		if fd.Kind() != pref.MessageKind {
			return false, false
		}
		mx, my := x.Message(), y.Message()
		xIsTimestamp := mx.Descriptor().FullName() == "google.protobuf.Timestamp"
		yIsTimestamp := my.Descriptor().FullName() == "google.protobuf.Timestamp"
		if !xIsTimestamp && !yIsTimestamp {
			return false, false
		}
		if xIsTimestamp != yIsTimestamp {
			return false, true
		}
		if !mx.IsValid() || !my.IsValid() {
			return mx.IsValid() == my.IsValid(), true
		}

		xt, yt := toTime(mx), toTime(my)
		if xt.Before(yt) {
			return yt.Sub(xt) <= d, true
		}
		return xt.Sub(yt) <= d, true
	}
}

func toTime(x pref.Message) time.Time {
	return x.Interface().(*timestamppb.Timestamp).AsTime()
}

func DurationValueWithin(d time.Duration) Value {
	return func(fd pref.FieldDescriptor, x, y pref.Value) (equal bool, ok bool) {
		if fd.Kind() != pref.MessageKind {
			return false, false
		}
		mx, my := x.Message(), y.Message()
		xIsDuration := mx.Descriptor().FullName() == "google.protobuf.Duration"
		yIsDuration := my.Descriptor().FullName() == "google.protobuf.Duration"
		if !xIsDuration && !yIsDuration {
			return false, false
		}
		if xIsDuration != yIsDuration {
			return false, true
		}
		if !mx.IsValid() || !my.IsValid() {
			return mx.IsValid() == my.IsValid(), true
		}

		xd, yd := toDuration(mx), toDuration(my)
		if xd < yd {
			return yd-xd <= d, true
		}
		return xd-yd <= d, true
	}
}

func toDuration(x pref.Message) time.Duration {
	return x.Interface().(*durationpb.Duration).AsDuration()
}
