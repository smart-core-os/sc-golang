package memory

import (
	"context"
	"sync"

	fieldMaskUtils "github.com/mennanov/fieldmask-utils"
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
		opt.apply(res)
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
	// make sure they can only write the fields we want
	mask, err := masks.ValidWritableMask(r.writableFields, request.updateMask, value)
	if err != nil {
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
			if mask != nil {
				// apply only selected fields
				if err := fieldMaskUtils.StructToStruct(mask, value, new); err != nil {
					return err
				}
			} else {
				// replace the booking data
				proto.Reset(new)
				proto.Merge(new, value)
			}

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

// Update applies properties from value to the underlying resource. Only updateMask properties will be changed
func (r *Resource) Update(value proto.Message, updateMask *fieldmaskpb.FieldMask) (proto.Message, error) {
	return r.Set(value, WithUpdateMask(updateMask))
}

// UpdateDelta works like Update but the given callback is called with the old value and the change to convert the
// change into absolute values.
func (r *Resource) UpdateDelta(value proto.Message, updateMask *fieldmaskpb.FieldMask, convertDelta UpdateInterceptor) (proto.Message, error) {
	return r.Set(value, WithUpdateMask(updateMask), InterceptBefore(convertDelta))
}

// UpdateModified works like Update but the callback is invoked after the update with the old and new values.
func (r *Resource) UpdateModified(value proto.Message, updateMask *fieldmaskpb.FieldMask, updateModifier UpdateInterceptor) (proto.Message, error) {
	return r.Set(value, WithUpdateMask(updateMask), InterceptAfter(updateModifier))
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
type ResourceOption interface {
	apply(*Resource)
}

// EmptyResourceOption does nothing to the Resource but can be used as a nil placeholder.
type EmptyResourceOption struct{}

func (EmptyResourceOption) apply(*Resource) {}

// funcResourceOption wraps a function that modifies a Resource into an
// implementation of the ResourceOption interface.
type funcResourceOption struct {
	f func(*Resource)
}

func (fro *funcResourceOption) apply(do *Resource) {
	fro.f(do)
}

func newFuncResourceOption(f func(*Resource)) *funcResourceOption {
	return &funcResourceOption{
		f: f,
	}
}

// WithInitialValue sets the initial value of a Resource
func WithInitialValue(v proto.Message) ResourceOption {
	return newFuncResourceOption(func(r *Resource) {
		r.value = v
	})
}

// WithWritableFields sets the fields that can be modified via Update calls
func WithWritableFields(fields *fieldmaskpb.FieldMask) ResourceOption {
	return newFuncResourceOption(func(r *Resource) {
		r.writableFields = fields
	})
}

// WithWritablePaths sets the fields that can be modified via Update calls
func WithWritablePaths(paths ...string) ResourceOption {
	return newFuncResourceOption(func(r *Resource) {
		r.writableFields = &fieldmaskpb.FieldMask{Paths: paths}
	})
}

type UpdateInterceptor func(old, new proto.Message)

type updateRequest struct {
	updateMask      *fieldmaskpb.FieldMask
	interceptBefore UpdateInterceptor
	interceptAfter  UpdateInterceptor
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

func InterceptBefore(interceptor UpdateInterceptor) UpdateOption {
	return func(request *updateRequest) {
		request.interceptBefore = interceptor
	}
}

func InterceptAfter(interceptor UpdateInterceptor) UpdateOption {
	return func(request *updateRequest) {
		request.interceptAfter = interceptor
	}
}
