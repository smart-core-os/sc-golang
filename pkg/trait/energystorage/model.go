package energystorage

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/cmp"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type Model struct {
	energyLevel *resource.Value // of traits.EnergyLevel
}

func NewModel() *Model {
	eq := cmp.Equal(
		cmp.FloatValueApprox(0, 0.1),
		cmp.TimeValueWithin(1*time.Second),
		cmp.DurationValueWithin(1*time.Second),
	)
	return &Model{
		energyLevel: resource.NewValue(resource.WithInitialValue(&traits.EnergyLevel{}), resource.WithMessageEquivalence(eq)),
	}
}

func (m *Model) GetEnergyLevel(opts ...resource.ReadOption) (*traits.EnergyLevel, error) {
	res := m.energyLevel.Get(opts...)
	return res.(*traits.EnergyLevel), nil
}

func (m *Model) UpdateEnergyLevel(energyLevel *traits.EnergyLevel, opts ...resource.UpdateOption) (*traits.EnergyLevel, error) {
	res, err := m.energyLevel.Set(energyLevel, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*traits.EnergyLevel), nil
}

type PullEnergyLevelChange struct {
	Value      *traits.EnergyLevel
	ChangeTime time.Time
}

func (m *Model) PullEnergyLevel(ctx context.Context, mask *fieldmaskpb.FieldMask) <-chan PullEnergyLevelChange {
	send := make(chan PullEnergyLevelChange)

	recv := m.energyLevel.Pull(ctx, resource.WithReadMask(mask))
	go func() {
		for change := range recv {
			demand := change.Value.(*traits.EnergyLevel)
			send <- PullEnergyLevelChange{
				Value:      demand,
				ChangeTime: change.ChangeTime.AsTime(),
			}
		}
	}()

	// when done is called, then the resource will close recv for us
	return send
}
