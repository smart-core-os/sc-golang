package resource

import (
	"time"

	"github.com/smart-core-os/sc-api/go/types"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-golang/pkg/masks"
)

// Comparer compares two messages for equivalence.
// This is used during the Pull operation to de-duplicate consecutive emissions.
type Comparer func(x, y proto.Message) bool

// ValueChange contains information about a change to a Value.
type ValueChange struct {
	Value      proto.Message
	ChangeTime time.Time
}

func (v *ValueChange) filter(filter *masks.ResponseFilter) *ValueChange {
	newValue := filter.FilterClone(v.Value)
	if newValue == v.Value {
		return v
	}
	return &ValueChange{Value: newValue, ChangeTime: v.ChangeTime}
}

// CollectionChange contains information about a change to a Collection.
type CollectionChange struct {
	Id         string
	ChangeTime time.Time
	ChangeType types.ChangeType
	OldValue   proto.Message
	NewValue   proto.Message
}

func (c *CollectionChange) filter(filter *masks.ResponseFilter) *CollectionChange {
	newNewValue := filter.FilterClone(c.NewValue)
	newOldValue := filter.FilterClone(c.OldValue)
	if newNewValue == c.NewValue && newOldValue == c.OldValue {
		return c
	}
	return &CollectionChange{
		Id:         c.Id,
		ChangeType: c.ChangeType,
		ChangeTime: c.ChangeTime,
		OldValue:   newOldValue,
		NewValue:   newNewValue,
	}
}

// include adjusts this change based on any filtering that is active on the collection.
// If items are being filtered, then an UPDATE that causes an items inclusion to change will report an ADD or REMOVE
// as needed. A new CollectionChange is returned if the underlying change type isn't accurate anymore.
// The ok return value will be true if an update should be sent, and false if the change shouldn't be forwarded on.
func (c *CollectionChange) include(includeFunc FilterFunc) (newChange *CollectionChange, ok bool) {
	if includeFunc == nil {
		return c, true
	}

	oldInclude := includeFunc(c.Id, c.OldValue)
	newInclude := includeFunc(c.Id, c.NewValue)
	if oldInclude == newInclude {
		// the only time we want to skip sending the update is if both the old and new values are excluded
		return c, !newInclude
	}

	if newInclude {
		// treat this like an Add
		return &CollectionChange{
			Id:         c.Id,
			ChangeType: types.ChangeType_ADD,
			ChangeTime: c.ChangeTime,
			NewValue:   c.NewValue,
		}, true
	}

	// treat this like a remove
	return &CollectionChange{
		Id:         c.Id,
		ChangeType: types.ChangeType_REMOVE,
		ChangeTime: c.ChangeTime,
		OldValue:   c.OldValue,
	}, true
}
