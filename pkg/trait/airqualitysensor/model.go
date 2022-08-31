package airqualitysensor

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	airQuality *resource.Value
}

func NewModel(initialState *traits.AirQuality) *Model {
	return &Model{
		airQuality: resource.NewValue(resource.WithInitialValue(initialState)),
	}
}

func (m *Model) UpdateAirQuality(airTemperature *traits.AirQuality, opts ...resource.WriteOption) (*traits.AirQuality, error) {
	res, err := m.airQuality.Set(airTemperature, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*traits.AirQuality), nil
}

func (m *Model) GetAirQuality(opts ...resource.ReadOption) (*traits.AirQuality, error) {
	return m.airQuality.Get(opts...).(*traits.AirQuality), nil
}

func (m *Model) PullAirQuality(ctx context.Context, opts ...resource.ReadOption) <-chan PullAirQualityChange {
	send := make(chan PullAirQualityChange)

	recv := m.airQuality.Pull(ctx, opts...)
	go func() {
		for change := range recv {
			value := change.Value.(*traits.AirQuality)
			send <- PullAirQualityChange{
				Value:      value,
				ChangeTime: change.ChangeTime,
			}
		}
	}()

	return send
}

type PullAirQualityChange struct {
	Value      *traits.AirQuality
	ChangeTime time.Time
}
