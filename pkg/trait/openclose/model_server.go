package openclose

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ModelServer adapts a Model as a traits.OpenCloseApiServer.
type ModelServer struct {
	traits.UnimplementedOpenCloseApiServer
	model *Model
}

func (s *ModelServer) GetPositions(ctx context.Context, request *traits.GetOpenClosePositionsRequest) (*traits.OpenClosePositions, error) {
	return s.model.Positions(), nil
}

func (s *ModelServer) UpdatePositions(ctx context.Context, request *traits.UpdateOpenClosePositionsRequest) (*traits.OpenClosePositions, error) {
	if request.GetDelta() {
		return nil, status.Error(codes.Unimplemented, "delta update not supported")
	}

	return s.model.UpdatePositions(request.GetStates())
}

func (s *ModelServer) Stop(ctx context.Context, request *traits.StopOpenCloseRequest) (*traits.OpenClosePositions, error) {
	// the model does not support tweening, so updates complete instantly and it's always 'stopped'
	// therefore do nothing

	return s.model.Positions(), nil
}

func (s *ModelServer) PullPositions(request *traits.PullOpenClosePositionsRequest, server traits.OpenCloseApi_PullPositionsServer) error {
	changes := s.model.PullPositions(server.Context())
	for change := range changes {
		err := server.Send(&traits.PullOpenClosePositionsResponse{
			Changes: []*traits.PullOpenClosePositionsResponse_Change{
				{
					Name:       request.GetName(),
					ChangeTime: timestamppb.New(change.ChangeTime),
					State:      change.Value,
				},
			},
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Unwrap() interface{} {
	return s.model
}

func (s *ModelServer) Register(server *grpc.Server) {
	traits.RegisterOpenCloseApiServer(server, s)
}
