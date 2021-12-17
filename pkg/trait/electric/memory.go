package electric

import (
	"context"
	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"github.com/smart-core-os/sc-golang/pkg/memory"
	"github.com/smart-core-os/sc-golang/pkg/time/clock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"math/rand"
	"sync"
	"time"
)

var (
	ErrModeNotFound     = status.Error(codes.NotFound, "electric mode not found")
	ErrNormalModeExists = status.Error(codes.AlreadyExists, "a normal electric mode already exists")
	ErrDeleteActiveMode = status.Error(codes.FailedPrecondition, "attempt to delete active mode")
)

// Memory is a simple data store for electric devices. It simply stores the data given to it, and does not implement
// any business logic.
// For the implementation of the gRPC trait based on Memory, see MemoryDevice.
// Invariants:
//   1. At most one mode has normal = true.
//   2. The active mode cannot be deleted.
//   3. Only a mode that exists can be active (except when the Memory is first created, when a dummy mode is active)
type Memory struct {
	demand     *memory.Resource // of *traits.ElectricDemand
	activeMode *memory.Resource // of *traits.ElectricMode
	modes      *memory.Collection

	// mu protects invariants
	mu    sync.RWMutex
	clock clock.Clock
	Rng   *rand.Rand // for generating mode ids
}

// NewMemory constructs a Memory with default values:
//	 Current: 0
//   Voltage: 240
//   Rating: 13
// No modes, active mode is empty.
func NewMemory(clk clock.Clock) *Memory {
	var voltage float32 = 240
	demand := &traits.ElectricDemand{
		Current: 0,
		Voltage: &voltage,
		Rating:  13,
	}
	*demand.Voltage = 240

	mode := &traits.ElectricMode{}

	mem := &Memory{
		demand:     memory.NewResource(memory.WithInitialValue(demand)),
		activeMode: memory.NewResource(memory.WithInitialValue(mode)),
		modes:      memory.NewCollection(memory.WithClockCollection(clk)),
		clock:      clk,
		Rng:        rand.New(rand.NewSource(rand.Int63())),
	}

	return mem
}

// Demand gets the demand stored in this Memory.
// The fields returned can be filtered by mask - if you want all fields, pass an empty mask.
func (m *Memory) Demand(mask *fieldmaskpb.FieldMask) *traits.ElectricDemand {
	return m.demand.Get(memory.WithGetMask(mask)).(*traits.ElectricDemand)
}

func (m *Memory) PullDemand(ctx context.Context, mask *fieldmaskpb.FieldMask) (changes <-chan PullDemandChange, done func()) {
	send := make(chan PullDemandChange)

	recv, done := m.demand.OnUpdate(ctx)
	go func() {
		filter := masks.NewResponseFilter(masks.WithFieldMask(mask))

		for change := range recv {
			demand := filter.FilterClone(change.Value).(*traits.ElectricDemand)
			send <- PullDemandChange{
				Value:      demand,
				ChangeTime: change.ChangeTime.AsTime(),
			}
		}
	}()

	// when done is called, then the resource will close recv for us
	return send, done
}

func (m *Memory) UpdateDemand(update *traits.ElectricDemand, mask *fieldmaskpb.FieldMask) (*traits.ElectricDemand, error) {
	updated, err := m.demand.Set(update, memory.WithUpdateMask(mask))
	if err != nil {
		return nil, err
	}
	return updated.(*traits.ElectricDemand), nil
}

func (m *Memory) ActiveMode(mask *fieldmaskpb.FieldMask) *traits.ElectricMode {
	return m.activeMode.Get(memory.WithGetMask(mask)).(*traits.ElectricMode)
}

func (m *Memory) PullActiveMode(ctx context.Context, mask *fieldmaskpb.FieldMask) (changes <-chan PullActiveModeChange, done func()) {
	send := make(chan PullActiveModeChange)

	recv, done := m.activeMode.OnUpdate(ctx)
	go func() {
		defer close(send)
		filter := masks.NewResponseFilter(masks.WithFieldMask(mask))

		for change := range recv {
			activeMode := filter.FilterClone(change.Value).(*traits.ElectricMode)
			send <- PullActiveModeChange{
				ActiveMode: activeMode,
				ChangeTime: change.ChangeTime.AsTime(),
			}
		}
	}()

	// when done is called, then the resource will close recv for us
	return send, done
}

func (m *Memory) ChangeActiveMode(id string) (*traits.ElectricMode, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.changeActiveMode(id)
}

func (m *Memory) changeActiveMode(id string) (*traits.ElectricMode, error) {
	mode, ok := m.findMode(id)
	if !ok {
		return nil, ErrModeNotFound
	}

	// clone mode to prevent modifying shared copy accidentally
	mode = proto.Clone(mode).(*traits.ElectricMode)
	mode.StartTime = timestamppb.New(m.clock.Now()) // set the reference time

	updated, err := m.activeMode.Set(mode)
	if err != nil {
		return nil, err
	}

	return updated.(*traits.ElectricMode), nil
}

func (m *Memory) ChangeToNormalMode() (*traits.ElectricMode, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	normal, ok := m.normalMode()
	if !ok {
		return nil, ErrModeNotFound
	}

	return m.changeActiveMode(normal.Id)
}

func (m *Memory) FindMode(id string) (*traits.ElectricMode, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.findMode(id)
}

func (m *Memory) findMode(id string) (*traits.ElectricMode, bool) {
	mode, ok := m.modes.Get(id)
	if !ok {
		return nil, false
	}
	return mode.(*traits.ElectricMode), true
}

func (m *Memory) Modes(mask *fieldmaskpb.FieldMask) []*traits.ElectricMode {
	entries := m.modes.List()
	filter := masks.NewResponseFilter(masks.WithFieldMask(mask))

	var modes []*traits.ElectricMode
	for _, entry := range entries {
		mode := filter.FilterClone(entry)
		modes = append(modes, mode.(*traits.ElectricMode))
	}
	return modes
}

func (m *Memory) CreateMode(mode *traits.ElectricMode) (*traits.ElectricMode, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.createMode(mode)
}

func (m *Memory) createMode(mode *traits.ElectricMode) (*traits.ElectricMode, error) {
	// clone mode to avoid mutating the caller's copy
	mode = proto.Clone(mode).(*traits.ElectricMode)

	if mode.Id != "" {
		// If the ID is set, this indicates a bug in the calling code
		panic("ID field is set")
	}

	// if this mode is normal, check that there isn't another normal mode
	if mode.Normal {
		_, ok := m.normalMode()
		if ok {
			return nil, ErrNormalModeExists
		}
	}

	msg, err := m.modes.AddFn(func(id string) proto.Message {
		mode.Id = id
		return mode
	})
	if err != nil {
		return nil, err
	}
	mode = msg.(*traits.ElectricMode)

	return mode, nil
}

func (m *Memory) DeleteMode(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.deleteMode(id)
}

func (m *Memory) deleteMode(id string) error {
	active := m.activeMode.Get().(*traits.ElectricMode)
	if id == active.Id {
		return ErrDeleteActiveMode
	}

	msg := m.modes.Delete(id)
	if msg == nil {
		return ErrModeNotFound
	}

	return nil
}

func (m *Memory) UpdateMode(mode *traits.ElectricMode, mask *fieldmaskpb.FieldMask) (*traits.ElectricMode, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updateMode(mode, mask)
}

func (m *Memory) updateMode(mode *traits.ElectricMode, mask *fieldmaskpb.FieldMask) (*traits.ElectricMode, error) {
	msg, err := m.modes.Update(mode.Id, func(oldMsg, updatedMsg proto.Message) error {
		writer := masks.NewFieldUpdater(
			masks.WithUpdateMask(mask),
		)
		if err := writer.Validate(updatedMsg); err != nil {
			return err
		}

		writer.Merge(updatedMsg, mode)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return msg.(*traits.ElectricMode), nil
}

func (m *Memory) PullModes(ctx context.Context, mask *fieldmaskpb.FieldMask) (changes <-chan PullModesChange, done func()) {
	send := make(chan PullModesChange)
	recv, done := m.modes.PullChanges(ctx)

	go func() {
		defer close(send)
		filter := masks.NewResponseFilter(masks.WithFieldMask(mask))

		// no need to listen to ctx.Done, as modes.PullChanges does that.
		for change := range recv {
			newValue := filter.FilterClone(change.NewValue)
			oldValue := filter.FilterClone(change.OldValue)

			send <- PullModesChange{
				Type:       change.ChangeType,
				NewValue:   newValue.(*traits.ElectricMode),
				OldValue:   oldValue.(*traits.ElectricMode),
				ChangeTime: change.ChangeTime,
			}
		}
	}()

	// when the caller invokes done, then recv will automatically be closed
	return send, done
}

func (m *Memory) NormalMode() (*traits.ElectricMode, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.normalMode()
}

func (m *Memory) normalMode() (*traits.ElectricMode, bool) {
	modes := m.modes.List()

	for _, mode := range modes {
		mode := mode.(*traits.ElectricMode)
		if mode.Normal {
			return mode, true
		}
	}

	return nil, false
}

type PullModesChange struct {
	Type       types.ChangeType
	NewValue   *traits.ElectricMode
	OldValue   *traits.ElectricMode
	ChangeTime time.Time
}

type PullDemandChange struct {
	Value      *traits.ElectricDemand
	ChangeTime time.Time
}

type PullActiveModeChange struct {
	ActiveMode *traits.ElectricMode
	ChangeTime time.Time
}
