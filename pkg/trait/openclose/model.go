package openclose

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-golang/pkg/cmp"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Preset struct {
	Name       string
	Percentage float32
}

// Model provides a data structure for supporting the Fan Speed trait.
// Unless configured otherwise, this model supports DefaultPresets presets.
type Model struct {
	positions *resource.Value // of *traits.OpenClosePositions
}

// NewModel constructs a Model with default values using the first of DefaultPresets.
func NewModel() *Model {
	position := &traits.OpenClosePosition{
		OpenPercent: 0,
		Direction:   traits.OpenClosePosition_UNSPECIFIED,
	}
	positions := &traits.OpenClosePositions{
		States: []*traits.OpenClosePosition{position},
	}

	model := &Model{
		positions: resource.NewValue(
			resource.WithInitialValue(positions),
			resource.WithMessageEquivalence(cmp.Equal(cmp.FloatValueApprox(0, 0.01))),
		),
	}

	return model
}

// Positions gets the current positions
func (m *Model) Positions(opts ...resource.ReadOption) *traits.OpenClosePositions {
	return m.positions.Get(opts...).(*traits.OpenClosePositions)
}

// UpdatePositions changes the positions
// Provide write options to support features like relative changes.
// Related properties of the positions will be set automatically,
// for example if the open_percent is set, then target_open_percent will be modified accordingly.
// Providing a resource.InterceptAfter option will disable related property updating.
// Using DeriveValues can re-enable related property computation if needed.
func (m *Model) UpdatePositions(positions *traits.OpenClosePositions, opts ...resource.WriteOption) (
	*traits.OpenClosePositions, error,
) {
	if err := m.validateUpdate(positions); err != nil {
		return nil, err
	}

	opts = append([]resource.WriteOption{resource.InterceptAfter(m.DeriveValues)}, opts...)
	val, err := m.positions.Set(positions, opts...)
	if val == nil {
		return nil, err
	}
	return val.(*traits.OpenClosePositions), err
}

func (m *Model) validateUpdate(positions *traits.OpenClosePositions) error {
	for _, position := range positions.GetStates() {
		percent := position.GetOpenPercent()
		if percent < 0 || percent > 100 {
			return status.Errorf(codes.InvalidArgument, "open_percent %f is out of range 0-100", percent)
		}

		if position.GetOpenPercentTween() != nil {
			return status.Error(codes.Unimplemented, "tweening support not implemented")
		}
	}
	return nil
}

// DeriveValues derives values that haven't been set from those that have.
// For example setting percentage when the preset changes.
// Updated values will be set on new.
func (m *Model) DeriveValues(old, new proto.Message) {
	newVal := new.(*traits.OpenClosePositions)

	for _, position := range newVal.GetStates() {
		position.TargetOpenPercent = position.OpenPercent
	}

}

//goland:noinspection GoNameStartsWithPackageName
type OpenCloseChange struct {
	Value      *traits.OpenClosePositions
	ChangeTime time.Time
}

func (m *Model) PullPositions(ctx context.Context, opts ...resource.ReadOption) <-chan OpenCloseChange {
	send := make(chan OpenCloseChange)
	go func() {
		defer close(send)
		for change := range m.positions.Pull(ctx, opts...) {
			val := change.Value.(*traits.OpenClosePositions)
			select {
			case <-ctx.Done():
				return
			case send <- OpenCloseChange{Value: val, ChangeTime: change.ChangeTime}:
			}
		}
	}()
	return send
}
