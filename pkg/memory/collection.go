package memory

import (
	"io"
	"sync"

	"github.com/olebedev/emitter"
	"github.com/smart-core-os/sc-api/go/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Collection struct {
	byId   map[string]*item
	byIdMu sync.RWMutex
	// Rng is used to generate new ids.
	// It should be rand.Rand or crypto.Rand. The reader shouldn't close
	Rng io.Reader
	Now func() *timestamppb.Timestamp
	// "change" events contain a Change instance
	bus *emitter.Emitter
}

// Add associates the given body with the id.
// If id already exists then an error is returned.
func (c *Collection) Add(id string, body proto.Message) (string, error) {
	c.byIdMu.Lock()
	defer c.byIdMu.Unlock()

	if id == "" {
		var err error
		if id, err = c.genID(); err != nil {
			return "", err
		}
	} else {
		if _, exists := c.byId[id]; exists {
			return "", status.Errorf(codes.AlreadyExists, "%s cannot be created, already exists", id)
		}
	}

	c.byId[id] = &item{body: body}
	c.bus.Emit("change", Change{
		ChangeTime: c.now(),
		ChangeType: types.ChangeType_ADD,
		NewValue:   body,
	})

	return id, nil
}

// Update allows the given ChangeFn to update the item with the given id.
func (c *Collection) Update(id string, fn ChangeFn) (proto.Message, error) {
	oldValue, newValue, err := GetAndUpdate(
		&c.byIdMu,
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
	c.byIdMu.Lock()
	defer c.byIdMu.Unlock()

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
	return GenerateUniqueId(c.Rng, func(candidate string) bool {
		_, exists := c.byId[candidate]
		return exists
	})
}

func (c *Collection) now() *timestamppb.Timestamp {
	if c.Now != nil {
		return c.Now()
	} else {
		return timestamppb.Now()
	}
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
