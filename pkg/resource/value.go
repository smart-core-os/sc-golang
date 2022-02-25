package resource

import (
	"context"
	"sync"

	"github.com/olebedev/emitter"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// Value represents a simple state field in an object. Think Temperature or Volume or Occupancy. Use a Value to
// gain thread safe reads/writes that also support FieldMasks and update notifications.
type Value struct {
	*config

	mu    sync.RWMutex
	value proto.Message

	bus *emitter.Emitter
}

func NewValue(opts ...Option) *Value {
	c := computeConfig(opts...)
	res := &Value{
		config: c,
		bus:    &emitter.Emitter{},
	}
	res.value = c.initialValue
	c.initialValue = nil // clear so it can be GC'd when the value changes
	return res
}

func (r *Value) Get(opts ...ReadOption) proto.Message {
	return r.get(computeReadConfig(opts...))
}

func (r *Value) get(req *readRequest) proto.Message {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return req.FilterClone(r.value)
}

// Set updates the current value of this Value with the given value.
// Returns the new value.
// Provide UpdateOption to control masks and other variables during the update.
func (r *Value) Set(value proto.Message, opts ...UpdateOption) (proto.Message, error) {
	request := updateRequest{}
	for _, opt := range DefaultUpdateOptions {
		opt(&request)
	}
	for _, opt := range opts {
		opt(&request)
	}
	return r.set(value, request)
}

func (r *Value) set(value proto.Message, request updateRequest) (proto.Message, error) {
	opts := []masks.FieldUpdaterOption{
		masks.WithUpdateMask(request.updateMask),
		masks.WithResetMask(request.resetMask),
	}
	if !request.nilWritableFields {
		// A nil writable fields means all fields are writable, no point merging in this case.
		// If we blindly merged r.writableFields with request.moreWritableFields we could end up with
		// an empty FieldMask when both are nil resulting in no writable fields instead of all writable.
		if r.writableFields != nil {
			fields := fieldmaskpb.Union(r.writableFields, request.moreWritableFields)
			opts = append(opts, masks.WithWritableFields(fields))
		}
	}
	writer := masks.NewFieldUpdater(opts...)
	if err := writer.Validate(value); err != nil {
		return nil, err
	}

	_, newValue, err := GetAndUpdate(
		&r.mu,
		func() (proto.Message, error) {
			return r.value, nil
		},
		func(old, new proto.Message) error {
			if request.expectedValue != nil {
				if !proto.Equal(old, request.expectedValue) {
					return ExpectedValuePreconditionFailed
				}
			}

			if request.interceptBefore != nil {
				// convert the value from relative to absolute values
				request.interceptBefore(old, value)
			}

			writer.Merge(new, value)

			if request.interceptAfter != nil {
				// apply any after change changes, like setting update times
				request.interceptAfter(old, new)
			}
			return nil
		},
		func(message proto.Message) {
			r.value = message
		},
	)

	if err != nil {
		return nil, err
	}

	r.bus.Emit("update", &ValueChange{
		Value:      newValue,
		ChangeTime: r.clock.Now(),
	})

	return newValue, err
}

// Pull emits a ValueChange on the returned chan whenever the underlying value changes.
// The changes emitted can be adjusted using WithEquivalence.
// The returned chan will be closed when no more events will be emitted, either because ctx was cancelled or for other
// reasons.
func (r *Value) Pull(ctx context.Context, opts ...ReadOption) <-chan *ValueChange {
	readConfig := computeReadConfig(opts...)
	filter := readConfig.ResponseFilter()
	on := r.bus.On("update")
	typedEvents := make(chan *ValueChange)
	go func() {
		defer close(typedEvents)
		defer r.bus.Off("update", on)

		var last proto.Message
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-on:
				if !ok {
					return // the listener was cancelled
				}
				change := event.Args[0].(*ValueChange).filter(filter)
				if r.equivalence != nil && r.equivalence.Compare(last, change.Value) {
					continue
				}
				last = change.Value
				select {
				case <-ctx.Done():
					return // give up sending
				case typedEvents <- change:
				}
			}
		}
	}()
	return typedEvents
}

type UpdateInterceptor func(old, new proto.Message)

type updateRequest struct {
	updateMask    *fieldmaskpb.FieldMask
	resetMask     *fieldmaskpb.FieldMask
	expectedValue proto.Message

	interceptBefore UpdateInterceptor
	interceptAfter  UpdateInterceptor

	nilWritableFields  bool
	moreWritableFields *fieldmaskpb.FieldMask
}

type UpdateOption func(request *updateRequest)

// DefaultUpdateOptions defined the options that apply unless overridden by callers
// when updates are applied to the resource.
var DefaultUpdateOptions []UpdateOption

// WithUpdateMask configures the update to only apply to these fields.
// nil will update all writable fields.
// Fields specified here that aren't in the Resources writable fields will result in an error
func WithUpdateMask(mask *fieldmaskpb.FieldMask) UpdateOption {
	return func(request *updateRequest) {
		request.updateMask = mask
	}
}

// WithUpdatePaths is like WithUpdateMask but the FieldMask is made from the given paths.
func WithUpdatePaths(paths ...string) UpdateOption {
	return WithUpdateMask(&fieldmaskpb.FieldMask{Paths: paths})
}

// WithResetMask configures the update to clear these fields from the final value.
// This will happen after InterceptBefore, but before InterceptAfter.
// WithWritableFields does not affect this.
func WithResetMask(mask *fieldmaskpb.FieldMask) UpdateOption {
	return func(request *updateRequest) {
		request.resetMask = mask
	}
}

// WithResetPaths is like WithResetMask but the FieldMask is made from the given paths.
func WithResetPaths(paths ...string) UpdateOption {
	return WithResetMask(&fieldmaskpb.FieldMask{Paths: paths})
}

// InterceptBefore registers a function that will be called before the update occurs.
// The new value will be the passed update value.
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
func InterceptBefore(interceptor UpdateInterceptor) UpdateOption {
	return func(request *updateRequest) {
		request.interceptBefore = interceptor
	}
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
func InterceptAfter(interceptor UpdateInterceptor) UpdateOption {
	return func(request *updateRequest) {
		request.interceptAfter = interceptor
	}
}

// WithAllFieldsWritable instructs the update to ignore the resources configured writable fields.
// All fields will be writable if using this option.
// Prefer WithMoreWritableFields if possible.
func WithAllFieldsWritable() UpdateOption {
	return func(request *updateRequest) {
		request.nilWritableFields = true
	}
}

// WithMoreWritableFields adds the given fields to the resources configured writable fields before validating the update.
// Prefer this over WithAllFieldsWritable.
func WithMoreWritableFields(writableFields *fieldmaskpb.FieldMask) UpdateOption {
	return func(request *updateRequest) {
		request.moreWritableFields = fieldmaskpb.Union(request.moreWritableFields, writableFields)
	}
}

// WithMoreWritablePaths is like WithMoreWritableFields but with paths instead.
func WithMoreWritablePaths(writablePaths ...string) UpdateOption {
	return WithMoreWritableFields(&fieldmaskpb.FieldMask{Paths: writablePaths})
}

// ExpectedValuePreconditionFailed is returned when an update configured WithExpectedValue fails its comparison.
var ExpectedValuePreconditionFailed = status.Errorf(codes.FailedPrecondition, "current value is not as expected")

// WithExpectedValue instructs the update to only proceed if the current value is equal to expectedValue.
// If the precondition fails the update will return the error ExpectedValuePreconditionFailed.
// The precondition will be checked _before_ InterceptBefore.
func WithExpectedValue(expectedValue proto.Message) UpdateOption {
	return func(request *updateRequest) {
		request.expectedValue = expectedValue
	}
}
