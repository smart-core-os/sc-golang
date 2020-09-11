package memory

import (
	"context"
	"sync"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	fieldMaskUtils "github.com/mennanov/fieldmask-utils"
	"github.com/olebedev/emitter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"git.vanti.co.uk/smartcore/sc-golang/pkg/masks"
)

// EmergencyApiWritableFields are all fields that can be updated
var EmergencyApiWritableFields *fieldmaskpb.FieldMask

type EmergencyApi struct {
	traits.UnimplementedEmergencyApiServer
	state   *traits.Emergency
	stateMu sync.RWMutex
	bus     *emitter.Emitter
}

func NewEmergencyApi() *EmergencyApi {
	return &EmergencyApi{
		state: InitialEmergency(),

		bus: &emitter.Emitter{},
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
	t.stateMu.RLock()
	defer t.stateMu.RUnlock()
	return t.state, nil
}

func (t *EmergencyApi) UpdateEmergency(ctx context.Context, request *traits.UpdateEmergencyRequest) (*traits.Emergency, error) {
	// make sure they can only write the fields we want
	mask, err := masks.ValidWritableMask(EmergencyApiWritableFields, request.UpdateMask, request.Emergency)
	if err != nil {
		return nil, err
	}

	_, newValue, err := applyChange(
		&t.stateMu,
		func() (proto.Message, bool) {
			return t.state, true
		},
		func(message proto.Message) error {
			value := message.(*traits.Emergency)
			// track these so we can update the LastChangeTime if needed
			oldLevel := value.Level
			oldLevelChangeTime := value.LevelChangeTime

			if mask != nil {
				// apply only selected fields
				if err := fieldMaskUtils.StructToStruct(mask, request.Emergency, value); err != nil {
					return err
				}
			} else {
				// replace the booking data
				proto.Reset(value)
				proto.Merge(value, request.Emergency)
			}

			// user server time if the level changed but the change time didn't
			if value.Level != oldLevel && oldLevelChangeTime == value.LevelChangeTime {
				value.LevelChangeTime = serverTimestamp()
			}

			return nil
		},
		func(message proto.Message) {
			t.state = message.(*traits.Emergency)
		},
	)

	if err != nil {
		return nil, err
	}

	t.bus.Emit("change", &traits.EmergencyChange{
		Name:       request.Name,
		Emergency:  newValue.(*traits.Emergency),
		CreateTime: serverTimestamp(),
	})

	return newValue.(*traits.Emergency), err
}

func (t *EmergencyApi) PullEmergency(request *traits.PullEmergencyRequest, server traits.EmergencyApi_PullEmergencyServer) error {
	changes := t.bus.On("change")
	defer t.bus.Off("change", changes)

	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case event := <-changes:
			change := event.Args[0].(*traits.EmergencyChange)
			err := server.Send(&traits.PullEmergencyResponse{
				Changes: []*traits.EmergencyChange{change},
			})
			if err != nil {
				return err
			}
		}
	}
}
