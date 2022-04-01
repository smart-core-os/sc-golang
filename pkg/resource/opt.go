package resource

import (
	"io"
	"math/rand"
	"time"

	"github.com/smart-core-os/sc-golang/pkg/cmp"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// Option configures a resource value or collection.
type Option interface {
	apply(s *config)
}

// WithClock configures the clock used when time is needed.
// Defaults to a Clock backed by the time package.
func WithClock(c Clock) Option {
	return optionFunc(func(s *config) {
		s.clock = c
	})
}

// WithEquivalence configures how consecutive emissions are compared, equivalent emissions are not emitted.
// Defaults to nil, no equivalence checking is performed, all events will be emitted.
func WithEquivalence(e Comparer) Option {
	return optionFunc(func(s *config) {
		s.equivalence = e
	})
}

// WithMessageEquivalence is like WithEquivalence but using a cmp.Message.
func WithMessageEquivalence(e cmp.Message) Option {
	return WithEquivalence(ComparerFunc(e))
}

// WithNoDuplicates is like WithMessageEquivalence(cmp.Equal()).
func WithNoDuplicates() Option {
	return WithMessageEquivalence(cmp.Equal())
}

// WithRNG configures the source of randomness for the resource.
// Defaults to rand.Rand with a time seed.
func WithRNG(rng io.Reader) Option {
	return optionFunc(func(s *config) {
		s.rng = rng
	})
}

// WithInitialValue configures the initial value for the resource.
// Applies only to Value.
func WithInitialValue(initialValue proto.Message) Option {
	return optionFunc(func(s *config) {
		s.initialValue = initialValue
	})
}

// WithWritableFields configures write operations on the resource to accept updates to the given fields only.
// Explicit writes to fields not in this mask will fail.
func WithWritableFields(mask *fieldmaskpb.FieldMask) Option {
	return optionFunc(func(s *config) {
		s.writableFields = mask
	})
}

// WithWritablePaths is like WithWritableFields using fieldmaskpb.New.
func WithWritablePaths(m proto.Message, paths ...string) Option {
	mask, err := fieldmaskpb.New(m, paths...)
	if err != nil {
		panic(err)
	}
	return WithWritableFields(mask)
}

type config struct {
	clock          Clock
	equivalence    Comparer
	rng            io.Reader
	initialValue   proto.Message
	writableFields *fieldmaskpb.FieldMask
}

func computeConfig(opts ...Option) *config {
	c := &config{
		clock: WallClock(),
		rng:   rand.New(rand.NewSource(time.Now().Unix())),
	}
	for _, opt := range opts {
		opt.apply(c)
	}
	return c
}

type optionFunc func(s *config)

func (f optionFunc) apply(s *config) {
	f(s)
}

// ReadOption configures settings for reading data.
type ReadOption interface {
	apply(rr *readRequest)
}

// WithReadMask configures the properties that will be filled in the response value.
func WithReadMask(mask *fieldmaskpb.FieldMask) ReadOption {
	return readOptionFunc(func(rr *readRequest) {
		rr.readMask = mask
	})
}

// WithReadPaths configures the properties that will be filled in the response value.
// Panics if paths aren't part of m.
func WithReadPaths(m proto.Message, paths ...string) ReadOption {
	mask, err := fieldmaskpb.New(m, paths...)
	if err != nil {
		panic(err)
	}
	return WithReadMask(mask)
}

// WithUpdatesOnly instructs Pull methods to only send updates.
// The default behaviour is to send the current value, followed by future updates.
func WithUpdatesOnly(updatesOnly bool) ReadOption {
	return readOptionFunc(func(rr *readRequest) {
		rr.updatesOnly = updatesOnly
	})
}

func computeReadConfig(opts ...ReadOption) *readRequest {
	rr := &readRequest{}
	for _, opt := range opts {
		opt.apply(rr)
	}
	return rr
}

type readRequest struct {
	readMask *fieldmaskpb.FieldMask

	updatesOnly bool
}

// ResponseFilter returns a masks.ResponseFilter configured using this readRequest properties.
func (rr *readRequest) ResponseFilter() *masks.ResponseFilter {
	return masks.NewResponseFilter(masks.WithFieldMask(rr.readMask))
}

// FilterClone in the equivalent of rr.ResponseFilter().FilterClone(m).
func (rr *readRequest) FilterClone(m proto.Message) proto.Message {
	return rr.ResponseFilter().FilterClone(m)
}

type readOptionFunc func(rr *readRequest)

func (r readOptionFunc) apply(rr *readRequest) {
	r(rr)
}

type WriteOption interface{ apply(wr *writeRequest) }

func computeWriteConfig(opts ...WriteOption) writeRequest {
	req := &writeRequest{}
	for _, opt := range opts {
		opt.apply(req)
	}
	return *req
}

// UpdateInterceptor describes a function that applies changes to an existing message
type UpdateInterceptor func(old, new proto.Message)

type writeRequest struct {
	writeTime *time.Time

	updateMask    *fieldmaskpb.FieldMask
	resetMask     *fieldmaskpb.FieldMask
	expectedValue proto.Message

	interceptBefore UpdateInterceptor
	interceptAfter  UpdateInterceptor

	nilWritableFields  bool
	moreWritableFields *fieldmaskpb.FieldMask

	createIfAbsent  bool
	createdCallback func()
}

func (wr writeRequest) fieldUpdater(writableFields *fieldmaskpb.FieldMask) *masks.FieldUpdater {
	opts := []masks.FieldUpdaterOption{
		masks.WithUpdateMask(wr.updateMask),
		masks.WithResetMask(wr.resetMask),
	}
	if !wr.nilWritableFields {
		// A nil writable fields means all fields are writable, no point merging in this case.
		// If we blindly merged r.writableFields with request.moreWritableFields we could end up with
		// an empty FieldMask when both are nil resulting in no writable fields instead of all writable.
		if writableFields != nil {
			fields := fieldmaskpb.Union(writableFields, wr.moreWritableFields)
			opts = append(opts, masks.WithWritableFields(fields))
		}
	}
	return masks.NewFieldUpdater(opts...)
}

func (wr writeRequest) changeFn(writer *masks.FieldUpdater, value proto.Message) ChangeFn {
	return func(old, new proto.Message) error {
		if wr.expectedValue != nil {
			if !proto.Equal(old, wr.expectedValue) {
				return ExpectedValuePreconditionFailed
			}
		}

		if wr.interceptBefore != nil {
			// allow callers to update the value based on the old message
			wr.interceptBefore(old, value)
		}

		writer.Merge(new, value)

		if wr.interceptAfter != nil {
			// apply any after change changes, like setting update times
			wr.interceptAfter(old, new)
		}
		return nil
	}
}

func (wr writeRequest) updateTime(clock Clock) time.Time {
	if wr.writeTime != nil {
		return *wr.writeTime
	}
	return clock.Now()
}

type writeOptionFunc func(wr *writeRequest)

func (w writeOptionFunc) apply(wr *writeRequest) {
	w(wr)
}

// WithWriteTime configures the update to behave as if the write happened at time t, instead of now.
// Any change events that may be emitted with this write use t as their ChangeTime.
// Computational values, for example tweening, can use this to correctly determine the computed value.
func WithWriteTime(t time.Time) WriteOption {
	return writeOptionFunc(func(wr *writeRequest) {
		wr.writeTime = &t
	})
}

// WithUpdateMask configures the update to only apply to these fields.
// nil will update all writable fields.
// Fields specified here that aren't in the Resources writable fields will result in an error
func WithUpdateMask(mask *fieldmaskpb.FieldMask) WriteOption {
	return writeOptionFunc(func(request *writeRequest) {
		request.updateMask = mask
	})
}

// WithUpdatePaths is like WithUpdateMask but the FieldMask is made from the given paths.
func WithUpdatePaths(paths ...string) WriteOption {
	return WithUpdateMask(&fieldmaskpb.FieldMask{Paths: paths})
}

// WithResetMask configures the update to clear these fields from the final value.
// This will happen after InterceptBefore, but before InterceptAfter.
// WithWritableFields does not affect this.
func WithResetMask(mask *fieldmaskpb.FieldMask) WriteOption {
	return writeOptionFunc(func(request *writeRequest) {
		request.resetMask = mask
	})
}

// WithResetPaths is like WithResetMask but the FieldMask is made from the given paths.
func WithResetPaths(paths ...string) WriteOption {
	return WithResetMask(&fieldmaskpb.FieldMask{Paths: paths})
}

// InterceptBefore registers a function that will be called before the update occurs.
// The new value passed to the function will be the Message given as part of the update operation.
// Do not write to the old value of the callback, this is for information only.
// This is useful when applying delta update to a value, in this case you can append the old value to the update value
// to get the sum.
//
// Example
//
//   r.Set(val, InterceptBefore(func(old, change proto.Message) {
//     if val.Delta {
//       // assume casting
//       change.Quantity += old.Quantity
//     }
//   }))
func InterceptBefore(interceptor UpdateInterceptor) WriteOption {
	return writeOptionFunc(func(request *writeRequest) {
		request.interceptBefore = interceptor
	})
}

// InterceptAfter registers a function that will be called after changes have been made but before they are saved.
// This is useful if there are computed properties in the message that might need setting if an update has occurred,
// for example a `LastUpdateTime` or similar.
//
// Example
//
//   r.Set(val, InterceptAfter(func(old, new proto.Message) {
//     // assume casting
//     if old.Quantity != new.Quantity {
//       new.UpdateTime = timestamppb.Now()
//     }
//   }))
func InterceptAfter(interceptor UpdateInterceptor) WriteOption {
	return writeOptionFunc(func(request *writeRequest) {
		request.interceptAfter = interceptor
	})
}

// WithAllFieldsWritable instructs the update to ignore the resources configured writable fields.
// All fields will be writable if using this option.
// Prefer WithMoreWritableFields if possible.
func WithAllFieldsWritable() WriteOption {
	return writeOptionFunc(func(request *writeRequest) {
		request.nilWritableFields = true
	})
}

// WithMoreWritableFields adds the given fields to the resources configured writable fields before validating the update.
// Prefer this over WithAllFieldsWritable.
func WithMoreWritableFields(writableFields *fieldmaskpb.FieldMask) WriteOption {
	return writeOptionFunc(func(request *writeRequest) {
		request.moreWritableFields = fieldmaskpb.Union(request.moreWritableFields, writableFields)
	})
}

// WithMoreWritablePaths is like WithMoreWritableFields but with paths instead.
func WithMoreWritablePaths(writablePaths ...string) WriteOption {
	return WithMoreWritableFields(&fieldmaskpb.FieldMask{Paths: writablePaths})
}

// ExpectedValuePreconditionFailed is returned when an update configured WithExpectedValue fails its comparison.
var ExpectedValuePreconditionFailed = status.Errorf(codes.FailedPrecondition, "current value is not as expected")

// WithExpectedValue instructs the update to only proceed if the current value is equal to expectedValue.
// If the precondition fails the update will return the error ExpectedValuePreconditionFailed.
// The precondition will be checked _before_ InterceptBefore.
func WithExpectedValue(expectedValue proto.Message) WriteOption {
	return writeOptionFunc(func(request *writeRequest) {
		request.expectedValue = expectedValue
	})
}

// WithCreateIfAbsent instructs the write to create an entry if none already exist.
// Applicable only to Collection updates.
// When specified any interceptors will receive a zero old value of the collection item type.
func WithCreateIfAbsent() WriteOption {
	return writeOptionFunc(func(wr *writeRequest) {
		wr.createIfAbsent = true
	})
}

// WithCreatedCallback calls cb if during an update, a new value is created.
// Applicable only to Collection updates.
// Use the response from the Update call to get the actual value.
func WithCreatedCallback(cb func()) WriteOption {
	return writeOptionFunc(func(wr *writeRequest) {
		wr.createdCallback = cb
	})
}
