package memory

import (
	"context"
	"log"
	"math/rand"
	goTime "time"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type OccupancyApi struct {
	traits.UnimplementedOccupancyApiServer
	occupancy *Resource
}

func NewOccupancyApi(initialState *traits.Occupancy) *OccupancyApi {
	return &OccupancyApi{
		occupancy: NewResource(WithInitialValue(initialState)),
	}
}

func (o *OccupancyApi) Register(server *grpc.Server) {
	traits.RegisterOccupancyApiServer(server, o)
}

// SetOccupancy updates the known occupancy state for this device
func (o *OccupancyApi) SetOccupancy(ctx context.Context, occupancy *traits.Occupancy) {
	_, _ = o.occupancy.Update(occupancy, nil)
}

func (o *OccupancyApi) GetOccupancy(ctx context.Context, request *traits.GetOccupancyRequest) (*traits.Occupancy, error) {
	return o.occupancy.Get().(*traits.Occupancy), nil
}

func (o *OccupancyApi) PullOccupancy(request *traits.PullOccupancyRequest, server traits.OccupancyApi_PullOccupancyServer) error {
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
			change := &traits.OccupancyChange{
				Name:       request.Name,
				Occupancy:  event.Value.(*traits.Occupancy),
				CreateTime: event.ChangeTime,
			}
			if err := server.Send(&traits.PullOccupancyResponse{Changes: []*traits.OccupancyChange{
				change,
			}}); err != nil {
				return err
			}
			sentItems++
		}
	}
}
