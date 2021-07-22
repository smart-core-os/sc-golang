package memory

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CountApi struct {
	traits.UnimplementedCountApiServer

	count *Resource
}

// compile time check that we implement the interface we need
var _ traits.CountApiServer = &CountApi{}

func NewCountApi() *CountApi {
	return &CountApi{
		count: NewResource(
			WithInitialValue(InitialCount()),
			WithWritablePaths("added", "removed"),
		),
	}
}

func InitialCount() *traits.Count {
	return &traits.Count{
		ResetTime: timestamppb.Now(),
	}
}

func (t *CountApi) GetCount(_ context.Context, _ *traits.GetCountRequest) (*traits.Count, error) {
	return t.count.Get().(*traits.Count), nil
}

func (t *CountApi) ResetCount(_ context.Context, request *traits.ResetCountRequest) (*traits.Count, error) {
	rt := request.ResetTime
	if rt == nil {
		rt = timestamppb.Now()
	}
	res, err := t.count.Set(&traits.Count{Added: 0, Removed: 0, ResetTime: rt}, WithAllFieldsWritable())
	return res.(*traits.Count), err
}

func (t *CountApi) UpdateCount(_ context.Context, request *traits.UpdateCountRequest) (*traits.Count, error) {
	res, err := t.count.Set(request.Count, WithUpdateMask(request.UpdateMask), InterceptBefore(func(old, value proto.Message) {
		if request.Delta {
			tOld := old.(*traits.Count)
			tValue := value.(*traits.Count)
			tValue.Added += tOld.Added
			tValue.Removed += tOld.Removed
		}
	}))
	return res.(*traits.Count), err
}

func (t *CountApi) PullCounts(request *traits.PullCountsRequest, server traits.CountApi_PullCountsServer) error {
	changes, done := t.count.OnUpdate(server.Context())
	defer done()

	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case event := <-changes:
			change := &traits.PullCountsResponse_Change{
				Name:  request.Name,
				Count: event.Value.(*traits.Count),
			}
			err := server.Send(&traits.PullCountsResponse{
				Changes: []*traits.PullCountsResponse_Change{change},
			})
			if err != nil {
				return err
			}
		}
	}
}
