package memory

import (
	"context"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	"git.vanti.co.uk/smartcore/sc-api/go/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type Thermostat struct {
	traits.UnimplementedThermostatServer
	state *Resource
}

func NewThermostat() *Thermostat {
	return &Thermostat{
		state: NewResource(
			WithInitialValue(InitialThermostatState()),
			WithWritableFields(&fieldmaskpb.FieldMask{
				Paths: []string{
					"mode",
					"temperature_goal",
				},
			}),
		),
	}
}

func InitialThermostatState() *traits.ThermostatState {
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
	return t.state.Get().(*traits.ThermostatState), nil
}

func (t *Thermostat) UpdateState(ctx context.Context, request *traits.UpdateThermostatStateRequest) (*traits.ThermostatState, error) {
	update, err := t.state.Update(request.State, request.UpdateMask)
	return update.(*traits.ThermostatState), err
}

func (t *Thermostat) PullState(request *traits.PullThermostatStateRequest, server traits.Thermostat_PullStateServer) error {
	changes, done := t.state.OnUpdate(server.Context())
	defer done()

	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case event := <-changes:
			change := &traits.ThermostatStateChange{
				Name:       request.Name,
				State:      event.Value.(*traits.ThermostatState),
				CreateTime: event.ChangeTime,
			}
			err := server.Send(&traits.PullThermostatStateResponse{
				Changes: []*traits.ThermostatStateChange{change},
			})
			if err != nil {
				return err
			}
		}
	}
}
