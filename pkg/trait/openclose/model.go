package openclose

import (
	"context"
	"sort"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/cmp"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	positions *resource.Collection // of *traits.OpenClosePosition
}

func NewModel(opts ...resource.Option) *Model {
	return &Model{
		positions: resource.NewCollection(opts...),
	}
}

func (m *Model) GetPositions(opts ...resource.ReadOption) (*traits.OpenClosePositions, error) {
	allPositions := m.positions.List(opts...)
	dst := &traits.OpenClosePositions{
		States: make([]*traits.OpenClosePosition, len(allPositions)),
	}
	for i, position := range allPositions {
		dst.States[i] = position.(*traits.OpenClosePosition)
	}
	return dst, nil
}

func (m *Model) GetPosition(id string, opts ...resource.ReadOption) (*traits.OpenClosePosition, error) {
	position, ok := m.positions.Get(id, opts...)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "position %s not found", id)
	}
	return position.(*traits.OpenClosePosition), nil
}

func (m *Model) UpdatePositions(positions *traits.OpenClosePositions, opts ...resource.WriteOption) (*traits.OpenClosePositions, error) {
	writeRequest := resource.ComputeWriteConfig(opts...)
	opts = append([]resource.WriteOption{}, opts...)
	if writeRequest.UpdateMask != nil {
		opts = append(opts, resource.WithUpdateMask(masks.RemovePrefix("states", writeRequest.UpdateMask)))
	}
	opts = append(opts, resource.WithCreateIfAbsent())

	for i, state := range positions.States {
		_, err := m.positions.Update(strconv.Itoa(i), state, opts...)
		if err != nil {
			return nil, err
		}
	}

	return m.GetPositions()
}

func (m *Model) UpdatePosition(position *traits.OpenClosePosition, opts ...resource.WriteOption) (*traits.OpenClosePosition, error) {
	return m.UpdatePositionN(0, position, opts...)
}

func (m *Model) UpdatePositionN(id uint, position *traits.OpenClosePosition, opts ...resource.WriteOption) (*traits.OpenClosePosition, error) {
	msg, err := m.positions.Update(strconv.Itoa(int(id)), position, opts...)
	if err != nil {
		return nil, err
	}
	return msg.(*traits.OpenClosePosition), nil
}

func (m *Model) PullPositions(ctx context.Context, ops ...resource.ReadOption) <-chan PullOpenClosePositionsChange {
	readRequest := resource.ComputeReadConfig(ops...)
	responseFilter := readRequest.ResponseFilter()
	eq := cmp.Equal()

	send := make(chan PullOpenClosePositionsChange)
	go func() {
		defer close(send)
		all := make(map[string]*traits.OpenClosePosition, 1)
		seenAll := false
		var last *traits.OpenClosePositions

		for change := range m.positions.Pull(ctx) {
			if change.NewValue == nil {
				delete(all, change.Id)
			} else {
				all[change.Id] = change.NewValue.(*traits.OpenClosePosition)
			}

			shouldSend := seenAll || (change.LastSeedValue && !readRequest.UpdatesOnly)
			if change.LastSeedValue {
				seenAll = true
			}
			if !shouldSend {
				continue
			}

			// transform into the correct output format
			positions := &traits.OpenClosePositions{
				States: make([]*traits.OpenClosePosition, len(all)),
			}
			order := make([]string, len(all))
			i := 0
			for id := range all {
				order[i] = id
				i++
			}
			sort.Strings(order)
			for i, id := range order {
				positions.States[i] = all[id]
			}

			// projection and filtering
			responseFilter.Filter(positions)
			if eq(last, positions) {
				continue
			}
			last = positions

			// do the send
			select {
			case <-ctx.Done():
				return
			case send <- PullOpenClosePositionsChange{
				Positions:  positions,
				ChangeTime: change.ChangeTime,
			}:
			}
		}
	}()
	return send
}

type PullOpenClosePositionsChange struct {
	Positions  *traits.OpenClosePositions
	ChangeTime time.Time
}
