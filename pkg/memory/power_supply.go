package memory

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

// PowerSupplyApi is an in-memory implementation of PowerSupplyApiServer scoped to a single device.
type PowerSupplyApi struct {
	traits.UnimplementedPowerSupplyApiServer

	powerCapacity *Resource

	reserved            float32       // the difference between capacity and free
	maxRampDuration     time.Duration // caps write requests
	defaultRampDuration time.Duration // applies when writes don't include the value

	notificationsById   map[string]*drawNotification
	notificationsByIdMu sync.RWMutex
	// used for generating ids
	Rng *rand.Rand
}

func NewPowerSupplyApi() *PowerSupplyApi {
	return &PowerSupplyApi{
		reserved:            0,
		maxRampDuration:     10 * time.Minute,
		defaultRampDuration: 30 * time.Second,

		powerCapacity: NewResource(
			WithInitialValue(InitialPowerCapacity()),
		),
		notificationsById: make(map[string]*drawNotification),
		Rng:               rand.New(rand.NewSource(rand.Int63())),
	}
}

func InitialPowerCapacity() *traits.PowerCapacity {
	c := &traits.PowerCapacity{
		Rating:  60,
		Voltage: 240,
	}
	adjustPowerCapacityForLoad(c, 0)
	return c
}

func (s *PowerSupplyApi) SetLoad(load float32) {
	_, _ = s.powerCapacity.Set(&traits.PowerCapacity{Load: &load}, WithUpdatePaths("load"), InterceptAfter(func(old, new proto.Message) {
		newCapacity := new.(*traits.PowerCapacity)
		adjustPowerCapacityForLoad(newCapacity, s.reserved)
	}))
}

func (s *PowerSupplyApi) GetPowerCapacity(_ context.Context, _ *traits.GetPowerCapacityRequest) (*traits.PowerCapacity, error) {
	return s.powerCapacity.Get().(*traits.PowerCapacity), nil
}

func (s *PowerSupplyApi) PullPowerCapacity(request *traits.PullPowerCapacityRequest, server traits.PowerSupplyApi_PullPowerCapacityServer) error {
	changes, done := s.powerCapacity.OnUpdate(server.Context())
	defer done()

	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case change := <-changes:
			typedChange := &traits.PullPowerCapacityResponse_Change{
				Name:              request.Name,
				AvailableCapacity: change.Value.(*traits.PowerCapacity),
				ChangeTime:        change.ChangeTime,
			}
			err := server.Send(&traits.PullPowerCapacityResponse{
				Changes: []*traits.PullPowerCapacityResponse_Change{typedChange},
			})
			if err != nil {
				return err
			}
		}
	}
}

func (s *PowerSupplyApi) CreateDrawNotification(_ context.Context, request *traits.CreateDrawNotificationRequest) (*traits.DrawNotification, error) {
	if err := validateWriteRequest(request); err != nil {
		return nil, err
	}

	n := request.DrawNotification
	s.normaliseMaxDraw(n)
	s.normaliseRampDuration(n)

	if n.MaxDraw == 0 {
		// we don't need to save this notification as it won't be drawing anything
		return n, nil
	}

	s.notificationsByIdMu.Lock()
	defer s.notificationsByIdMu.Unlock()

	notifiedValue := n.MaxDraw
	if err := s.generateId(n); err != nil {
		return nil, err
	}
	s.addNotified(notifiedValue)

	return s.setDrawNotification(n)
}

func (s *PowerSupplyApi) UpdateDrawNotification(ctx context.Context, request *traits.UpdateDrawNotificationRequest) (*traits.DrawNotification, error) {
	if err := validateUpdateRequest(request); err != nil {
		return nil, err
	}

	n := request.DrawNotification
	s.normaliseMaxDraw(n)
	s.normaliseRampDuration(n)

	if n.MaxDraw == 0 {
		if _, err := s.DeleteDrawNotification(ctx, &traits.DeleteDrawNotificationRequest{Name: request.Name, Id: n.Id}); err != nil {
			return nil, err
		}
		return n, nil
	}

	s.notificationsByIdMu.Lock()
	defer s.notificationsByIdMu.Unlock()

	notifiedValue := n.MaxDraw
	if oldRecord, ok := s.notificationsById[n.Id]; ok {
		// adjust the notified value of the capacity to match the new value
		oldRecord.abort()
		s.addNotified(notifiedValue - oldRecord.notification.MaxDraw)
	} else {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Notification %v not found, it may have expired. Try creating a new one", n.Id))
	}

	return s.setDrawNotification(n)
}

func (s *PowerSupplyApi) DeleteDrawNotification(_ context.Context, request *traits.DeleteDrawNotificationRequest) (*emptypb.Empty, error) {
	s.notificationsByIdMu.Lock()
	defer s.notificationsByIdMu.Unlock()

	if n, ok := s.notificationsById[request.Id]; ok {
		n.cancel()
		delete(s.notificationsById, request.Id)
	}
	return &emptypb.Empty{}, nil
}
