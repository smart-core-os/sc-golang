package resource

import (
	"time"

	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Comparer compares two messages for equivalence.
// This interface is used during the Pull operation to de-duplicate consecutive emissions.
type Comparer interface {
	// Compare compares two messages that changed at a specific time.
	// If ok is false then this Comparator did not attempt to compare the two messages, in other words the equal result
	// should be ignored.
	Compare(x, y proto.Message) bool
}

// ComparerFunc converts a func of the correct signature into a Comparer.
type ComparerFunc func(x, y proto.Message) bool

func (c ComparerFunc) Compare(x, y proto.Message) bool {
	return c(x, y)
}

// ValueChange contains information about a change to a Value.
type ValueChange struct {
	Value      proto.Message
	ChangeTime *timestamppb.Timestamp
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
