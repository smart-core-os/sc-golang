package memory

import (
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// getFn is called to retrieve the message from the external store.
// Return exists=false if no such item is present
type getFn func() (item proto.Message, exists bool)

// changeFn is called to apply changes to the new proto.Message.
type changeFn func(old, new proto.Message) error

// saveFn is called to save the message in the external store.
type saveFn func(msg proto.Message)

// getAndUpdate applies an atomic get and update operation in the context of proto messages.
// mu.RLock will be held during the get call.
// mu.Lock will be held during the save call.
// No locks will be held during the change call.
func getAndUpdate(mu *sync.RWMutex, get getFn, change changeFn, save saveFn) (oldValue proto.Message, newValue proto.Message, err error) {
	mu.RLock()
	oldValue, exists := get()
	mu.RUnlock()
	if !exists {
		return nil, nil, status.Errorf(codes.NotFound, "not found")
	}

	newValue = proto.Clone(oldValue)
	if err := change(oldValue, newValue); err != nil {
		return oldValue, newValue, err
	}

	mu.Lock()
	defer mu.Unlock()
	oldValueAgain, _ := get()
	if oldValue != oldValueAgain {
		return oldValue, newValue, status.Errorf(codes.Aborted, "concurrent update detected")
	}

	save(newValue)
	return oldValue, newValue, nil
}
