package memory

import (
	"context"
	"sync"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	"git.vanti.co.uk/smartcore/sc-api/go/types"
	fieldMaskUtils "github.com/mennanov/fieldmask-utils"
	"github.com/olebedev/emitter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"git.vanti.co.uk/smartcore/sc-golang/pkg/masks"
)

// WritableFields are all fields that can be updated
var WritableFields = &fieldmaskpb.FieldMask{
	Paths: []string{
		"mode",
		"temperature_goal",
	},
}

type Thermostat struct {
	traits.UnimplementedThermostatServer
	state   *traits.ThermostatState
	stateMu sync.RWMutex
	bus     *emitter.Emitter
}

func NewThermostat() *Thermostat {
	return &Thermostat{
		state: InitialState(),

		bus: &emitter.Emitter{},
	}
}

func InitialState() *traits.ThermostatState {
	return &traits.ThermostatState{
		AmbientTemperature: &types.Temperature{ValueCelsius: 22},
		TemperatureGoal: &traits.ThermostatState_TemperatureSetPoint{
			TemperatureSetPoint: &types.Temperature{ValueCelsius: 22},
		},
	}
}

func (t *Thermostat) Register(server *grpc.Server) {
	traits.RegisterThermostatServer(server, t)
}

func (t *Thermostat) GetState(ctx context.Context, request *traits.GetThermostatStateRequest) (*traits.ThermostatState, error) {
	t.stateMu.RLock()
	defer t.stateMu.RUnlock()
	return t.state, nil
}

func (t *Thermostat) UpdateState(ctx context.Context, request *traits.UpdateThermostatStateRequest) (*traits.ThermostatState, error) {
	// make sure they can only write the fields we want
	mask, err := masks.ValidWritableMask(WritableFields, request.UpdateMask, request.State)
	if err != nil {
		return nil, err
	}

	_, newValue, err := applyChange(
		&t.stateMu,
		func() (proto.Message, bool) {
			return t.state, true
		},
		func(message proto.Message) error {
			if mask != nil {
				// apply only selected fields
				return fieldMaskUtils.StructToStruct(mask, request.State, message.(*traits.ThermostatState))
			} else {
				// replace the booking data
				proto.Reset(message)
				proto.Merge(message, request.State)
				return nil
			}
		},
		func(message proto.Message) {
			t.state = message.(*traits.ThermostatState)
		},
	)

	if err != nil {
		return nil, err
	}

	t.bus.Emit("change", &traits.ThermostatStateChange{
		Name:       request.Name,
		State:      newValue.(*traits.ThermostatState),
		CreateTime: timestamppb.Now(),
	})

	return newValue.(*traits.ThermostatState), err
}

func (t *Thermostat) PullState(request *traits.PullThermostatStateRequest, server traits.Thermostat_PullStateServer) error {
	changes := t.bus.On("change")
	defer t.bus.Off("change", changes)

	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case event := <-changes:
			change := event.Args[0].(*traits.ThermostatStateChange)
			err := server.Send(&traits.PullThermostatStateResponse{
				Changes: []*traits.ThermostatStateChange{change},
			})
			if err != nil {
				return err
			}
		}
	}
}
