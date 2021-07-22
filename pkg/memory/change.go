package memory

import (
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// applyChangeOld is the equivalent to a getAndSet operation that handles absent properties.
// mu is the lock that protects the value in the underlying store.
// get is a function that retrieves the value from the store, mu will be RLock during the call.
// Return exists=false if the value does not exist, typically used for collection resources.
// update is a function that should apply changes to new.
// set is a function that should write the new value to the underlying store, my will be Lock during the call.
func applyChangeOld(mu *sync.RWMutex, get func() (item proto.Message, exists bool), update func(old, new proto.Message) error, set func(message proto.Message)) (oldValue proto.Message, newValue proto.Message, err error) {
	mu.RLock()
	oldValue, exists := get()
	mu.RUnlock()
	if !exists {
		return nil, nil, status.Errorf(codes.NotFound, "not found")
	}

	newValue = proto.Clone(oldValue)
	if err := update(oldValue, newValue); err != nil {
		return oldValue, newValue, err
	}

	mu.Lock()
	defer mu.Unlock()
	oldValueAgain, _ := get()
	if oldValue != oldValueAgain {
		return oldValue, newValue, status.Errorf(codes.Aborted, "concurrent update detected")
	}

	set(newValue)
	return oldValue, newValue, nil
}
