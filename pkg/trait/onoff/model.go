package onoff

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type Model struct {
	onOff *resource.Value // of *traits.OnOff
}

func NewModel(initialState traits.OnOff_State) *Model {
	return &Model{
		onOff: resource.NewValue(resource.WithInitialValue(&traits.OnOff{
			State: initialState,
		})),
	}
}

func (m *Model) GetOnOff(opts ...resource.GetOption) (*traits.OnOff, error) {
	return m.onOff.Get(opts...).(*traits.OnOff), nil
}

func (m *Model) UpdateOnOff(value *traits.OnOff, opts ...resource.UpdateOption) (*traits.OnOff, error) {
	res, err := m.onOff.Set(value, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*traits.OnOff), nil
}

func (m *Model) PullOnOff(ctx context.Context, mask *fieldmaskpb.FieldMask) (changes <-chan PullOnOffChange, done func()) {
	send := make(chan PullOnOffChange)

	ctx, done = context.WithCancel(ctx)
	recv := m.onOff.Pull(ctx)
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
