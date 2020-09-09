package memory

import (
	"context"
	"sync"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	"github.com/olebedev/emitter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OccupancyApi struct {
	traits.UnimplementedOccupancyApiServer
	mu             sync.Mutex // guards the lastKnownState
	lastKnownState *traits.Occupancy

	bus *emitter.Emitter
}

func NewOccupancyApi(initialState *traits.Occupancy) *OccupancyApi {
	return &OccupancyApi{
		lastKnownState: initialState,
		bus:            &emitter.Emitter{},
	}
}

func (o *OccupancyApi) Register(server *grpc.Server) {
	traits.RegisterOccupancyApiServer(server, o)
}

// SetOccupancy updates the known occupancy state for this device
func (o *OccupancyApi) SetOccupancy(ctx context.Context, occupancy *traits.Occupancy) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.lastKnownState = occupancy
	o.bus.Emit("change:occupancy", occupancy)
}

func (o *OccupancyApi) GetOccupancy(ctx context.Context, request *traits.GetOccupancyRequest) (*traits.Occupancy, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.lastKnownState, nil
}

func (o *OccupancyApi) PullOccupancy(request *traits.PullOccupancyRequest, server traits.OccupancyApi_PullOccupancyServer) error {
	changes := o.bus.On("change:occupancy")
	defer o.bus.Off("change:occupancy", changes)

	name := request.Name
	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case change, ok := <-changes:
			if !ok {
				return nil
			}
			occupancy := change.Args[0].(*traits.Occupancy)
			if err := server.Send(&traits.PullOccupancyResponse{Changes: []*traits.OccupancyChange{
				{Name: name, Occupancy: occupancy, CreateTime: timestamppb.Now()},
			}}); err != nil {
				return err
			}
		}
	}
}
