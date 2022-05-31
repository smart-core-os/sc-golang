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

type Collection struct {
	*config

	mu   sync.RWMutex // protects byId and rng from concurrent access
	byId map[string]*item
	// "change" events contain a *CollectionChange instance
	bus minibus.Bus
}

func NewCollection(options ...Option) *Collection {
	conf := computeConfig(options...)
	initialItems := make(map[string]*item)
	for k, v := range conf.initialRecords {
		initialItems[k] = &item{body: v, changeTime: conf.clock.Now()}
	}
	conf.initialRecords = nil // so the gc can collect them

	c := &Collection{
		config: conf,
		byId:   initialItems,
		mu:     sync.RWMutex{},
	}

	return c
}

// Get will find the entry with the given ID. If no such entry exists, returns false.
func (c *Collection) Get(id string, opts ...ReadOption) (proto.Message, bool) {
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
func (c *Collection) List(opts ...ReadOption) []proto.Message {
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
func (c *Collection) Add(id string, body proto.Message) (proto.Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	body, _, err := c.add(id, func(id string) proto.Message {
		return body
	})
	return body, err
}

// AddFn adds an entry to the collection by invoking create with a newly allocated ID.
func (c *Collection) AddFn(create CreateFn) (proto.Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	body, _, err := c.add("", create)
	return body, err
}

func (c *Collection) add(id string, create CreateFn) (proto.Message, string, error) {
	// todo: convert Collection.Add to use WriteOption

	if id == "" {
		var err error
		if id, err = c.genID(); err != nil {
			return nil, "", err
		}
	} else {
		if _, exists := c.byId[id]; exists {
			return nil, "", status.Errorf(codes.AlreadyExists, "%s cannot be created, already exists", id)
		}
	}

	body := create(id)
	c.byId[id] = &item{body: body, changeTime: c.clock.Now()}
	c.bus.Send(context.TODO(), &CollectionChange{
		Id:         id,
		ChangeTime: c.clock.Now(), // todo: allow specifying the writeTime, as part of using WriteOption
		ChangeType: types.ChangeType_ADD,
		NewValue:   body,
	})

	return body, id, nil
}

func (c *Collection) Update(id string, msg proto.Message, opts ...WriteOption) (proto.Message, error) {
	writeRequest := computeWriteConfig(opts...)
	writer := writeRequest.fieldUpdater(c.writableFields)
	if err := writer.Validate(msg); err != nil {
		return nil, err
	}

	var created proto.Message // during create, this is returned by GetFn so concurrent reference checks pass
	oldValue, newValue, err := GetAndUpdate(
		&c.mu,
		func() (item proto.Message, err error) {
			if created != nil {
				return created, nil
			}

			// handle empty ids, generating them, and invoking callbacks
			if id == "" && writeRequest.genEmptyID {
				id, err = c.genID()
				if err != nil {
					return nil, err
				}
				if writeRequest.idCallback != nil {
					writeRequest.idCallback(id)
				}
			}

			val, exists := c.byId[id]
			if exists {
				return val.body, nil
			}
			if !writeRequest.createIfAbsent {
				return nil, status.Errorf(codes.NotFound, "id %v not found", id)
			}
			created = msg.ProtoReflect().New().Interface()
			if writeRequest.createdCallback != nil {
				writeRequest.createdCallback()
			}
			return created, nil
		},
		writeRequest.changeFn(writer, msg),
		func(msg proto.Message) {
			c.byId[id] = &item{body: msg, changeTime: writeRequest.updateTime(c.clock)}
		})

	if err != nil {
		if s, ok := status.FromError(err); ok {
			return nil, status.Errorf(s.Code(), "%v %v", s.Message(), id)
		}
		return nil, err
	}
	changeType := types.ChangeType_UPDATE
	if oldValue == nil {
		changeType = types.ChangeType_ADD
	}
	c.bus.Send(context.TODO(), &CollectionChange{
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
func (c *Collection) Delete(id string) proto.Message {
	c.mu.Lock()
	defer c.mu.Unlock()

	oldVal, exists := c.byId[id]
	if !exists {
		return nil
	}
	delete(c.byId, id)
	c.bus.Send(context.TODO(), &CollectionChange{
		Id:         id,
		ChangeTime: c.clock.Now(),
		ChangeType: types.ChangeType_REMOVE,
		OldValue:   oldVal.body,
	})
	return oldVal.body
}

func (c *Collection) Pull(ctx context.Context, opts ...ReadOption) <-chan *CollectionChange {
	readConfig := computeReadConfig(opts...)
	filter := readConfig.ResponseFilter()

	emit, currentValues := c.onUpdate(ctx, readConfig)
	send := make(chan *CollectionChange)

	go func() {
		defer close(send)

		if len(currentValues) > 0 {
			for _, value := range currentValues {
				change := &CollectionChange{
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
			change := event.(*CollectionChange)
			change, ok := change.include(readConfig.include)
			if !ok {
				continue
			}
			change = change.filter(filter)
			if c.equivalence != nil && c.equivalence.Compare(change.OldValue, change.NewValue) {
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
func (c *Collection) PullID(ctx context.Context, id string, opts ...ReadOption) <-chan *ValueChange {
	send := make(chan *ValueChange)
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
			if change.NewValue == nil {
				log.Printf("WARN: CollectionChange.NewValue is nil, but not a REMOVE change! %v", change)
				return
			}

			select {
			case <-ctx.Done():
				return
			case send <- &ValueChange{ChangeTime: change.ChangeTime, Value: change.NewValue}:
			}
		}
	}()
	return send
}

func (c *Collection) onUpdate(ctx context.Context, config *readRequest) (<-chan interface{}, []idItem) {
	var res []idItem
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
func (c *Collection) Clock() Clock {
	return c.clock
}

// itemSlice returns all the values in byId adjusted to match readConfig settings like readRequest.include.
func (c *Collection) itemSlice(readConfig *readRequest) []idItem {
	res := make([]idItem, 0, len(c.byId))
	for id, value := range c.byId {
		if readConfig.Exclude(id, value.body) {
			continue
		}
		res = append(res, idItem{item: *value, id: id})
	}
	return res
}

func (c *Collection) genID() (string, error) {
	return GenerateUniqueId(c.rng, func(candidate string) bool {
		_, exists := c.byId[candidate]
		return exists
	})
}

type item struct {
	body       proto.Message
	changeTime time.Time
}

type idItem struct {
	item
	id string
}
