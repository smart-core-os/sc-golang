package occupancy

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type ModelServer struct {
	traits.UnimplementedOccupancySensorApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{
		model: model,
	}
}

func (s *ModelServer) Register(server *grpc.Server) {
	traits.RegisterOccupancySensorApiServer(server, s)
}

func (s *ModelServer) GetOccupancy(_ context.Context, _ *traits.GetOccupancyRequest) (*traits.Occupancy, error) {
	return s.model.GetOccupancy()
}

func (s *ModelServer) PullOccupancy(request *traits.PullOccupancyRequest, server traits.OccupancySensorApi_PullOccupancyServer) error {
	updates, done := s.model.PullOccupancy(server.Context(), nil)
	defer done()

	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case update := <-updates:
			change := &traits.PullOccupancyResponse_Change{
				Name:       request.Name,
				ChangeTime: timestamppb.New(update.ChangeTime),
				Occupancy:  update.Value,
			}

			err := server.Send(&traits.PullOccupancyResponse{
				Changes: []*traits.PullOccupancyResponse_Change{change},
			})
			if err != nil {
				return err
			}
		}
	}
}
