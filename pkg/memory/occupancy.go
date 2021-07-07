package memory

import (
	"context"
	"log"
	"math/rand"
	goTime "time"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type OccupancySensorApi struct {
	traits.UnimplementedOccupancySensorApiServer
	occupancy *Resource
}

func NewOccupancyApi(initialState *traits.Occupancy) *OccupancySensorApi {
	return &OccupancySensorApi{
		occupancy: NewResource(WithInitialValue(initialState)),
	}
}

func (o *OccupancySensorApi) Register(server *grpc.Server) {
	traits.RegisterOccupancySensorApiServer(server, o)
}

// SetOccupancy updates the known occupancy state for this device
func (o *OccupancySensorApi) SetOccupancy(_ context.Context, occupancy *traits.Occupancy) {
	_, _ = o.occupancy.Update(occupancy, nil)
}

func (o *OccupancySensorApi) GetOccupancy(_ context.Context, _ *traits.GetOccupancyRequest) (*traits.Occupancy, error) {
	return o.occupancy.Get().(*traits.Occupancy), nil
}

func (o *OccupancySensorApi) PullOccupancy(request *traits.PullOccupancyRequest, server traits.OccupancySensorApi_PullOccupancyServer) error {
	id := rand.Int()
	t0 := goTime.Now()
	sentItems := 0
	defer func() {
		log.Printf("[%x] PullOccupancy done in %v: sent %v", id, goTime.Now().Sub(t0), sentItems)
	}()
	log.Printf("[%x] PullOccupancy start %v", id, request)

	changes, done := o.occupancy.OnUpdate(server.Context())
	defer done()

	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case event, ok := <-changes:
			if !ok {
				return nil
			}
			change := &traits.PullOccupancyResponse_Change{
				Name:       request.Name,
				Occupancy:  event.Value.(*traits.Occupancy),
				ChangeTime: event.ChangeTime,
			}
			if err := server.Send(&traits.PullOccupancyResponse{Changes: []*traits.PullOccupancyResponse_Change{
				change,
			}}); err != nil {
				return err
			}
			sentItems++
		}
	}
}
