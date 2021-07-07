package memory

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type AirTemperatureApi struct {
	traits.UnimplementedAirTemperatureApiServer
	airTemperature *Resource
}

func NewAirTemperatureApi() *AirTemperatureApi {
	return &AirTemperatureApi{
		airTemperature: NewResource(
			WithInitialValue(InitialAirTemperatureState()),
			WithWritableFields(&fieldmaskpb.FieldMask{
				Paths: []string{
					"mode",
					"temperature_goal",
				},
			}),
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

func (t *AirTemperatureApi) Register(server *grpc.Server) {
	traits.RegisterAirTemperatureApiServer(server, t)
}

func (t *AirTemperatureApi) GetAirTemperature(_ context.Context, _ *traits.GetAirTemperatureRequest) (*traits.AirTemperature, error) {
	return t.airTemperature.Get().(*traits.AirTemperature), nil
}

func (t *AirTemperatureApi) UpdateAirTemperature(_ context.Context, request *traits.UpdateAirTemperatureRequest) (*traits.AirTemperature, error) {
	update, err := t.airTemperature.Update(request.State, request.UpdateMask)
	return update.(*traits.AirTemperature), err
}

func (t *AirTemperatureApi) PullAirTemperature(request *traits.PullAirTemperatureRequest, server traits.AirTemperatureApi_PullAirTemperatureServer) error {
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
