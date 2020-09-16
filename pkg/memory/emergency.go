package memory

import (
	"context"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type EmergencyApi struct {
	traits.UnimplementedEmergencyApiServer
	state *Resource
}

func NewEmergencyApi() *EmergencyApi {
	return &EmergencyApi{
		state: NewResource(WithInitialValue(InitialEmergency())),
	}
}

func InitialEmergency() *traits.Emergency {
	return &traits.Emergency{
		Level:           traits.EmergencyLevel_EMERGENCY_LEVEL_OK,
		LevelChangeTime: serverTimestamp(),
	}
}

func (t *EmergencyApi) Register(server *grpc.Server) {
	traits.RegisterEmergencyApiServer(server, t)
}

func (t *EmergencyApi) GetEmergency(ctx context.Context, request *traits.GetEmergencyRequest) (*traits.Emergency, error) {
	return t.state.Get().(*traits.Emergency), nil
}

func (t *EmergencyApi) UpdateEmergency(ctx context.Context, request *traits.UpdateEmergencyRequest) (*traits.Emergency, error) {
	update, err := t.state.UpdateModified(request.Emergency, request.UpdateMask, func(old, new proto.Message) {
		// user server time if the level changed but the change time didn't
		oldt, newt := old.(*traits.Emergency), new.(*traits.Emergency)
		if newt.Level != oldt.Level && oldt.LevelChangeTime == newt.LevelChangeTime {
			newt.LevelChangeTime = serverTimestamp()
		}
	})
	return update.(*traits.Emergency), err
}

func (t *EmergencyApi) PullEmergency(request *traits.PullEmergencyRequest, server traits.EmergencyApi_PullEmergencyServer) error {
	changes, done := t.state.OnUpdate(server.Context())
	defer done()

	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case event := <-changes:
			change := &traits.EmergencyChange{
				Name:       request.Name,
				Emergency:  event.Value.(*traits.Emergency),
				CreateTime: event.ChangeTime,
			}
			err := server.Send(&traits.PullEmergencyResponse{
				Changes: []*traits.EmergencyChange{change},
			})
			if err != nil {
				return err
			}
		}
	}
}
