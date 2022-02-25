package resource

import (
	"google.golang.org/protobuf/proto"
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
