package powersupply

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/olebedev/emitter"
	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/memory"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:generate protoc -I. --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. page_token.proto

// MemoryDevice is an in-memory implementation of PowerSupplyApiServer scoped to a single device.
type MemoryDevice struct {
	traits.UnimplementedPowerSupplyApiServer
	UnimplementedMemorySettingsApiServer

	powerCapacity *memory.Resource   // of *traits.PowerCapacity
	settings      *memory.Resource   // of *MemorySettings
	notifications *memory.Collection // of *drawNotification

	notificationsById   map[string]*drawNotification
	notificationsByIdMu sync.RWMutex
	// "change" event args are *traits.PullNotificationsResponse_Change
	bus *emitter.Emitter
	// used for generating ids
	Rng *rand.Rand
}

func NewMemoryDevice() *MemoryDevice {
	initialPowerCapacity := InitialPowerCapacity()
	return &MemoryDevice{
		powerCapacity: memory.NewResource(
			memory.WithInitialValue(initialPowerCapacity),
		),
		settings: memory.NewResource(
			memory.WithInitialValue(&MemorySettings{
				Rating:              initialPowerCapacity.Rating,
				Load:                *initialPowerCapacity.Load,
				Voltage:             initialPowerCapacity.Voltage,
				Reserved:            0,
				MaxRampDuration:     durationpb.New(10 * time.Minute),
				DefaultRampDuration: durationpb.New(30 * time.Second),
			}),
		),
		notificationsById: make(map[string]*drawNotification),
		bus:               &emitter.Emitter{},
		Rng:               rand.New(rand.NewSource(rand.Int63())),
	}
}

func InitialPowerCapacity() *traits.PowerCapacity {
	c := &traits.PowerCapacity{
		Rating:  60,
		Voltage: 240,
		Load:    new(float32),
	}
	adjustPowerCapacityForLoad(c, 0)
	return c
}

func (s *MemoryDevice) SetLoad(load float32) {
	_, err := s.UpdateSettings(context.Background(), &UpdateMemorySettingsReq{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"load"}},
		Settings:   &MemorySettings{Load: load},
	})
	if err != nil {
		log.Printf("SetLoad: %v", err)
	}
}

func (s *MemoryDevice) GetPowerCapacity(_ context.Context, _ *traits.GetPowerCapacityRequest) (*traits.PowerCapacity, error) {
	return s.powerCapacity.Get().(*traits.PowerCapacity), nil
}

func (s *MemoryDevice) PullPowerCapacity(request *traits.PullPowerCapacityRequest, server traits.PowerSupplyApi_PullPowerCapacityServer) error {
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

func (s *MemoryDevice) ListDrawNotifications(_ context.Context, request *traits.ListDrawNotificationsRequest) (*traits.ListDrawNotificationsResponse, error) {
	s.notificationsByIdMu.RLock()
	defer s.notificationsByIdMu.RUnlock()

	pageToken := &types.PageToken{}
	if err := decodePageToken(request.PageToken, pageToken); err != nil {
		return nil, err
	}

	lastKey := pageToken.GetLastResourceName() // the key() of the last item we sent
	pageSize := capPageSize(int(request.GetPageSize()))

	sortedNotifications := make([]*drawNotification, 0, len(s.notificationsById))
	for _, notification := range s.notificationsById {
		sortedNotifications = append(sortedNotifications, notification)
	}
	sort.Slice(sortedNotifications, func(i, j int) bool {
		left, right := sortedNotifications[i].key(), sortedNotifications[j].key()
		return left < right
	})
	nextIndex := 0
	if lastKey != "" {
		nextIndex = sort.Search(len(sortedNotifications), func(i int) bool {
			return sortedNotifications[i].key() > lastKey
		})
	}

	result := &traits.ListDrawNotificationsResponse{
		TotalSize: int32(len(sortedNotifications)),
	}
	upperBound := nextIndex + pageSize
	if upperBound > len(sortedNotifications) {
		upperBound = len(sortedNotifications)
		pageToken = nil
	} else {
		pageToken.PageStart = &types.PageToken_LastResourceName{
			LastResourceName: sortedNotifications[upperBound-1].key(),
		}
	}

	var err error
	result.NextPageToken, err = encodePageToken(pageToken)
	if err != nil {
		return nil, err
	}
	for _, n := range sortedNotifications[nextIndex:upperBound] {
		result.DrawNotifications = append(result.DrawNotifications, n.notification)
	}
	return result, nil
}

func (s *MemoryDevice) CreateDrawNotification(_ context.Context, request *traits.CreateDrawNotificationRequest) (*traits.DrawNotification, error) {
	if err := validateWriteRequest(request); err != nil {
		return nil, err
	}

	log.Printf("CreateDrawNotification")
	n := request.DrawNotification
	s.normaliseMaxDraw(n)
	s.normaliseRampDuration(n)
	n.NotificationTime = timestamppb.Now()

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

	n, err := s.setDrawNotification(n)
	if err != nil {
		return nil, err
	}
	s.bus.Emit("change", &traits.PullDrawNotificationsResponse_Change{
		Name:       request.Name,
		ChangeTime: timestamppb.Now(),
		Type:       types.ChangeType_ADD,
		NewValue:   n,
	})
	return n, err
}

func (s *MemoryDevice) UpdateDrawNotification(ctx context.Context, request *traits.UpdateDrawNotificationRequest) (*traits.DrawNotification, error) {
	if err := validateUpdateRequest(request); err != nil {
		return nil, err
	}

	n := request.DrawNotification
	s.normaliseMaxDraw(n)
	s.normaliseRampDuration(n)

	if n.MaxDraw == 0 {
		if _, err := s.DeleteDrawNotification(ctx, &traits.DeleteDrawNotificationRequest{Name: request.Name, Id: n.Id, AllowMissing: true}); err != nil {
			return nil, err
		}
		return n, nil
	}

	s.notificationsByIdMu.Lock()
	defer s.notificationsByIdMu.Unlock()

	notifiedValue := n.MaxDraw
	oldRecord, ok := s.notificationsById[n.Id]
	if !ok {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Notification %v not found, it may have expired. Try creating a new one", n.Id))
	}

	// adjust the notified value of the capacity to match the new value
	oldRecord.abort()
	s.addNotified(notifiedValue - oldRecord.notification.MaxDraw)

	n, err := s.setDrawNotification(n)
	if err != nil {
		return nil, err
	}
	s.bus.Emit("change", &traits.PullDrawNotificationsResponse_Change{
		Name:       request.Name,
		ChangeTime: timestamppb.Now(),
		Type:       types.ChangeType_UPDATE,
		OldValue:   oldRecord.notification,
		NewValue:   n,
	})
	return n, err
}

func (s *MemoryDevice) DeleteDrawNotification(_ context.Context, request *traits.DeleteDrawNotificationRequest) (*emptypb.Empty, error) {
	s.notificationsByIdMu.Lock()
	defer s.notificationsByIdMu.Unlock()

	n, ok := s.notificationsById[request.Id]
	if !ok {
		if request.AllowMissing {
			return &emptypb.Empty{}, nil
		}
		return nil, status.Errorf(codes.NotFound, "%v does not exist", request.Id)
	}
	n.cancel()
	delete(s.notificationsById, request.Id)
	s.bus.Emit("change", &traits.PullDrawNotificationsResponse_Change{
		Name:       request.Name,
		ChangeTime: timestamppb.Now(),
		Type:       types.ChangeType_REMOVE,
		OldValue:   n.notification,
	})
	return &emptypb.Empty{}, nil
}

func (s *MemoryDevice) PullDrawNotifications(req *traits.PullDrawNotificationsRequest, server traits.PowerSupplyApi_PullDrawNotificationsServer) error {
	changes := s.bus.On("change")
	defer s.bus.Off("change", changes)

	for {
		select {
		case <-server.Context().Done():
			return server.Context().Err()
		case e := <-changes:
			change := e.Args[0].(*traits.PullDrawNotificationsResponse_Change)
			if change.Name == "" {
				change.Name = req.Name
			}
			err := server.Send(&traits.PullDrawNotificationsResponse{
				Changes: []*traits.PullDrawNotificationsResponse_Change{change},
			})
			if err != nil {
				return err
			}
		}
	}
}
