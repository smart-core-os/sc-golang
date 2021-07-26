package memory

import (
	"context"
	"sync"

	"github.com/olebedev/emitter"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-golang/pkg/masks"
)

// Resource represents a simple state field in an object. Think Temperature or Volume or Occupancy. Use a Resource to
// gain thread safe reads/writes that also support FieldMasks and update notifications.
type Resource struct {
	writableFields *fieldmaskpb.FieldMask

	mu    sync.RWMutex
	value proto.Message

	bus *emitter.Emitter
}

func NewResource(opts ...ResourceOption) *Resource {
	res := &Resource{
		bus: &emitter.Emitter{},
	}
	for _, opt := range opts {
		opt(res)
	}
	return res
}

func (r *Resource) defaultUpdateRequest() updateRequest {
	request := updateRequest{}
	for _, opt := range DefaultUpdateOptions {
		opt(&request)
	}
	return request
}

func (r *Resource) Get() proto.Message {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.value
}

func (r *Resource) Set(value proto.Message, opts ...UpdateOption) (proto.Message, error) {
	request := r.defaultUpdateRequest()
	for _, opt := range opts {
		opt(&request)
	}
	return r.set(value, request)
}

func (r *Resource) set(value proto.Message, request updateRequest) (proto.Message, error) {
	writer := masks.NewFieldUpdater(masks.WithWritableFields(r.writableFields), masks.WithUpdateMask(request.updateMask))
	if err := writer.Validate(value); err != nil {
		return nil, err
	}

	_, newValue, err := getAndUpdate(
		&r.mu,
		func() (proto.Message, error) {
			return r.value, nil
		},
		func(old, new proto.Message) error {
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

	r.bus.Emit("update", &ResourceChange{
		Value:      newValue,
		ChangeTime: serverTimestamp(),
	})

	return newValue, err
}

func (r *Resource) OnUpdate(ctx context.Context) (updates <-chan *ResourceChange, done func()) {
	on := r.bus.On("update")
	typedEvents := make(chan *ResourceChange)
	go func() {
		defer close(typedEvents)
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-on:
				if !ok {
					return
				}
				select {
				case <-ctx.Done():
					return // give up sending
				case typedEvents <- event.Args[0].(*ResourceChange):
				}
			}
		}
	}()
	return typedEvents, func() {
		// note: causes the listener to close, which eventually closes the typedEvents chan too
		r.bus.Off("update", on)
	}
}

type ResourceChange struct {
	Value      proto.Message
	ChangeTime *timestamppb.Timestamp
}

// ResourceOption allows configuration of the resource
type ResourceOption func(resource *Resource)

// WithInitialValue sets the initial value of a Resource
func WithInitialValue(v proto.Message) ResourceOption {
	return func(r *Resource) {
		r.value = v
	}
}

// WithWritablePaths sets the fields that can be modified via Update calls.
// Will panic if paths are not valid according to the message type.
func WithWritablePaths(m proto.Message, paths ...string) ResourceOption {
	mask, err := fieldmaskpb.New(m, paths...)
	if err != nil {
		panic(err)
	}
	return func(r *Resource) {
		r.writableFields = mask
	}
}

type UpdateInterceptor func(old, new proto.Message)

type updateRequest struct {
	updateMask        *fieldmaskpb.FieldMask
	interceptBefore   UpdateInterceptor
	interceptAfter    UpdateInterceptor
	nilWritableFields bool
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
func WithAllFieldsWritable() UpdateOption {
	return func(request *updateRequest) {
		request.nilWritableFields = true
	}
}
