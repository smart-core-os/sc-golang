package memory

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

func (s *PowerSupplyApi) setNotified(notified float32) {
	_, _ = s.powerCapacity.Set(&traits.PowerCapacity{Notified: notified}, WithUpdatePaths("notified"))
}

func (s *PowerSupplyApi) addNotified(notified float32) {
	_, _ = s.powerCapacity.Set(
		&traits.PowerCapacity{Notified: notified},
		WithUpdatePaths("notified"),
		InterceptBefore(func(old, change proto.Message) {
			val := old.(*traits.PowerCapacity)
			delta := change.(*traits.PowerCapacity)
			delta.Notified += val.Notified
		}),
	)
}

func (s *PowerSupplyApi) normaliseMaxDraw(n *traits.DrawNotification) {
	capacity := s.powerCapacity.Get().(*traits.PowerCapacity)
	available := capacity.Free - capacity.Notified
	if available < 0 {
		available = 0
	}
	if n.MaxDraw > available { // can't satisfy the full requested power
		if n.MinDraw == 0 || n.MinDraw > available { // can't satisfy the minimum requested power
			n.MaxDraw = 0
		} else {
			n.MaxDraw = available
		}
	}
}

func (s *PowerSupplyApi) normaliseRampDuration(n *traits.DrawNotification) {
	if n.RampDuration == nil {
		n.RampDuration = durationpb.New(s.defaultRampDuration)
	}
	if n.RampDuration.AsDuration() > s.maxRampDuration {
		n.RampDuration = durationpb.New(s.maxRampDuration)
	}
}

// generateId assigns a unique id to the given DrawNotification.
// s.notificationsByIdMu must be locked before calling.
func (s *PowerSupplyApi) generateId(n *traits.DrawNotification) error {
	id, err := generateUniqueId(s.Rng, func(candidate string) bool {
		_, ok := s.notificationsById[candidate]
		return ok
	})
	if err != nil {
		return err
	}
	n.Id = id
	return nil
}

// setDrawNotification adds n to the set of known notifications.
// The notification will be removed and n.MaxDraw will be subtracted from the current capacity
// after n.RampDuration time.
func (s *PowerSupplyApi) setDrawNotification(n *traits.DrawNotification) (*traits.DrawNotification, error) {
	id := n.Id
	notifiedValue := n.MaxDraw

	// Without a buffer of 1 there's a race condition where the abort func could be called
	// before the go routine has started watching.
	// The race is replaced with a small increase in memory - though I think struct{} chans use counters not pointers
	abort := make(chan struct{}, 1)
	ctx, stop := context.WithTimeout(context.Background(), n.RampDuration.AsDuration())
	go func() {
		select {
		case <-ctx.Done():
			// clean up state changes
			s.addNotified(-notifiedValue)
			// clean up the entry in the map
			s.notificationsByIdMu.Lock()
			defer s.notificationsByIdMu.Unlock()
			if _, ok := s.notificationsById[id]; ok {
				delete(s.notificationsById, id)
			}
		case <-abort:
			stop() // clean up timers tracking the timeout
		}
	}()

	s.notificationsById[n.Id] = &drawNotification{
		notification: n,
		cancel:       stop,
		abort: func() {
			select {
			case abort <- struct{}{}:
			default:
			}
		},
	}

	return n, nil
}

func adjustPowerCapacityForLoad(c *traits.PowerCapacity, headroom float32) {
	load := float32(0)
	if c.Load != nil {
		load = *c.Load
	}
	capacity := c.Rating - load
	free := capacity - headroom
	c.Capacity = &capacity
	c.Free = free
}

func validateWriteRequest(request *traits.CreateDrawNotificationRequest) error {
	if request.DrawNotification.Id != "" {
		return status.Error(codes.InvalidArgument, "Id must be unset on create")
	}
	return nil
}

func validateUpdateRequest(request *traits.UpdateDrawNotificationRequest) error {
	if request.DrawNotification.Id == "" {
		return status.Error(codes.InvalidArgument, "Id must be set on update")
	}
	return nil
}

type drawNotification struct {
	cancel       func() // clean up and undo changes made when this notification was created
	abort        func() // clean up without undoing state changes
	notification *traits.DrawNotification
}
