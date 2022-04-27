package enterleavesensor

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// Model provides the data structure for representing an EnterLeaveSensor trait.
type Model struct {
	enterLeaveEvents *resource.Value // of *traits.EnterLeaveEvent
}

// NewModel constructs a new model with no prior enter or leave information.
func NewModel() *Model {
	return &Model{
		enterLeaveEvents: resource.NewValue(resource.WithInitialValue(&traits.EnterLeaveEvent{})),
	}
}

// CreateEnterLeaveEvent creates and publishes a new EnterLeaveEvent to Pull subscribers.
// The last call to CreateEnterLeaveEvent is the one that is used as the current state if Pull is called with a false
// updates_only field.
func (m *Model) CreateEnterLeaveEvent(event *traits.EnterLeaveEvent, opts ...resource.WriteOption) error {
	_, err := m.enterLeaveEvents.Set(event, opts...)
	return err
}

type EnterLeaveEventChange struct {
	ChangeTime time.Time
	Value      *traits.EnterLeaveEvent
}

// PullEnterLeaveEvents subscribes to changes in the enter leave sensor resource.
// The returned chan will be closed when the given context is Done.
func (m *Model) PullEnterLeaveEvents(ctx context.Context, opts ...resource.ReadOption) <-chan EnterLeaveEventChange {
	send := make(chan EnterLeaveEventChange)
	go func() {
		defer close(send)
		for change := range m.enterLeaveEvents.Pull(ctx, opts...) {
			select {
			case <-ctx.Done():
				return
			case send <- EnterLeaveEventChange{ChangeTime: change.ChangeTime, Value: change.Value.(*traits.EnterLeaveEvent)}:
			}
		}
	}()
	return send
}
