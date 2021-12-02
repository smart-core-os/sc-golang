package electric

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"github.com/smart-core-os/sc-golang/pkg/memory"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:generate protoc -I../../../ -I../../../.protomod -I../../../.protomod/github.com/smart-core-os/sc-api/protobuf --go_out=paths=source_relative:../../../ --go-grpc_out=paths=source_relative:../../../ pkg/trait/electric/memory_settings.proto

func (d *MemoryDevice) UpdateDemand(_ context.Context, request *UpdateDemandRequest) (*traits.ElectricDemand, error) {
	res, err := d.demand.Set(request.GetDemand(),
		memory.WithMoreWritablePaths("current"),
		memory.WithUpdateMask(request.GetUpdateMask()),
	)
	if err != nil {
		return nil, err
	}
	return res.(*traits.ElectricDemand), nil
}

func (d *MemoryDevice) CreateMode(_ context.Context, request *CreateModeRequest) (*traits.ElectricMode, error) {
	// start by validating things
	if request.GetMode().GetId() != "" {
		return nil, status.Errorf(codes.InvalidArgument, "id '%v' should be empty", request.GetMode().GetId())
	}

	d.modesByIdMu.Lock()
	defer d.modesByIdMu.Unlock()

	mode := request.GetMode()
	if err := d.generateId(mode); err != nil {
		return nil, err
	}
	d.modesById[mode.Id] = &electricMode{
		msg:        mode,
		createTime: time.Now(),
	}
	d.bus.Emit("change", &traits.PullModesResponse_Change{
		Name:       request.Name,
		Type:       types.ChangeType_ADD,
		ChangeTime: timestamppb.Now(),
		NewValue:   mode,
	})
	return mode, nil
}

func (d *MemoryDevice) UpdateMode(_ context.Context, request *UpdateModeRequest) (*traits.ElectricMode, error) {
	// start by validating things
	if request.GetMode().GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	d.modesByIdMu.Lock()
	defer d.modesByIdMu.Unlock()

	newMode := request.GetMode()
	oldMode, exists := d.modesById[newMode.GetId()]
	if !exists {
		return nil, status.Errorf(codes.NotFound, "mode '%v' not found", newMode.GetId())
	}
	fieldUpdater := masks.NewFieldUpdater(masks.WithUpdateMask(request.GetUpdateMask()))
	if err := fieldUpdater.Validate(newMode); err != nil {
		return nil, err
	}

	// Don't merge in place as we're going to be sending a change event with old and new values.
	// Also, we don't want to update active mode without sending a notification.
	oldModeMsg := oldMode.msg
	newModeMsg := proto.Clone(oldModeMsg).(*traits.ElectricMode)
	fieldUpdater.Merge(newModeMsg, newMode)
	oldMode.msg = newModeMsg

	d.bus.Emit("change", &traits.PullModesResponse_Change{
		Name:       request.Name,
		Type:       types.ChangeType_UPDATE,
		ChangeTime: timestamppb.Now(),
		OldValue:   oldModeMsg,
		NewValue:   newModeMsg,
	})
	return newModeMsg, nil
}

func (d *MemoryDevice) DeleteMode(_ context.Context, request *DeleteModeRequest) (*emptypb.Empty, error) {
	// start by validating things
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	d.modesByIdMu.Lock()
	defer d.modesByIdMu.Unlock()

	oldMode, exists := d.modesById[request.GetId()]
	if !exists {
		if request.GetAllowMissing() {
			return &emptypb.Empty{}, nil
		} else {
			return nil, status.Errorf(codes.NotFound, "mode '%v' not found", request.GetId())
		}
	}

	delete(d.modesById, request.GetId())
	d.bus.Emit("change", &traits.PullModesResponse_Change{
		Name:       request.Name,
		Type:       types.ChangeType_REMOVE,
		ChangeTime: timestamppb.Now(),
		OldValue:   oldMode.msg,
	})
	return &emptypb.Empty{}, nil
}

// generateId assigns a unique id to the given ElectricMode.
// d.modesByIdMu must be locked before calling.
func (d *MemoryDevice) generateId(m *traits.ElectricMode) error {
	id, err := memory.GenerateUniqueId(d.Rng, func(candidate string) bool {
		_, ok := d.modesById[candidate]
		return ok
	})
	if err != nil {
		return err
	}
	m.Id = id
	return nil
}
