package airtemperature

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/memory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type MemoryDevice struct {
	traits.UnimplementedAirTemperatureApiServer
	airTemperature *memory.Resource
}

func NewMemoryDevice() *MemoryDevice {
	return &MemoryDevice{
		airTemperature: memory.NewResource(
			memory.WithInitialValue(InitialAirTemperatureState()),
			memory.WithWritablePaths(&traits.AirTemperature{},
				"mode",
				// temperature_goal oneof options
				"temperature_set_point",
				"temperature_set_point_delta",
				"temperature_range",
			),
		),
	}
}

func InitialAirTemperatureState() *traits.AirTemperature {
	return &traits.AirTemperature{
		AmbientTemperature: &types.Temperature{ValueCelsius: 22},
		TemperatureGoal: &traits.AirTemperature_TemperatureSetPoint{
			TemperatureSetPoint: &types.Temperature{ValueCelsius: 22},
		},
	}
}

func (t *MemoryDevice) Register(server *grpc.Server) {
	traits.RegisterAirTemperatureApiServer(server, t)
}

func (t *MemoryDevice) GetAirTemperature(_ context.Context, _ *traits.GetAirTemperatureRequest) (*traits.AirTemperature, error) {
	return t.airTemperature.Get().(*traits.AirTemperature), nil
}

func (t *MemoryDevice) UpdateAirTemperature(_ context.Context, request *traits.UpdateAirTemperatureRequest) (*traits.AirTemperature, error) {
	update, err := t.airTemperature.Set(request.State, memory.WithUpdateMask(request.UpdateMask))
	return update.(*traits.AirTemperature), err
}

func (t *MemoryDevice) PullAirTemperature(request *traits.PullAirTemperatureRequest, server traits.AirTemperatureApi_PullAirTemperatureServer) error {
	changes, done := t.airTemperature.OnUpdate(server.Context())
	defer done()

	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case event := <-changes:
			change := &traits.PullAirTemperatureResponse_Change{
				Name:       request.Name,
				State:      event.Value.(*traits.AirTemperature),
				ChangeTime: event.ChangeTime,
			}
			err := server.Send(&traits.PullAirTemperatureResponse{
				Changes: []*traits.PullAirTemperatureResponse_Change{change},
			})
			if err != nil {
				return err
			}
		}
	}
}
