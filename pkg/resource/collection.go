package resource

import (
	"context"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/smart-core-os/sc-api/go/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-golang/internal/minibus"
)

type Collection[T Message] struct {
	*config[T]

	mu   sync.RWMutex // protects byId and rng from concurrent access
	byId map[string]*item[T]
	// "change" events contain a *CollectionChange instance
	bus minibus.Bus[*CollectionChange[T]]
}

func NewCollection[T Message](options ...Option[T]) *Collection[T] {
	conf := computeConfig(options...)
	initialItems := make(map[string]*item[T])
	for k, v := range conf.initialRecords {
		initialItems[k] = &item[T]{body: v, changeTime: conf.clock.Now()}
	}
	conf.initialRecords = nil // so the gc can collect them

	c := &Collection[T]{
		config: conf,
		byId:   initialItems,
		mu:     sync.RWMutex{},
	}

	return c
}

// Get will find the entry with the given ID. If no such entry exists, returns false.
func (c *Collection[T]) Get(id string, opts ...ReadOption) (proto.Message, bool) {
	readConfig := computeReadConfig(opts...)

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.byId[id]
	if !ok {
		return nil, false
	}

	return readConfig.FilterClone(entry.body), true
}

// List returns a list of all the entries, sorted by their ID.
func (c *Collection[T]) List(opts ...ReadOption) []proto.Message {
	readConfig := computeReadConfig(opts...)

	c.mu.RLock()
	defer c.mu.RUnlock()
	tmp := c.itemSlice(readConfig)
	sort.Slice(tmp, func(i, j int) bool {
		return tmp[i].id < tmp[j].id
	})

	result := make([]proto.Message, 0, len(tmp))
	filter := readConfig.ResponseFilter()
	for _, e := range tmp {
		result = append(result, filter.FilterClone(e.body))
	}
	return result
}

// Add associates the given body with the id.
// If id already exists then an error is returned.
func (c *Collection[T]) Add(id string, body T) (T, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	body, _, err := c.add(id, func(id string) T {
		return body
	})
	return body, err
}

// AddFn adds an entry to the collection by invoking create with a newly allocated ID.
func (c *Collection[T]) AddFn(create CreateFn[T]) (proto.Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	body, _, err := c.add("", create)
	return body, err
}

func (c *Collection[T]) add(id string, create CreateFn[T]) (T, string, error) {
	// todo: convert Collection.Add to use WriteOption

	if id == "" {
		var err error
		if id, err = c.genID(); err != nil {
			return zero[T](), "", err
		}
	} else {
		if _, exists := c.byId[id]; exists {
			return zero[T](), "", status.Errorf(codes.AlreadyExists, "%s cannot be created, already exists", id)
		}
	}

	body := create(id)
	c.byId[id] = &item[T]{body: body, changeTime: c.clock.Now()}
	c.bus.Send(context.TODO(), &CollectionChange[T]{
		Id:         id,
		ChangeTime: c.clock.Now(), // todo: allow specifying the writeTime, as part of using WriteOption
		ChangeType: types.ChangeType_ADD,
		NewValue:   body,
	})

	return body, id, nil
}

func (c *Collection[T]) Update(id string, msg T, opts ...WriteOption[T]) (T, error) {
	writeRequest := computeWriteConfig(opts...)
	writer := writeRequest.fieldUpdater(c.writableFields)
	if err := writer.Validate(msg); err != nil {
		return zero[T](), err
	}

	var created T // during create, this is returned by GetFn so concurrent reference checks pass
	oldValue, newValue, err := GetAndUpdate[T](
		&c.mu,
		func() (item T, err error) {
			if created != zero[T]() {
				return created, nil
			}
			val, exists := c.byId[id]
			if exists {
				return val.body, nil
			}
			if !writeRequest.createIfAbsent {
				return zero[T](), status.Errorf(codes.NotFound, "id %v not found", id)
			}
			created = msg.ProtoReflect().New().Interface().(T)
			if writeRequest.createdCallback != nil {
				writeRequest.createdCallback()
			}
			return created, nil
		},
		writeRequest.changeFn(writer, msg),
		func(msg T) {
			c.byId[id] = &item[T]{body: msg, changeTime: writeRequest.updateTime(c.clock)}
		})

	if err != nil {
		if s, ok := status.FromError(err); ok {
			return zero[T](), status.Errorf(s.Code(), "%v %v", s.Message(), id)
		}
		return zero[T](), err
	}
	changeType := types.ChangeType_UPDATE
	if oldValue == zero[T]() {
		changeType = types.ChangeType_ADD
	}
	c.bus.Send(context.TODO(), &CollectionChange[T]{
		Id:         id,
		ChangeTime: writeRequest.updateTime(c.clock),
		ChangeType: changeType,
		OldValue:   oldValue,
		NewValue:   newValue,
	})
	return newValue, nil
}

// Delete removes the item with the given id from this collection.
// The removed item will be returned, or nil if the item did not exist.
func (c *Collection[T]) Delete(id string) T {
	c.mu.Lock()
	defer c.mu.Unlock()

	oldVal, exists := c.byId[id]
	if !exists {
		return zero[T]()
	}
	delete(c.byId, id)
	c.bus.Send(context.TODO(), &CollectionChange[T]{
		Id:         id,
		ChangeTime: c.clock.Now(),
		ChangeType: types.ChangeType_REMOVE,
		OldValue:   oldVal.body,
	})
	return oldVal.body
}

func (c *Collection[T]) Pull(ctx context.Context, opts ...ReadOption) <-chan *CollectionChange[T] {
	readConfig := computeReadConfig(opts...)
	filter := readConfig.ResponseFilter()

	emit, currentValues := c.onUpdate(ctx, readConfig)
	send := make(chan *CollectionChange[T])

	go func() {
		defer close(send)

		if len(currentValues) > 0 {
			for _, value := range currentValues {
				change := &CollectionChange[T]{
					Id:         value.id,
					ChangeTime: value.changeTime,
					ChangeType: types.ChangeType_ADD,
					NewValue:   value.body,
				}
				change = change.filter(filter)
				select {
				case <-ctx.Done():
					return
				case send <- change:
				}
			}
		}

		for event := range emit {
			change := event
			change, ok := change.include(readConfig.include)
			if !ok {
				continue
			}
			change = change.filter(filter)
			if c.equivalence != nil && c.equivalence(change.OldValue, change.NewValue) {
				continue
			}
			select {
			case send <- change:
			case <-ctx.Done():
				return
			}
		}
	}()

	return send
}

// PullID subscribes to changes for a single item in the collection.
// The returned channel will close if ctx is Done or the item identified by id is deleted.
func (c *Collection[T]) PullID(ctx context.Context, id string, opts ...ReadOption) <-chan *ValueChange[T] {
	send := make(chan *ValueChange[T])
	go func() {
		defer close(send)
		for change := range c.Pull(ctx, opts...) {
			if change.Id != id {
				continue
			}

			if change.ChangeType == types.ChangeType_REMOVE {
				return
			}

			// not sure how this case could happen, but let's deal with it anyway
			if change.NewValue == zero[T]() {
				log.Printf("WARN: CollectionChange.NewValue is nil, but not a REMOVE change! %v", change)
				return
			}

			select {
			case <-ctx.Done():
				return
			case send <- &ValueChange[T]{ChangeTime: change.ChangeTime, Value: change.NewValue}:
			}
		}
	}()
	return send
}

func (c *Collection[T]) onUpdate(ctx context.Context, config *readRequest) (<-chan *CollectionChange[T], []idItem[T]) {
	var res []idItem[T]
	if !config.updatesOnly {
		c.mu.RLock()
		defer c.mu.RUnlock()
		res = c.itemSlice(config)
	}

	ch := c.bus.Listen(ctx)
	if !config.backpressure {
		ch = minibus.DropExcess(ch)
	}

	return ch, res
}

// Clock returns the clock used by this resource for reporting time.
func (c *Collection[T]) Clock() Clock {
	return c.clock
}

// itemSlice returns all the values in byId adjusted to match readConfig settings like readRequest.include.
func (c *Collection[T]) itemSlice(readConfig *readRequest) []idItem[T] {
	res := make([]idItem[T], 0, len(c.byId))
	for id, value := range c.byId {
		if readConfig.Exclude(id, value.body) {
			continue
		}
		res = append(res, idItem[T]{item: *value, id: id})
	}
	return res
}

func (c *Collection[T]) genID() (string, error) {
	return GenerateUniqueId(c.rng, func(candidate string) bool {
		_, exists := c.byId[candidate]
		return exists
	})
}

type item[T Message] struct {
	body       T
	changeTime time.Time
}

type idItem[T Message] struct {
	item[T]
	id string
}
