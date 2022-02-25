package energystorage

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/cmp"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"google.golang.org/protobuf/proto"
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
		energyLevel: resource.NewValue(resource.WithInitialValue(&traits.EnergyLevel{}), resource.WithValueMessageEquivalence(eq)),
	}
}

func (m *Model) GetEnergyLevel(opts ...resource.GetOption) (*traits.EnergyLevel, error) {
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

func (m *Model) PullEnergyLevel(ctx context.Context, mask *fieldmaskpb.FieldMask) (changes <-chan PullEnergyLevelChange, done func()) {
	send := make(chan PullEnergyLevelChange)

	recv, done := m.energyLevel.Pull(ctx)
	go func() {
		filter := masks.NewResponseFilter(masks.WithFieldMask(mask))
		var lastSent *traits.EnergyLevel

		for change := range recv {
			demand := filter.FilterClone(change.Value).(*traits.EnergyLevel)
			if proto.Equal(lastSent, demand) {
				continue
			}
			lastSent = demand
			send <- PullEnergyLevelChange{
				Value:      demand,
				ChangeTime: change.ChangeTime.AsTime(),
			}
		}
	}()

	// when done is called, then the resource will close recv for us
	return send, done
}
