package resource

import (
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// CreateFn is called to generate a message based on the ID the message is going to have.
type CreateFn[T Message] func(id string) T

// GetFn is called to retrieve the message from the external store.
type GetFn[T Message] func() (item T, err error)

// ChangeFn is called to apply changes to the new proto.Message.
type ChangeFn[T Message] func(old, new T) error

// SaveFn is called to save the message in the external store.
type SaveFn[T Message] func(msg T)

// GetAndUpdate applies an atomic get and update operation in the context of proto messages.
// mu.RLock will be held during the get call.
// mu.Lock will be held during the save call.
// No locks will be held during the change call.
//
// An error will be returned if the value returned by get changes during the change call.
func GetAndUpdate[T Message](mu *sync.RWMutex, get GetFn[T], change ChangeFn[T],
	save SaveFn[T]) (oldValue T, newValue T, err error) {

	mu.RLock()
	oldValue, err = get()
	mu.RUnlock()
	if err != nil {
		return zero[T](), zero[T](), err
	}

	newValue = proto.Clone(oldValue).(T)
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
