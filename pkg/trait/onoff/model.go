package onoff

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"github.com/smart-core-os/sc-golang/pkg/memory"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type Model struct {
	onOff *memory.Resource // of *traits.OnOff
}

func NewModel(initialState traits.OnOff_State) *Model {
	return &Model{
		onOff: memory.NewResource(memory.WithInitialValue(&traits.OnOff{
			State: initialState,
		})),
	}
}

func (m *Model) GetOnOff(opts ...memory.GetOption) (*traits.OnOff, error) {
	return m.onOff.Get(opts...).(*traits.OnOff), nil
}

func (m *Model) UpdateOnOff(value *traits.OnOff, opts ...memory.UpdateOption) (*traits.OnOff, error) {
	res, err := m.onOff.Set(value, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*traits.OnOff), nil
}

func (m *Model) PullOnOff(ctx context.Context, mask *fieldmaskpb.FieldMask) (changes <-chan PullOnOffChange, done func()) {
	send := make(chan PullOnOffChange)

	recv, done := m.onOff.OnUpdate(ctx)
	go func() {
		filter := masks.NewResponseFilter(masks.WithFieldMask(mask))

		for change := range recv {
			value := filter.FilterClone(change.Value).(*traits.OnOff)
			send <- PullOnOffChange{
				Value:      value,
				ChangeTime: change.ChangeTime.AsTime(),
			}
		}
	}()

	// when done is called, then the resource will close recv for us
	return send, done
}

type PullOnOffChange struct {
	Value      *traits.OnOff
	ChangeTime time.Time
}
