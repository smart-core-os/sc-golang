package resource

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-golang/internal/minibus"
)

// Value represents a simple state field in an object. Think Temperature or Volume or Occupancy. Use a Value to
// gain thread safe reads/writes that also support FieldMasks and update notifications.
type Value[T Message] struct {
	*config[T]

	mu         sync.RWMutex
	value      T
	changeTime time.Time

	bus minibus.Bus[*ValueChange[T]]
}

func NewValue[T Message](opts ...Option[T]) *Value[T] {
	c := computeConfig(opts...)
	res := &Value[T]{
		config: c,
	}
	res.value = c.initialValue
	res.changeTime = c.clock.Now()
	c.initialValue = zero[T]() // clear so it can be GC'd when the value changes
	return res
}

func (r *Value[T]) Get(opts ...ReadOption) proto.Message {
	return r.get(computeReadConfig(opts...))
}

func (r *Value[T]) get(req *readRequest) proto.Message {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return req.FilterClone(r.value)
}

// Set updates the current value of this Value with the given value.
// Returns the new value.
// Provide WriteOption to control masks and other variables during the update.
func (r *Value[T]) Set(value T, opts ...WriteOption[T]) (T, error) {
	return r.set(value, computeWriteConfig[T](opts...))
}

func (r *Value[T]) set(value T, request writeRequest[T]) (T, error) {
	writer := request.fieldUpdater(r.writableFields)
	if err := writer.Validate(value); err != nil {
		return zero[T](), err
	}

	disarm := timeoutAlarm(time.Second, "GetAndUpdate took too long")
	_, newValue, err := GetAndUpdate[T](
		&r.mu,
		func() (T, error) {
			return r.value, nil
		},
		request.changeFn(writer, value),
		func(message T) {
			r.value = message
			r.changeTime = request.updateTime(r.clock)
		},
	)
	disarm()

	if err != nil {
		return zero[T](), err
	}

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*5)
	defer cancel()
	r.bus.Send(ctx, &ValueChange[T]{
		Value:      newValue,
		ChangeTime: request.updateTime(r.clock),
	})
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return zero[T](), errors.New("bus.Send blocked for too long")
	}

	return newValue, err
}

// Pull emits a ValueChange on the returned chan whenever the underlying value changes.
// The changes emitted can be adjusted using WithEquivalence.
// The returned chan will be closed when no more events will be emitted, either because ctx was cancelled or for other
// reasons.
func (r *Value[T]) Pull(ctx context.Context, opts ...ReadOption) <-chan *ValueChange[T] {
	readConfig := computeReadConfig(opts...)
	filter := readConfig.ResponseFilter()
	on, currentValue, changeTime := r.onUpdate(ctx, readConfig)
	typedEvents := make(chan *ValueChange[T])
	go func() {
		defer close(typedEvents)

		if currentValue != zero[T]() {
			change := &ValueChange[T]{Value: currentValue, ChangeTime: changeTime}
			change = change.filter(filter)
			select {
			case <-ctx.Done():
				return // give up sending
			case typedEvents <- change:
			}
		}

		var last T
		for event := range on {
			change := event.filter(filter)
			if r.equivalence != nil && r.equivalence(last, change.Value) {
				continue
			}
			last = change.Value
			select {
			case <-ctx.Done():
				return // give up sending
			case typedEvents <- change:
			}
		}
	}()
	return typedEvents
}

func (r *Value[T]) onUpdate(ctx context.Context, config *readRequest) (<-chan *ValueChange[T], T, time.Time) {
	var (
		value      T
		changeTime time.Time
	)
	if !config.updatesOnly {
		r.mu.RLock()
		defer r.mu.RUnlock()
		value = r.value
		changeTime = r.changeTime
	}

	ch := r.bus.Listen(ctx)
	if !config.backpressure {
		ch = minibus.DropExcess(ch)
	}

	return ch, value, changeTime
}

func timeoutAlarm(duration time.Duration, fmt string, args ...interface{}) (disarm func()) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)

	go func() {
		<-ctx.Done()
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.Printf(fmt, args...)
		}
	}()

	return cancel
}

// Clock returns the clock used by this resource for reporting time.
func (r *Value[T]) Clock() Clock {
	return r.clock
}
