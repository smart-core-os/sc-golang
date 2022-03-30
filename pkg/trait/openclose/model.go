package openclose

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-golang/pkg/cmp"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// Model provides a data structure for supporting the Open Close trait.
type Model struct {
	positions *resource.Value // of *traits.OpenClosePositions
}

// NewModel constructs a Model with default values
func NewModel() *Model {
	position := &traits.OpenClosePosition{}
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
func (m *Model) UpdatePositions(positions *traits.OpenClosePositions, opts ...resource.WriteOption) (*traits.OpenClosePositions, error) {
	if err := m.validateUpdate(positions); err != nil {
		return nil, err
	}

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
	}
	return nil
}

type PositionsChange struct {
	Value      *traits.OpenClosePositions
	ChangeTime time.Time
}

func (m *Model) PullPositions(ctx context.Context, opts ...resource.ReadOption) <-chan PositionsChange {
	send := make(chan PositionsChange)
	go func() {
		defer close(send)
		for change := range m.positions.Pull(ctx, opts...) {
			val := change.Value.(*traits.OpenClosePositions)
			select {
			case <-ctx.Done():
				return
			case send <- PositionsChange{Value: val, ChangeTime: change.ChangeTime}:
			}
		}
	}()
	return send
}
