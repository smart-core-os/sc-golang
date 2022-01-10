// Code generated by protoc-gen-router. DO NOT EDIT.

package electric

import (
	context "context"
	traits "github.com/smart-core-os/sc-api/go/traits"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	sync "sync"
)

// MemorySettingsApiRouter is a MemorySettingsApiServer that allows routing named requests to specific MemorySettingsApiClient
type MemorySettingsApiRouter struct {
	UnimplementedMemorySettingsApiServer

	mu       sync.Mutex
	registry map[string]MemorySettingsApiClient
	// Factory can be used to dynamically create api clients if requests come in for devices we haven't seen.
	Factory func(string) (MemorySettingsApiClient, error)
}

// compile time check that we implement the interface we need
var _ MemorySettingsApiServer = (*MemorySettingsApiRouter)(nil)

func NewMemorySettingsApiRouter() *MemorySettingsApiRouter {
	return &MemorySettingsApiRouter{
		registry: make(map[string]MemorySettingsApiClient),
	}
}

func (r *MemorySettingsApiRouter) Register(server *grpc.Server) {
	RegisterMemorySettingsApiServer(server, r)
}

func (r *MemorySettingsApiRouter) Add(name string, client MemorySettingsApiClient) MemorySettingsApiClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	r.registry[name] = client
	return old
}

func (r *MemorySettingsApiRouter) Remove(name string) MemorySettingsApiClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	delete(r.registry, name)
	return old
}

func (r *MemorySettingsApiRouter) Has(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, exists := r.registry[name]
	return exists
}

func (r *MemorySettingsApiRouter) Get(name string) (MemorySettingsApiClient, error) {
	r.mu.Lock()
	child, exists := r.registry[name]
	defer r.mu.Unlock()
	if !exists {
		if r.Factory != nil {
			child, err := r.Factory(name)
			if err != nil {
				return nil, err
			}
			r.registry[name] = child
			return child, nil
		}
		return nil, status.Error(codes.NotFound, name)
	}
	return child, nil
}

func (r *MemorySettingsApiRouter) UpdateDemand(ctx context.Context, request *UpdateDemandRequest) (*traits.ElectricDemand, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.UpdateDemand(ctx, request)
}

func (r *MemorySettingsApiRouter) CreateMode(ctx context.Context, request *CreateModeRequest) (*traits.ElectricMode, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.CreateMode(ctx, request)
}

func (r *MemorySettingsApiRouter) UpdateMode(ctx context.Context, request *UpdateModeRequest) (*traits.ElectricMode, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.UpdateMode(ctx, request)
}

func (r *MemorySettingsApiRouter) DeleteMode(ctx context.Context, request *DeleteModeRequest) (*emptypb.Empty, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.DeleteMode(ctx, request)
}