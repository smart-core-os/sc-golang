package occupancysensor

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-golang/pkg/masks"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-api/go/traits"
)

type Model struct {
	occupancy *resource.Value
}

func NewModel(initialState *traits.Occupancy) *Model {
	return &Model{
		occupancy: resource.NewValue(resource.WithInitialValue(initialState)),
	}
}

// SetOccupancy updates the known occupancy state for this device
func (m *Model) SetOccupancy(occupancy *traits.Occupancy, opts ...resource.UpdateOption) (*traits.Occupancy, error) {
	res, err := m.occupancy.Set(occupancy, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*traits.Occupancy), nil
}

func (m *Model) GetOccupancy(opts ...resource.GetOption) (*traits.Occupancy, error) {
	return m.occupancy.Get(opts...).(*traits.Occupancy), nil
}

func (m *Model) PullOccupancy(ctx context.Context, mask *fieldmaskpb.FieldMask) (changes <-chan PullOccupancyChange, done func()) {
	send := make(chan PullOccupancyChange)

	recv, done := m.occupancy.Pull(ctx)
	go func() {
		filter := masks.NewResponseFilter(masks.WithFieldMask(mask))

		for change := range recv {
			value := filter.FilterClone(change.Value).(*traits.Occupancy)
			send <- PullOccupancyChange{
				Value:      value,
				ChangeTime: change.ChangeTime.AsTime(),
			}
		}
	}()

	// when done is called, then the resource will close recv for us
	return send, done
}

type PullOccupancyChange struct {
	Value      *traits.Occupancy
	ChangeTime time.Time
}
