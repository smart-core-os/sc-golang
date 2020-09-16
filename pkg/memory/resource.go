package memory

import (
	"context"
	"sync"

	fieldMaskUtils "github.com/mennanov/fieldmask-utils"
	"github.com/olebedev/emitter"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"git.vanti.co.uk/smartcore/sc-golang/pkg/masks"
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

func (r *Resource) Get() proto.Message {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.value
}

func (r *Resource) Update(value proto.Message, updateMask *fieldmaskpb.FieldMask) (proto.Message, error) {
	return r.UpdateModified(value, updateMask, func(old, new proto.Message) {})
}

func (r *Resource) UpdateModified(value proto.Message, updateMask *fieldmaskpb.FieldMask, updateModifier func(old, new proto.Message)) (proto.Message, error) {
	// make sure they can only write the fields we want
	mask, err := masks.ValidWritableMask(r.writableFields, updateMask, value)
	if err != nil {
		return nil, err
	}

	_, newValue, err := applyChangeOld(
		&r.mu,
		func() (proto.Message, bool) {
			return r.value, true
		},
		func(old, new proto.Message) error {
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

			updateModifier(old, new)
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
				typedEvents <- event.Args[0].(*ResourceChange)
			}
		}
	}()
	return updates, func() {
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
