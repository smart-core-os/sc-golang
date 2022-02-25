package metadata

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type ModelServer struct {
	traits.UnimplementedMetadataApiServer

	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	s := &ModelServer{model: model}
	return s
}

func (s *ModelServer) Unwrap() interface{} {
	return s.model
}

func (s *ModelServer) Register(server *grpc.Server) {
	traits.RegisterMetadataApiServer(server, s)
}

func (s *ModelServer) GetMetadata(_ context.Context, request *traits.GetMetadataRequest) (*traits.Metadata, error) {
	return s.model.GetMetadata(resource.WithGetMask(request.ReadMask))
}

func (s *ModelServer) PullMetadata(request *traits.PullMetadataRequest, server traits.MetadataApi_PullMetadataServer) error {
	filter := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))
	var lastSent *traits.Metadata
	for change := range s.model.PullMetadata(server.Context()) {
		m := filter.FilterClone(change.Metadata).(*traits.Metadata)
		if proto.Equal(lastSent, m) {
			continue
		}
		lastSent = m

		err := server.Send(&traits.PullMetadataResponse{Changes: []*traits.PullMetadataResponse_Change{
			{Name: request.Name, ChangeTime: change.ChangeTime, Metadata: m},
		}})
		if err != nil {
			return err
		}
	}
	return server.Context().Err() // the loop only ends when the context is done
}
