package memory

import (
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// applyChange is the equivalent to a getAndSet operation that handles absent properties.
func applyChange(mu *sync.RWMutex, get func() (item proto.Message, exists bool), update func(proto.Message) error, set func(message proto.Message)) (oldValue proto.Message, newValue proto.Message, err error) {
	mu.RLock()
	oldValue, exists := get()
	mu.RUnlock()
	if !exists {
		return nil, nil, status.Errorf(codes.NotFound, "not found")
	}

	newValue = proto.Clone(oldValue)
	if err := update(newValue); err != nil {
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
