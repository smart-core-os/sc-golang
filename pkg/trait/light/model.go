package light

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	brightness *resource.Value
}

func NewModel(opts ...resource.Option) *Model {
	args := calcModelArgs(opts...)
	return &Model{
		brightness: resource.NewValue(args.brightnessOpts...),
	}
}

func (m *Model) GetBrightness(opts ...resource.ReadOption) (*traits.Brightness, error) {
	return m.brightness.Get(opts...).(*traits.Brightness), nil
}

func (m *Model) UpdateBrightness(light *traits.Brightness, opts ...resource.WriteOption) (*traits.Brightness, error) {
	res, err := m.brightness.Set(light, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*traits.Brightness), nil
}

func (m *Model) PullBrightness(ctx context.Context, opts ...resource.ReadOption) <-chan PullBrightnessChange {
	send := make(chan PullBrightnessChange)

	recv := m.brightness.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			value := change.Value.(*traits.Brightness)
			send <- PullBrightnessChange{
				Value:      value,
				ChangeTime: change.ChangeTime,
			}
		}
	}()

	return send
}

type PullBrightnessChange struct {
	Value      *traits.Brightness
	ChangeTime time.Time
}
