package resource

import (
	"context"
	"sync"
	"time"

	"github.com/olebedev/emitter"
	"google.golang.org/protobuf/proto"
)

// Value represents a simple state field in an object. Think Temperature or Volume or Occupancy. Use a Value to
// gain thread safe reads/writes that also support FieldMasks and update notifications.
type Value struct {
	*config

	mu         sync.RWMutex
	value      proto.Message
	changeTime time.Time

	bus *emitter.Emitter
}

func NewValue(opts ...Option) *Value {
	c := computeConfig(opts...)
	res := &Value{
		config: c,
		bus:    &emitter.Emitter{},
	}
	res.value = c.initialValue
	res.changeTime = c.clock.Now()
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
// Provide WriteOption to control masks and other variables during the update.
func (r *Value) Set(value proto.Message, opts ...WriteOption) (proto.Message, error) {
	return r.set(value, computeWriteConfig(opts...))
}

func (r *Value) set(value proto.Message, request writeRequest) (proto.Message, error) {
	writer := request.fieldUpdater(r.writableFields)
	if err := writer.Validate(value); err != nil {
		return nil, err
	}

	_, newValue, err := GetAndUpdate(
		&r.mu,
		func() (proto.Message, error) {
			return r.value, nil
		},
		request.changeFn(writer, value),
		func(message proto.Message) {
			r.value = message
			r.changeTime = request.updateTime(r.clock)
		},
	)

	if err != nil {
		return nil, err
	}

	r.bus.Emit("update", &ValueChange{
		Value:      newValue,
		ChangeTime: request.updateTime(r.clock),
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
	on, currentValue, changeTime := r.onUpdate(readConfig)
	typedEvents := make(chan *ValueChange)
	go func() {
		defer close(typedEvents)
		defer r.bus.Off("update", on)

		if currentValue != nil {
			change := &ValueChange{Value: currentValue, ChangeTime: changeTime}
			change = change.filter(filter)
			select {
			case <-ctx.Done():
				return // give up sending
			case typedEvents <- change:
			}
		}

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

func (r *Value) onUpdate(config *readRequest) (<-chan emitter.Event, proto.Message, time.Time) {
	if config.updatesOnly {
		return r.bus.On("update"), nil, r.changeTime
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.bus.On("update"), r.value, r.changeTime
}
