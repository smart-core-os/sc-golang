package electric

import (
	"context"
	"errors"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/smart-core-os/sc-golang/pkg/memory"
)

//go:generate protomod protoc -- -I../../.. --go_out=paths=source_relative:../../../ --go-grpc_out=paths=source_relative:../../../ pkg/trait/electric/memory_settings.proto

func (d *ModelServer) UpdateDemand(_ context.Context, request *UpdateDemandRequest) (*traits.ElectricDemand, error) {
	return d.model.UpdateDemand(request.Demand, memory.WithUpdateMask(request.UpdateMask))
}

func (d *ModelServer) CreateMode(_ context.Context, request *CreateModeRequest) (*traits.ElectricMode, error) {
	// start by validating things
	if request.GetMode().GetId() != "" {
		return nil, status.Errorf(codes.InvalidArgument, "id '%v' should be empty", request.GetMode().GetId())
	}

	return d.model.CreateMode(request.Mode)
}

func (d *ModelServer) UpdateMode(_ context.Context, request *UpdateModeRequest) (*traits.ElectricMode, error) {
	// start by validating things
	if request.GetMode().GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	return d.model.UpdateMode(request.Mode, request.UpdateMask)
}

func (d *ModelServer) DeleteMode(_ context.Context, request *DeleteModeRequest) (*emptypb.Empty, error) {
	// start by validating things
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	err := d.model.DeleteMode(request.Id)
	if request.AllowMissing && errors.Is(err, ErrModeNotFound) {
		// the client specified that deleting a non-existent mode is OK and should not error
		err = nil
	}
	return &emptypb.Empty{}, err
}
