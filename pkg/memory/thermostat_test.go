package memory

import (
	"context"
	"testing"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	"git.vanti.co.uk/smartcore/sc-api/go/types"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestThermostat_GetState_Initial(t *testing.T) {
	api := NewThermostat()
	state, err := api.GetState(context.Background(), &traits.GetThermostatStateRequest{Name: "test"})
	if err != nil {
		t.Errorf("error not expected %v", err)
	}
	if diff := cmp.Diff(InitialState(), state, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected initial value (-want,+got)\n%v", diff)
	}
}

func TestThermostat_UpdateState(t *testing.T) {
	api := NewThermostat()
	initialState, _ := api.GetState(context.Background(), &traits.GetThermostatStateRequest{Name: "test"})
	newState := &traits.ThermostatState{
		// fields we can edit
		Mode: traits.ThermostatMode_ECO,
		TemperatureGoal: &traits.ThermostatState_TemperatureSetPoint{
			TemperatureSetPoint: &types.Temperature{ValueCelsius: 30},
		},
		// fields we can't edit
		AmbientTemperature: &types.Temperature{ValueCelsius: -12},
		AmbientHumidity:    wrapperspb.Float(12.2),
	}
	updatedState, err := api.UpdateState(context.Background(), &traits.UpdateThermostatStateRequest{
		Name:  "test",
		State: newState,
	})
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	// check the response is what we expect
	// writable fields
	if diff := cmp.Diff(newState.Mode, updatedState.Mode, protocmp.Transform()); diff != "" {
		t.Errorf("UpdateState() Mode mismatch (-want,+got)\n%v", diff)
	}
	if diff := cmp.Diff(newState.TemperatureGoal, updatedState.TemperatureGoal, protocmp.Transform()); diff != "" {
		t.Errorf("UpdateState() TemperatureGoal mismatch (-want,+got)\n%v", diff)
	}
	// read-only fields
	if diff := cmp.Diff(initialState.AmbientHumidity, updatedState.AmbientHumidity, protocmp.Transform()); diff != "" {
		t.Errorf("UpdateState() AmbientHumidity mismatch (-want,+got)\n%v", diff)
	}
	if diff := cmp.Diff(initialState.AmbientTemperature, updatedState.AmbientTemperature, protocmp.Transform()); diff != "" {
		t.Errorf("UpdateState() AmbientTemperature mismatch (-want,+got)\n%v", diff)
	}
}

func TestThermostat_UpdateState_Mask(t *testing.T) {
	api := NewThermostat()
	initialState, _ := api.GetState(context.Background(), &traits.GetThermostatStateRequest{Name: "test"})
	newState := &traits.ThermostatState{
		// fields we can edit
		Mode: traits.ThermostatMode_ECO,
		TemperatureGoal: &traits.ThermostatState_TemperatureSetPoint{
			TemperatureSetPoint: &types.Temperature{ValueCelsius: 30},
		},
		// fields we can't edit
		AmbientTemperature: &types.Temperature{ValueCelsius: -12},
		AmbientHumidity:    wrapperspb.Float(12.2),
	}
	updatedState, err := api.UpdateState(context.Background(), &traits.UpdateThermostatStateRequest{
		Name:       "test",
		State:      newState,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"mode"}},
	})
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	// check the response is what we expect
	// writable fields
	if diff := cmp.Diff(newState.Mode, updatedState.Mode, protocmp.Transform()); diff != "" {
		t.Errorf("UpdateState() Mode mismatch (-want,+got)\n%v", diff)
	}
	// unedited
	if diff := cmp.Diff(initialState.TemperatureGoal, updatedState.TemperatureGoal, protocmp.Transform()); diff != "" {
		t.Errorf("UpdateState() TemperatureGoal mismatch (-want,+got)\n%v", diff)
	}
	// read-only fields
	if diff := cmp.Diff(initialState.AmbientHumidity, updatedState.AmbientHumidity, protocmp.Transform()); diff != "" {
		t.Errorf("UpdateState() AmbientHumidity mismatch (-want,+got)\n%v", diff)
	}
	if diff := cmp.Diff(initialState.AmbientTemperature, updatedState.AmbientTemperature, protocmp.Transform()); diff != "" {
		t.Errorf("UpdateState() AmbientTemperature mismatch (-want,+got)\n%v", diff)
	}
}
