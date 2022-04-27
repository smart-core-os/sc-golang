package enterleavesensor

import (
	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ModelServer struct {
	traits.UnimplementedEnterLeaveSensorApiServer
	model *Model
}

// NewModelServer converts a Model into a type implementing traits.EnterLeaveSensorApiServer.
func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Unwrap() interface{} {
	return m.model
}

func (m *ModelServer) Register(server *grpc.Server) {
	traits.RegisterEnterLeaveSensorApiServer(server, m)
}

func (m *ModelServer) PullEnterLeaveEvents(request *traits.PullEnterLeaveEventsRequest, server traits.EnterLeaveSensorApi_PullEnterLeaveEventsServer) error {
	for change := range m.model.PullEnterLeaveEvents(server.Context(), resource.WithReadMask(request.ReadMask)) {
		err := server.Send(&traits.PullEnterLeaveEventsResponse{Changes: []*traits.PullEnterLeaveEventsResponse_Change{
			{Name: request.Name, ChangeTime: timestamppb.New(change.ChangeTime), EnterLeaveEvent: change.Value},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}
