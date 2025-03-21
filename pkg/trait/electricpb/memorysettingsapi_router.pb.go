// Code generated by protoc-gen-router. DO NOT EDIT.

package electricpb

import (
	context "context"
	fmt "fmt"
	traits "github.com/smart-core-os/sc-api/go/traits"
	router "github.com/smart-core-os/sc-golang/pkg/router"
	grpc "google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// MemorySettingsApiRouter is a MemorySettingsApiServer that allows routing named requests to specific MemorySettingsApiClient
type MemorySettingsApiRouter struct {
	UnimplementedMemorySettingsApiServer

	router.Router
}

// compile time check that we implement the interface we need
var _ MemorySettingsApiServer = (*MemorySettingsApiRouter)(nil)

func NewMemorySettingsApiRouter(opts ...router.Option) *MemorySettingsApiRouter {
	return &MemorySettingsApiRouter{
		Router: router.NewRouter(opts...),
	}
}

// WithMemorySettingsApiClientFactory instructs the router to create a new
// client the first time Get is called for that name.
func WithMemorySettingsApiClientFactory(f func(name string) (MemorySettingsApiClient, error)) router.Option {
	return router.WithFactory(func(name string) (any, error) {
		return f(name)
	})
}

func (r *MemorySettingsApiRouter) Register(server grpc.ServiceRegistrar) {
	RegisterMemorySettingsApiServer(server, r)
}

// Add extends Router.Add to panic if client is not of type MemorySettingsApiClient.
func (r *MemorySettingsApiRouter) Add(name string, client any) any {
	if !r.HoldsType(client) {
		panic(fmt.Sprintf("not correct type: client of type %T is not a MemorySettingsApiClient", client))
	}
	return r.Router.Add(name, client)
}

func (r *MemorySettingsApiRouter) HoldsType(client any) bool {
	_, ok := client.(MemorySettingsApiClient)
	return ok
}

func (r *MemorySettingsApiRouter) AddMemorySettingsApiClient(name string, client MemorySettingsApiClient) MemorySettingsApiClient {
	res := r.Add(name, client)
	if res == nil {
		return nil
	}
	return res.(MemorySettingsApiClient)
}

func (r *MemorySettingsApiRouter) RemoveMemorySettingsApiClient(name string) MemorySettingsApiClient {
	res := r.Remove(name)
	if res == nil {
		return nil
	}
	return res.(MemorySettingsApiClient)
}

func (r *MemorySettingsApiRouter) GetMemorySettingsApiClient(name string) (MemorySettingsApiClient, error) {
	res, err := r.Get(name)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.(MemorySettingsApiClient), nil
}

func (r *MemorySettingsApiRouter) UpdateDemand(ctx context.Context, request *UpdateDemandRequest) (*traits.ElectricDemand, error) {
	child, err := r.GetMemorySettingsApiClient(request.Name)
	if err != nil {
		return nil, err
	}

	return child.UpdateDemand(ctx, request)
}

func (r *MemorySettingsApiRouter) CreateMode(ctx context.Context, request *CreateModeRequest) (*traits.ElectricMode, error) {
	child, err := r.GetMemorySettingsApiClient(request.Name)
	if err != nil {
		return nil, err
	}

	return child.CreateMode(ctx, request)
}

func (r *MemorySettingsApiRouter) UpdateMode(ctx context.Context, request *UpdateModeRequest) (*traits.ElectricMode, error) {
	child, err := r.GetMemorySettingsApiClient(request.Name)
	if err != nil {
		return nil, err
	}

	return child.UpdateMode(ctx, request)
}

func (r *MemorySettingsApiRouter) DeleteMode(ctx context.Context, request *DeleteModeRequest) (*emptypb.Empty, error) {
	child, err := r.GetMemorySettingsApiClient(request.Name)
	if err != nil {
		return nil, err
	}

	return child.DeleteMode(ctx, request)
}
