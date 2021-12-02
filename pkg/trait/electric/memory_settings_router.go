package electric

import (
	"context"
	"sync"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// MemorySettingsRouter is a MemorySettingsApiServer that allows routing named requests to specific MemorySettingsApiClients
type MemorySettingsRouter struct {
	UnimplementedMemorySettingsApiServer

	mu       sync.Mutex
	registry map[string]MemorySettingsApiClient
	// Factory can be used to dynamically create api clients if requests come in for devices we haven't seen.
	Factory func(string) (MemorySettingsApiClient, error)
}

// compile time check that we implement the interface we need
var _ MemorySettingsApiServer = (*MemorySettingsRouter)(nil)

func NewMemorySettingsRouter() *MemorySettingsRouter {
	return &MemorySettingsRouter{
		registry: make(map[string]MemorySettingsApiClient),
	}
}

func (r *MemorySettingsRouter) Register(server *grpc.Server) {
	RegisterMemorySettingsApiServer(server, r)
}

func (r *MemorySettingsRouter) Add(name string, client MemorySettingsApiClient) MemorySettingsApiClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	r.registry[name] = client
	return old
}

func (r *MemorySettingsRouter) Remove(name string) MemorySettingsApiClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	delete(r.registry, name)
	return old
}

func (r *MemorySettingsRouter) Has(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, exists := r.registry[name]
	return exists
}

func (r *MemorySettingsRouter) Get(name string) (MemorySettingsApiClient, error) {
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

func (r *MemorySettingsRouter) UpdateDemand(ctx context.Context, request *UpdateDemandRequest) (*traits.ElectricDemand, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.UpdateDemand(ctx, request)
}

func (r *MemorySettingsRouter) CreateMode(ctx context.Context, request *CreateModeRequest) (*traits.ElectricMode, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.CreateMode(ctx, request)
}

func (r *MemorySettingsRouter) UpdateMode(ctx context.Context, request *UpdateModeRequest) (*traits.ElectricMode, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.UpdateMode(ctx, request)
}

func (r *MemorySettingsRouter) DeleteMode(ctx context.Context, request *DeleteModeRequest) (*emptypb.Empty, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.DeleteMode(ctx, request)
}
