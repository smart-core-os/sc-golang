package energystorage

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/memory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ModelServer struct {
	traits.UnimplementedEnergyStorageApiServer

	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Register(server *grpc.Server) {
	traits.RegisterEnergyStorageApiServer(server, s)
}

func (s *ModelServer) GetEnergyLevel(_ context.Context, request *traits.GetEnergyLevelRequest) (*traits.EnergyLevel, error) {
	return s.model.GetEnergyLevel(memory.WithGetMask(request.GetFields()))
}

func (s *ModelServer) PullEnergyLevel(request *traits.PullEnergyLevelRequest, server traits.EnergyStorageApi_PullEnergyLevelServer) error {
	updates, done := s.model.PullEnergyLevel(server.Context(), request.GetFields())
	defer done()

	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case update := <-updates:
			change := &traits.PullEnergyLevelResponse_Change{
				Name:        request.Name,
				ChangeTime:  timestamppb.New(update.ChangeTime),
				EnergyLevel: update.Value,
			}

			err := server.Send(&traits.PullEnergyLevelResponse{
				Changes: []*traits.PullEnergyLevelResponse_Change{change},
			})
			if err != nil {
				return err
			}
		}
	}
}

func (s *ModelServer) Charge(_ context.Context, request *traits.ChargeRequest) (*traits.ChargeResponse, error) {
	_, err := s.model.UpdateEnergyLevel(&traits.EnergyLevel{Charging: request.GetCharge()}, memory.WithUpdatePaths("charging"))
	if err != nil {
		return nil, err
	}
	return &traits.ChargeResponse{}, nil
}
