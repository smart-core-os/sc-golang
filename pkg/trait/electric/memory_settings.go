package electric

import (
	"context"
	"errors"
	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

//go:generate protoc -I../../../ -I../../../.protomod -I../../../.protomod/github.com/smart-core-os/sc-api/protobuf --go_out=paths=source_relative:../../../ --go-grpc_out=paths=source_relative:../../../ pkg/trait/electric/memory_settings.proto

func (d *MemoryDevice) UpdateDemand(_ context.Context, request *UpdateDemandRequest) (*traits.ElectricDemand, error) {
	return d.memory.UpdateDemand(request.Demand, request.UpdateMask)
}

func (d *MemoryDevice) CreateMode(_ context.Context, request *CreateModeRequest) (*traits.ElectricMode, error) {
	// start by validating things
	if request.GetMode().GetId() != "" {
		return nil, status.Errorf(codes.InvalidArgument, "id '%v' should be empty", request.GetMode().GetId())
	}

	return d.memory.CreateMode(request.Mode)
}

func (d *MemoryDevice) UpdateMode(_ context.Context, request *UpdateModeRequest) (*traits.ElectricMode, error) {
	// start by validating things
	if request.GetMode().GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	return d.memory.UpdateMode(request.Mode, request.UpdateMask)
}

func (d *MemoryDevice) DeleteMode(_ context.Context, request *DeleteModeRequest) (*emptypb.Empty, error) {
	// start by validating things
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	err := d.memory.DeleteMode(request.Id)
	if request.AllowMissing && errors.Is(err, ErrModeNotFound) {
		// the client specified that deleting a non-existent mode is OK and should not error
		err = nil
	}
	return &emptypb.Empty{}, err
}
