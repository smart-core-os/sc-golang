package memory

import (
	"github.com/smart-core-os/sc-golang/internal/clock"
	"io"
	"math/rand"
	"sort"
	"sync"

	"github.com/olebedev/emitter"
	"github.com/smart-core-os/sc-api/go/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CollectionOption func(collection *Collection)

var DefaultCollectionOptions = []CollectionOption{
	WithClockCollection(clock.Real()),
	WithRandom(nil), // create a new rng for each Collection instance
}

type Collection struct {
	mu   sync.RWMutex // protects byId and rng from concurrent access
	byId map[string]*item
	// rng is used to generate new ids.
	// It should be rand.Rand or crypto.Rand. The reader shouldn't close
	rng io.Reader
	// "change" events contain a Change instance
	bus   *emitter.Emitter
	clock clock.Clock
}

func NewCollection(options ...CollectionOption) *Collection {
	c := &Collection{
		byId: make(map[string]*item),
		mu:   sync.RWMutex{},
		bus:  emitter.New(0),
	}

	for _, opt := range DefaultCollectionOptions {
		opt(c)
	}
	for _, opt := range options {
		opt(c)
	}

	return c
}

// Get will find the entry with the given ID. If no such entry exists, returns false.
func (c *Collection) Get(id string) (proto.Message, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.byId[id]
	if !ok {
		return nil, false
	}

	return entry.body, true
}

// List returns a list of all the entries, sorted by their ID.
func (c *Collection) List() []proto.Message {
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
	for _, e := range tmp {
		result = append(result, e.body)
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
	c.bus.Emit("change", Change{
		ChangeTime: c.now(),
		ChangeType: types.ChangeType_ADD,
		NewValue:   body,
	})

	return body, id, nil
}

// Update allows the given ChangeFn to update the item with the given id.
func (c *Collection) Update(id string, fn ChangeFn) (proto.Message, error) {
	oldValue, newValue, err := GetAndUpdate(
		&c.mu,
		func() (proto.Message, error) {
			val, exists := c.byId[id]
			if !exists {
				return nil, status.Errorf(codes.NotFound, "id %v not found", id)
			}
			return val.body, nil
		},
		fn,
		func(message proto.Message) {
			c.byId[id] = &item{body: message}
		})

	if err != nil {
		if s, ok := status.FromError(err); ok {
			return nil, status.Errorf(s.Code(), "%v %v", s.Message(), id)
		}
		return nil, err
	}
	c.bus.Emit("change", Change{
		ChangeTime: c.now(),
		ChangeType: types.ChangeType_UPDATE,
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
		ChangeTime: c.now(),
		ChangeType: types.ChangeType_REMOVE,
		OldValue:   oldVal.body,
	})
	return oldVal.body
}

func (c *Collection) genID() (string, error) {
	return GenerateUniqueId(c.rng, func(candidate string) bool {
		_, exists := c.byId[candidate]
		return exists
	})
}

func (c *Collection) now() *timestamppb.Timestamp {
	return timestamppb.New(c.clock.Now())
}

type item struct {
	body proto.Message
}

type Change struct {
	ChangeTime *timestamppb.Timestamp
	ChangeType types.ChangeType
	OldValue   proto.Message
	NewValue   proto.Message
}

// TODO: figure out a way to resolve this naming collision
func WithClockCollection(clk clock.Clock) CollectionOption {
	return func(collection *Collection) {
		collection.clock = clk
	}
}

// WithRandom sets the source of randomness used for generating IDs.
// The Collection will not call Read on the rng concurrently.
// If rng is nil, a new rand.Rand will be created and used.
func WithRandom(rng io.Reader) CollectionOption {
	return func(collection *Collection) {
		if rng == nil {
			rng = rand.New(rand.NewSource(rand.Int63()))
		}
		collection.rng = rng
	}
}
