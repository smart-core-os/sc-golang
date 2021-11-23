package powersupply

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/memory"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func (s *MemoryDevice) setNotified(notified float32) {
	_, _ = s.powerCapacity.Set(&traits.PowerCapacity{Notified: notified}, memory.WithUpdatePaths("notified"))
}

func (s *MemoryDevice) addNotified(notified float32) {
	log.Printf("addNotified(%v)", notified)
	_, _ = s.powerCapacity.Set(
		&traits.PowerCapacity{Notified: notified},
		memory.WithUpdatePaths("notified"),
		memory.InterceptBefore(func(old, change proto.Message) {
			val := old.(*traits.PowerCapacity)
			delta := change.(*traits.PowerCapacity)
			delta.Notified += val.Notified
		}),
	)
}

func (s *MemoryDevice) addPending(pending float32) {
	log.Printf("addPending(%f)", pending)
	_, _ = s.powerCapacity.Set(
		&traits.PowerCapacity{},
		memory.WithUpdatePaths("pending"),
		memory.InterceptBefore(func(old, new proto.Message) {
			new.(*traits.PowerCapacity).Pending = old.(*traits.PowerCapacity).Pending + pending
		}),
	)
}

func (s *MemoryDevice) normaliseMaxDraw(n *traits.DrawNotification, oldDraw float32) (ok bool) {
	capacity := s.powerCapacity.Get().(*traits.PowerCapacity)
	available := capacity.Free - capacity.Notified + oldDraw
	if available < 0 {
		available = 0
	}
	if n.MaxDraw > available { // can't satisfy the full requested power
		if n.MinDraw == 0 || n.MinDraw > available { // can't satisfy the minimum requested power
			return false
		} else {
			n.MaxDraw = available
		}
	}
	return true
}

func (s *MemoryDevice) normaliseRampDuration(n *traits.DrawNotification) {
	settings := s.readSettings()
	if n.RampDuration == nil {
		n.RampDuration = settings.DefaultRampDuration
	}
	if n.RampDuration.AsDuration() > settings.MaxRampDuration.AsDuration() {
		n.RampDuration = settings.MaxRampDuration
	}
}

// generateId assigns a unique id to the given DrawNotification.
// s.notificationsByIdMu must be locked before calling.
func (s *MemoryDevice) generateId(n *traits.DrawNotification) error {
	id, err := memory.GenerateUniqueId(s.Rng, func(candidate string) bool {
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
func (s *MemoryDevice) setDrawNotification(n *traits.DrawNotification) (*traits.DrawNotification, error) {
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
			log.Printf("reset after CreateDrawNotification")
			// clean up state changes
			s.addNotified(-notifiedValue)
			if n.Pending {
				s.addPending(-notifiedValue)
			}
			// clean up the entry in the map
			s.notificationsByIdMu.Lock()
			defer s.notificationsByIdMu.Unlock()
			if _, ok := s.notificationsById[id]; ok {
				delete(s.notificationsById, id)
			}
			// notify anyone who might be listening
			s.bus.Emit("change", &traits.PullDrawNotificationsResponse_Change{
				Type:       types.ChangeType_REMOVE,
				ChangeTime: timestamppb.Now(),
				OldValue:   n,
			})
		case <-abort:
			log.Printf("abort after CreateDrawNotification")
			stop() // clean up timers tracking the timeout
		}
	}()

	var createTime time.Time
	if oldVal, hasOld := s.notificationsById[n.Id]; hasOld {
		createTime = oldVal.createTime
	}
	if createTime.IsZero() {
		createTime = time.Now()
	}
	s.notificationsById[n.Id] = &drawNotification{
		notification: n,
		createTime:   createTime,
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
	capacity := c.Rating - *c.Load
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

func validateNotification(notification *traits.DrawNotification) error {
	if notification.Force && notification.Pending {
		return status.Error(codes.InvalidArgument, "A notification cannot be both forced and pending")
	}

	return nil
}

type drawNotification struct {
	cancel       func() // clean up and undo changes made when this notification was created
	abort        func() // clean up without undoing state changes
	notification *traits.DrawNotification
	createTime   time.Time // use for deterministic sorting in combination with notification.Id
}

func (d *drawNotification) key() string {
	return fmt.Sprintf("%v-%v", d.createTime.Format(time.RFC3339), d.notification.Id)
}

const (
	defaultPageSize = 50
	maxPageSize     = 1000
)

func capPageSize(pageSize int) int {
	if pageSize == 0 {
		return defaultPageSize
	}
	if pageSize > maxPageSize {
		return maxPageSize
	}
	return pageSize
}

func decodePageToken(token string, pageToken *types.PageToken) error {
	if token != "" {
		tokenBytes, err := base64.StdEncoding.DecodeString(token)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "bad page token: %v", err)
		}
		if err := proto.Unmarshal(tokenBytes, pageToken); err != nil {
			return status.Errorf(codes.InvalidArgument, "bad page token: %v", err)
		}
	}
	return nil
}

func encodePageToken(pageToken *types.PageToken) (string, error) {
	if pageToken != nil {
		tokenBytes, err := proto.Marshal(pageToken)
		if err != nil {
			return "", status.Errorf(codes.Unknown, "unable to create page token: %v", err)
		}
		return base64.StdEncoding.EncodeToString(tokenBytes), nil
	}
	return "", nil
}
