package emergency

import (
	"context"

	"github.com/smart-core-os/sc-golang/pkg/resource"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type MemoryDevice struct {
	traits.UnimplementedEmergencyApiServer
	state *resource.Value
}

func NewMemoryDevice() *MemoryDevice {
	return &MemoryDevice{
		state: resource.NewValue(resource.WithInitialValue(InitialEmergency())),
	}
}

func InitialEmergency() *traits.Emergency {
	return &traits.Emergency{
		Level:           traits.Emergency_OK,
		LevelChangeTime: serverTimestamp(),
	}
}

func (t *MemoryDevice) Register(server *grpc.Server) {
	traits.RegisterEmergencyApiServer(server, t)
}

func (t *MemoryDevice) GetEmergency(_ context.Context, _ *traits.GetEmergencyRequest) (*traits.Emergency, error) {
	return t.state.Get().(*traits.Emergency), nil
}

func (t *MemoryDevice) UpdateEmergency(_ context.Context, request *traits.UpdateEmergencyRequest) (*traits.Emergency, error) {
	update, err := t.state.Set(request.Emergency, resource.WithUpdateMask(request.UpdateMask), resource.InterceptAfter(func(old, new proto.Message) {
		// user server time if the level changed but the change time didn't
		oldt, newt := old.(*traits.Emergency), new.(*traits.Emergency)
		if newt.Level != oldt.Level && oldt.LevelChangeTime == newt.LevelChangeTime {
			newt.LevelChangeTime = serverTimestamp()
		}
	}))
	return update.(*traits.Emergency), err
}

func (t *MemoryDevice) PullEmergency(request *traits.PullEmergencyRequest, server traits.EmergencyApi_PullEmergencyServer) error {
	changes, done := t.state.Pull(server.Context())
	defer done()

	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case event := <-changes:
			change := &traits.PullEmergencyResponse_Change{
				Name:       request.Name,
				Emergency:  event.Value.(*traits.Emergency),
				ChangeTime: event.ChangeTime,
			}
			err := server.Send(&traits.PullEmergencyResponse{
				Changes: []*traits.PullEmergencyResponse_Change{change},
			})
			if err != nil {
				return err
			}
		}
	}
}
