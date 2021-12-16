package electric

import (
	"context"
	"log"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/olebedev/emitter"
	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"github.com/smart-core-os/sc-golang/pkg/memory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type MemoryDevice struct {
	traits.UnimplementedElectricApiServer
	UnimplementedMemorySettingsApiServer

	demand     *memory.Resource // of *traits.ElectricDemand
	activeMode *memory.Resource // of *traits.ElectricMode

	modesById   map[string]*electricMode
	modesByIdMu sync.RWMutex     // guards above
	bus         *emitter.Emitter // "change" emits *traits.PullModesResponse_Change
	Rng         *rand.Rand       // for generating mode ids
}

func NewMemoryDevice() *MemoryDevice {
	var voltage float32 = 240
	return &MemoryDevice{
		demand:     memory.NewResource(memory.WithInitialValue(&traits.ElectricDemand{Voltage: &voltage, Current: 0, Rating: 13})),
		activeMode: memory.NewResource(memory.WithInitialValue(&traits.ElectricMode{})),
		modesById:  map[string]*electricMode{},
		bus:        &emitter.Emitter{},
		Rng:        rand.New(rand.NewSource(rand.Int63())),
	}
}

func (d *MemoryDevice) Register(server *grpc.Server) {
	traits.RegisterElectricApiServer(server, d)
}

func (d *MemoryDevice) GetDemand(_ context.Context, request *traits.GetDemandRequest) (*traits.ElectricDemand, error) {
	res := d.demand.Get(
		memory.WithGetMask(request.GetReadMask()),
	)
	return res.(*traits.ElectricDemand), nil
}

func (d *MemoryDevice) PullDemand(request *traits.PullDemandRequest, server traits.ElectricApi_PullDemandServer) error {
	updates, done := d.demand.OnUpdate(server.Context())
	defer done()

	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case event := <-updates:
			change := &traits.PullDemandResponse_Change{
				Name:       request.Name,
				Demand:     event.Value.(*traits.ElectricDemand),
				ChangeTime: event.ChangeTime,
			}
			err := server.Send(&traits.PullDemandResponse{
				Changes: []*traits.PullDemandResponse_Change{change},
			})
			if err != nil {
				return err
			}
		}
	}
}

func (d *MemoryDevice) GetActiveMode(_ context.Context, request *traits.GetActiveModeRequest) (*traits.ElectricMode, error) {
	res := d.activeMode.Get(memory.WithGetMask(request.GetReadMask()))
	return res.(*traits.ElectricMode), nil
}

func (d *MemoryDevice) UpdateActiveMode(ctx context.Context, request *traits.UpdateActiveModeRequest) (*traits.ElectricMode, error) {
	mode := request.GetActiveMode()
	// hydrate the mode using the list of known modes (by id)
	if mode.GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Id should be present during update")
	}
	// Note: we hold this lock while we acquire the lock during the d.activeMode.Set
	d.modesByIdMu.RLock()
	defer d.modesByIdMu.RUnlock()

	knownMode, modeKnown := d.modesById[mode.GetId()]
	if !modeKnown {
		return nil, status.Errorf(codes.NotFound, "mode '%v' is unknown", mode.GetId())
	}

	if len(request.GetUpdateMask().GetPaths()) == 0 {
		// default to just replacing the mode
		mode = knownMode.msg
	} else {
		// clone and apply changes based on the update mask and known mode
		updater := masks.NewFieldUpdater(masks.WithUpdateMask(request.GetUpdateMask()))
		if err := updater.Validate(mode); err != nil {
			return nil, err
		}
		clone := proto.Clone(knownMode.msg).(*traits.ElectricMode)
		updater.Merge(clone, mode)
		mode = clone
	}

	res, err := d.activeMode.Set(mode)
	if err != nil {
		return nil, err
	}
	return res.(*traits.ElectricMode), nil
}

func (d *MemoryDevice) ClearActiveMode(ctx context.Context, request *traits.ClearActiveModeRequest) (*traits.ElectricMode, error) {
	d.modesByIdMu.RLock()
	defer d.modesByIdMu.RUnlock()

	mode := d.normalMode()
	if mode == nil {
		return nil, status.Error(codes.FailedPrecondition, "no modes")
	}

	res, err := d.activeMode.Set(mode, memory.WithAllFieldsWritable())
	if err != nil {
		return nil, err
	}
	return res.(*traits.ElectricMode), nil
}

// normalMode selects and returns the normal operation mode for the device.
// modesByIdMu should be held when calling this method.
// Can return nil if there are no modes.
func (d *MemoryDevice) normalMode() *traits.ElectricMode {
	// Use sorted modes to make the following deterministic, go maps are randomly ordered.
	modes := d.sortedModes()
	for _, mode := range modes {
		if mode.msg.Normal {
			return mode.msg
		}
	}

	if len(modes) > 0 {
		return modes[0].msg
	}

	return nil
}

func (d *MemoryDevice) PullActiveMode(request *traits.PullActiveModeRequest, server traits.ElectricApi_PullActiveModeServer) error {
	updates, done := d.activeMode.OnUpdate(server.Context())
	defer done()

	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case event := <-updates:
			change := &traits.PullActiveModeResponse_Change{
				Name:       request.Name,
				ActiveMode: event.Value.(*traits.ElectricMode),
				ChangeTime: event.ChangeTime,
			}
			err := server.Send(&traits.PullActiveModeResponse{
				Changes: []*traits.PullActiveModeResponse_Change{change},
			})
			if err != nil {
				return err
			}
		}
	}
}

func (d *MemoryDevice) ListModes(_ context.Context, request *traits.ListModesRequest) (*traits.ListModesResponse, error) {
	filter := masks.NewResponseFilter(masks.WithFieldMask(request.GetReadMask()))
	if err := filter.Validate(&traits.ElectricMode{}); err != nil {
		return nil, err
	}
	d.modesByIdMu.RLock()
	defer d.modesByIdMu.RUnlock()

	pageToken := &types.PageToken{}
	if err := decodePageToken(request.PageToken, pageToken); err != nil {
		return nil, err
	}

	lastKey := pageToken.GetLastResourceName() // the key() of the last item we sent
	pageSize := capPageSize(int(request.GetPageSize()))

	sortedModes := d.sortedModes()
	nextIndex := 0
	if lastKey != "" {
		nextIndex = sort.Search(len(sortedModes), func(i int) bool {
			return sortedModes[i].key() > lastKey
		})
	}

	result := &traits.ListModesResponse{
		TotalSize: int32(len(sortedModes)),
	}
	upperBound := nextIndex + pageSize
	if upperBound > len(sortedModes) {
		upperBound = len(sortedModes)
		pageToken = nil
	} else {
		pageToken.PageStart = &types.PageToken_LastResourceName{
			LastResourceName: sortedModes[upperBound-1].key(),
		}
	}

	var err error
	result.NextPageToken, err = encodePageToken(pageToken)
	if err != nil {
		return nil, err
	}
	for _, n := range sortedModes[nextIndex:upperBound] {
		msg := filter.FilterClone(n.msg).(*traits.ElectricMode)
		result.Modes = append(result.Modes, msg)
	}
	return result, nil
}

// sortedModes returns a slice of electricMode sorted by m.key().
// The lock must be held when calling this method.
func (d *MemoryDevice) sortedModes() []*electricMode {
	sorted := make([]*electricMode, 0, len(d.modesById))
	for _, notification := range d.modesById {
		sorted = append(sorted, notification)
	}
	sort.Slice(sorted, func(i, j int) bool {
		left, right := sorted[i].key(), sorted[j].key()
		return left < right
	})
	return sorted
}

func (d *MemoryDevice) PullModes(request *traits.PullModesRequest, server traits.ElectricApi_PullModesServer) error {
	changes := d.bus.On("change")
	defer d.bus.Off("change", changes)
	id := rand.Int()
	t0 := time.Now()
	sentItems := 0
	defer func() {
		log.Printf("[%x] PullModes done in %v: sent %v", id, time.Now().Sub(t0), sentItems)
	}()
	log.Printf("[%x] PullModes start %v", id, request)

	// watch for changes to the modes list and emit when one matches our query
	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case event := <-changes:
			change := event.Args[0].(*traits.PullModesResponse_Change)
			sendChange := modeChangeToRequestScope(request, change)
			if sendChange != nil {
				if err := server.Send(&traits.PullModesResponse{Changes: []*traits.PullModesResponse_Change{
					sendChange,
				}}); err != nil {
					return err
				}
				sentItems++
			}
		}
	}
}

// modeChangeToRequestScope converts a device scoped change into a request scoped change.
// Effectively it makes it so if you are using a read_mask then you only get notified if one of the requested properties
// changes.
// This also applies the mask to the Old and New values in the change.
func modeChangeToRequestScope(request *traits.PullModesRequest, change *traits.PullModesResponse_Change) *traits.PullModesResponse_Change {
	mask := request.GetReadMask()
	if mask != nil {
		filter := masks.NewResponseFilter(masks.WithFieldMask(mask))
		oldValue := filter.FilterClone(change.OldValue).(*traits.ElectricMode)
		newValue := filter.FilterClone(change.NewValue).(*traits.ElectricMode)

		if proto.Equal(oldValue, newValue) {
			return nil
		}

		// return a clone so we don't modify the original event, others might be listening
		clone := proto.Clone(change).(*traits.PullModesResponse_Change)
		clone.OldValue = oldValue
		clone.NewValue = newValue
		return clone
	}
	return change
}
