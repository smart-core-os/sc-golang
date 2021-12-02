package electric

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// WrapMemorySettings adapts a MemorySettingsApiServer and presents it as a MemorySettingsApiClient
func WrapMemorySettings(server MemorySettingsApiServer) MemorySettingsApiClient {
	return &memorySettingWrapper{server}
}

type memorySettingWrapper struct {
	server MemorySettingsApiServer
}

// compile time check that we implement the interface we need
var _ MemorySettingsApiClient = (*memorySettingWrapper)(nil)

func (w *memorySettingWrapper) UpdateDemand(ctx context.Context, in *UpdateDemandRequest, opts ...grpc.CallOption) (*traits.ElectricDemand, error) {
	return w.server.UpdateDemand(ctx, in)
}

func (w *memorySettingWrapper) CreateMode(ctx context.Context, in *CreateModeRequest, opts ...grpc.CallOption) (*traits.ElectricMode, error) {
	return w.server.CreateMode(ctx, in)
}

func (w *memorySettingWrapper) UpdateMode(ctx context.Context, in *UpdateModeRequest, opts ...grpc.CallOption) (*traits.ElectricMode, error) {
	return w.server.UpdateMode(ctx, in)
}

func (w *memorySettingWrapper) DeleteMode(ctx context.Context, in *DeleteModeRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return w.server.DeleteMode(ctx, in)
}
