package resource

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/olebedev/emitter"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type Collection struct {
	*config

	mu   sync.RWMutex // protects byId and rng from concurrent access
	byId map[string]*item
	// "change" events contain a Change instance
	bus *emitter.Emitter
}

func NewCollection(options ...Option) *Collection {
	conf := computeConfig(options...)
	c := &Collection{
		config: conf,
		byId:   make(map[string]*item),
		mu:     sync.RWMutex{},
		bus:    emitter.New(0),
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

	// temporary slice to allow sorting
	type entry struct {
		id   string
		body proto.Message
	}
	tmp := make([]entry, 0, len(c.byId))

	for id, value := range c.byId {
		tmp = append(tmp, entry{id, value.body})
	}
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
func (c *Collection) Add(id string, body proto.Message) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, id, err := c.add(id, func(id string) proto.Message {
		return body
	})
	return id, err
}

// AddFn adds an entry to the collection by invoking create with a newly allocated ID.
func (c *Collection) AddFn(create CreateFn) (proto.Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	body, _, err := c.add("", create)
	return body, err
}

func (c *Collection) add(id string, create CreateFn) (proto.Message, string, error) {
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
	c.byId[id] = &item{body: body}
	c.bus.Emit("change", &Change{
		Id:         id,
		ChangeTime: c.clock.Now(),
		ChangeType: types.ChangeType_ADD,
		NewValue:   body,
	})

	return body, id, nil
}

// Update allows the given ChangeFn to update the item with the given id.
func (c *Collection) Update(id string, fn ChangeFn) (proto.Message, error) {
	return c.UpdateOrCreate(id, fn, func(id string) proto.Message {
		return nil // nil means do not create, and results in NOT_FOUND error
	})
}

// UpdateOrCreate allows the given ChangeFn to update the item with the given id, creating a new item if needed.
// If CreateFn returns nil, and no existing value with id, an error representing NotFound will be returned.
func (c *Collection) UpdateOrCreate(id string, change ChangeFn, create CreateFn) (proto.Message, error) {
	var created proto.Message // so multiple gets return the same instance
	oldValue, newValue, err := GetAndUpdate(
		&c.mu,
		func() (proto.Message, error) {
			if created != nil {
				return created, nil
			}
			val, exists := c.byId[id]
			if !exists {
				created = create(id)
				if created == nil {
					return nil, status.Errorf(codes.NotFound, "id %v not found", id)
				}
				return created, nil
			}
			return val.body, nil
		},
		change,
		func(message proto.Message) {
			c.byId[id] = &item{body: message}
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
	c.bus.Emit("change", &Change{
		Id:         id,
		ChangeTime: c.clock.Now(),
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
	c.bus.Emit("change", &Change{
		Id:         id,
		ChangeTime: c.clock.Now(),
		ChangeType: types.ChangeType_REMOVE,
		OldValue:   oldVal.body,
	})
	return oldVal.body
}

func (c *Collection) Pull(ctx context.Context, opts ...ReadOption) <-chan *Change {
	readConfig := computeReadConfig(opts...)
	filter := readConfig.ResponseFilter()

	emit := c.bus.On("change")
	send := make(chan *Change)

	go func() {
		defer c.bus.Off("change", emit)
		defer close(send)

		for {
			select {
			case <-ctx.Done():
				return
			case event := <-emit:
				change := event.Args[0].(*Change).filter(filter)
				if c.equivalence != nil && c.equivalence.Compare(change.OldValue, change.NewValue) {
					continue
				}
				select {
				case send <- change:
				case <-ctx.Done():
					return
				}

			}
		}
	}()

	return send
}

func (c *Collection) genID() (string, error) {
	return GenerateUniqueId(c.rng, func(candidate string) bool {
		_, exists := c.byId[candidate]
		return exists
	})
}

type item struct {
	body proto.Message
}

type Change struct {
	Id         string
	ChangeTime time.Time
	ChangeType types.ChangeType
	OldValue   proto.Message
	NewValue   proto.Message
}

func (c *Change) filter(filter *masks.ResponseFilter) *Change {
	newNewValue := filter.FilterClone(c.NewValue)
	newOldValue := filter.FilterClone(c.OldValue)
	if newNewValue == c.NewValue && newOldValue == c.OldValue {
		return c
	}
	return &Change{
		Id:         c.Id,
		ChangeType: c.ChangeType,
		ChangeTime: c.ChangeTime,
		OldValue:   newOldValue,
		NewValue:   newNewValue,
	}
}
