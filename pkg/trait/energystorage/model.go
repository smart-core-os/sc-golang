package energystorage

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"github.com/smart-core-os/sc-golang/pkg/memory"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type Model struct {
	energyLevel *memory.Resource // of traits.EnergyLevel
}

func NewModel() *Model {
	return &Model{
		energyLevel: memory.NewResource(memory.WithInitialValue(&traits.EnergyLevel{})),
	}
}

func (m *Model) GetEnergyLevel(opts ...memory.GetOption) (*traits.EnergyLevel, error) {
	res := m.energyLevel.Get(opts...)
	return res.(*traits.EnergyLevel), nil
}

func (m *Model) UpdateEnergyLevel(energyLevel *traits.EnergyLevel, opts ...memory.UpdateOption) (*traits.EnergyLevel, error) {
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

func (m *Model) PullEnergyLevel(ctx context.Context, mask *fieldmaskpb.FieldMask) (changes <-chan PullEnergyLevelChange, done func()) {
	send := make(chan PullEnergyLevelChange)

	recv, done := m.energyLevel.OnUpdate(ctx)
	go func() {
		filter := masks.NewResponseFilter(masks.WithFieldMask(mask))

		for change := range recv {
			demand := filter.FilterClone(change.Value).(*traits.EnergyLevel)
			send <- PullEnergyLevelChange{
				Value:      demand,
				ChangeTime: change.ChangeTime.AsTime(),
			}
		}
	}()

	// when done is called, then the resource will close recv for us
	return send, done
}
