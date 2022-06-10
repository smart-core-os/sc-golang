package resource

import (
	"time"

	"github.com/smart-core-os/sc-api/go/types"

	"github.com/smart-core-os/sc-golang/pkg/masks"
)

// Comparer compares two messages for equivalence.
// This is used during the Pull operation to de-duplicate consecutive emissions.
type Comparer[T Message] func(x, y T) bool

// ValueChange contains information about a change to a Value.
type ValueChange[T Message] struct {
	Value      T
	ChangeTime time.Time
}

func (v *ValueChange[T]) filter(filter *masks.ResponseFilter) *ValueChange[T] {
	newValue := filter.FilterClone(v.Value).(T)
	if newValue == v.Value {
		return v
	}
	return &ValueChange[T]{Value: newValue, ChangeTime: v.ChangeTime}
}

// CollectionChange contains information about a change to a Collection.
type CollectionChange[T Message] struct {
	Id         string
	ChangeTime time.Time
	ChangeType types.ChangeType
	OldValue   T
	NewValue   T
}

func (c *CollectionChange[T]) filter(filter *masks.ResponseFilter) *CollectionChange[T] {
	newNewValue := filter.FilterClone(c.NewValue).(T)
	newOldValue := filter.FilterClone(c.OldValue).(T)
	if newNewValue == c.NewValue && newOldValue == c.OldValue {
		return c
	}
	return &CollectionChange[T]{
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
func (c *CollectionChange[T]) include(includeFunc FilterFunc) (newChange *CollectionChange[T], ok bool) {
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
		return &CollectionChange[T]{
			Id:         c.Id,
			ChangeType: types.ChangeType_ADD,
			ChangeTime: c.ChangeTime,
			NewValue:   c.NewValue,
		}, true
	}

	// treat this like a remove
	return &CollectionChange[T]{
		Id:         c.Id,
		ChangeType: types.ChangeType_REMOVE,
		ChangeTime: c.ChangeTime,
		OldValue:   c.OldValue,
	}, true
}
