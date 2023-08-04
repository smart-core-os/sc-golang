package openclose

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type ModelServer struct {
	traits.UnimplementedOpenCloseApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Register(server *grpc.Server) {
	traits.RegisterOpenCloseApiServer(server, s)
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) GetPositions(_ context.Context, request *traits.GetOpenClosePositionsRequest) (*traits.OpenClosePositions, error) {
	return s.model.GetPositions(resource.WithReadMask(request.GetReadMask()))
}

func (s *ModelServer) UpdatePositions(_ context.Context, request *traits.UpdateOpenClosePositionsRequest) (*traits.OpenClosePositions, error) {
	return s.model.UpdatePositions(request.GetStates(), resource.WithUpdateMask(request.GetUpdateMask()))
}

func (s *ModelServer) PullPositions(request *traits.PullOpenClosePositionsRequest, server traits.OpenCloseApi_PullPositionsServer) error {
	for change := range s.model.PullPositions(server.Context(), resource.WithReadMask(request.GetReadMask()), resource.WithUpdatesOnly(request.GetUpdatesOnly())) {
		msg := &traits.PullOpenClosePositionsResponse{Changes: []*traits.PullOpenClosePositionsResponse_Change{{
			Name:              request.Name,
			ChangeTime:        timestamppb.New(change.ChangeTime),
			OpenClosePosition: change.Positions,
		}}}
		if err := server.Send(msg); err != nil {
			return err
		}
	}
	return nil
}
