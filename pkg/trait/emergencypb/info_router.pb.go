// Code generated by protoc-gen-router. DO NOT EDIT.

package emergencypb

import (
	context "context"
	fmt "fmt"
	traits "github.com/smart-core-os/sc-api/go/traits"
	router "github.com/smart-core-os/sc-golang/pkg/router"
	grpc "google.golang.org/grpc"
)

// InfoRouter is a traits.EmergencyInfoServer that allows routing named requests to specific traits.EmergencyInfoClient
type InfoRouter struct {
	traits.UnimplementedEmergencyInfoServer

	router.Router
}

// compile time check that we implement the interface we need
var _ traits.EmergencyInfoServer = (*InfoRouter)(nil)

func NewInfoRouter(opts ...router.Option) *InfoRouter {
	return &InfoRouter{
		Router: router.NewRouter(opts...),
	}
}

// WithEmergencyInfoClientFactory instructs the router to create a new
// client the first time Get is called for that name.
func WithEmergencyInfoClientFactory(f func(name string) (traits.EmergencyInfoClient, error)) router.Option {
	return router.WithFactory(func(name string) (any, error) {
		return f(name)
	})
}

func (r *InfoRouter) Register(server grpc.ServiceRegistrar) {
	traits.RegisterEmergencyInfoServer(server, r)
}

// Add extends Router.Add to panic if client is not of type traits.EmergencyInfoClient.
func (r *InfoRouter) Add(name string, client any) any {
	if !r.HoldsType(client) {
		panic(fmt.Sprintf("not correct type: client of type %T is not a traits.EmergencyInfoClient", client))
	}
	return r.Router.Add(name, client)
}

func (r *InfoRouter) HoldsType(client any) bool {
	_, ok := client.(traits.EmergencyInfoClient)
	return ok
}

func (r *InfoRouter) AddEmergencyInfoClient(name string, client traits.EmergencyInfoClient) traits.EmergencyInfoClient {
	res := r.Add(name, client)
	if res == nil {
		return nil
	}
	return res.(traits.EmergencyInfoClient)
}

func (r *InfoRouter) RemoveEmergencyInfoClient(name string) traits.EmergencyInfoClient {
	res := r.Remove(name)
	if res == nil {
		return nil
	}
	return res.(traits.EmergencyInfoClient)
}

func (r *InfoRouter) GetEmergencyInfoClient(name string) (traits.EmergencyInfoClient, error) {
	res, err := r.Get(name)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.(traits.EmergencyInfoClient), nil
}

func (r *InfoRouter) DescribeEmergency(ctx context.Context, request *traits.DescribeEmergencyRequest) (*traits.EmergencySupport, error) {
	child, err := r.GetEmergencyInfoClient(request.Name)
	if err != nil {
		return nil, err
	}

	return child.DescribeEmergency(ctx, request)
}
