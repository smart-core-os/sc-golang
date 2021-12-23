package energystorage

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/memory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ModelServer struct {
	traits.UnimplementedEnergyStorageApiServer

	model *Model

	readOnly bool
}

func NewModelServer(model *Model, opts ...ServerOption) *ModelServer {
	s := &ModelServer{model: model}
	for _, opt := range opts {
		opt(s)
	}
	return s
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
	if s.readOnly {
		return nil, status.Errorf(codes.Unimplemented, "EnergyStorage.Charge")
	}

	level := traits.EnergyLevel{}
	if request.GetCharge() {
		level.Flow = &traits.EnergyLevel_Charge{}
	} else {
		level.Flow = &traits.EnergyLevel_Discharge{}
	}
	_, err := s.model.UpdateEnergyLevel(&level, memory.WithUpdatePaths("idle", "charge", "discharge"))
	if err != nil {
		return nil, err
	}
	return &traits.ChargeResponse{}, nil
}

type ServerOption func(s *ModelServer)

func ReadOnly() ServerOption {
	return func(s *ModelServer) {
		s.readOnly = true
	}
}