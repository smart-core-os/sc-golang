// Code generated by protoc-gen-router. DO NOT EDIT.

package powersupply

import (
	context "context"
	fmt "fmt"
	traits "github.com/smart-core-os/sc-api/go/traits"
	router "github.com/smart-core-os/sc-golang/pkg/router"
	grpc "google.golang.org/grpc"
)

// InfoRouter is a traits.PowerSupplyInfoServer that allows routing named requests to specific traits.PowerSupplyInfoClient
type InfoRouter struct {
	traits.UnimplementedPowerSupplyInfoServer

	router.Router
}

// compile time check that we implement the interface we need
var _ traits.PowerSupplyInfoServer = (*InfoRouter)(nil)

func NewInfoRouter(opts ...router.Option) *InfoRouter {
	return &InfoRouter{
		Router: router.NewRouter(opts...),
	}
}

// WithPowerSupplyInfoClientFactory instructs the router to create a new
// client the first time Get is called for that name.
func WithPowerSupplyInfoClientFactory(f func(name string) (traits.PowerSupplyInfoClient, error)) router.Option {
	return router.WithFactory(func(name string) (interface{}, error) {
		return f(name)
	})
}

func (r *InfoRouter) Register(server *grpc.Server) {
	traits.RegisterPowerSupplyInfoServer(server, r)
}

// Add extends Router.Add to panic if client is not of type traits.PowerSupplyInfoClient.
func (r *InfoRouter) Add(name string, client interface{}) interface{} {
	if !r.HoldsType(client) {
		panic(fmt.Sprintf("not correct type: client of type %T is not a traits.PowerSupplyInfoClient", client))
	}
	return r.Router.Add(name, client)
}

func (r *InfoRouter) HoldsType(client interface{}) bool {
	_, ok := client.(traits.PowerSupplyInfoClient)
	return ok
}

func (r *InfoRouter) AddPowerSupplyInfoClient(name string, client traits.PowerSupplyInfoClient) traits.PowerSupplyInfoClient {
	res := r.Add(name, client)
	if res == nil {
		return nil
	}
	return res.(traits.PowerSupplyInfoClient)
}

func (r *InfoRouter) RemovePowerSupplyInfoClient(name string) traits.PowerSupplyInfoClient {
	res := r.Remove(name)
	if res == nil {
		return nil
	}
	return res.(traits.PowerSupplyInfoClient)
}

func (r *InfoRouter) GetPowerSupplyInfoClient(name string) (traits.PowerSupplyInfoClient, error) {
	res, err := r.Get(name)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.(traits.PowerSupplyInfoClient), nil
}

func (r *InfoRouter) DescribePowerCapacity(ctx context.Context, request *traits.DescribePowerCapacityRequest) (*traits.PowerCapacitySupport, error) {
	child, err := r.GetPowerSupplyInfoClient(request.Name)
	if err != nil {
		return nil, err
	}

	return child.DescribePowerCapacity(ctx, request)
}

func (r *InfoRouter) DescribeDrawNotification(ctx context.Context, request *traits.DescribeDrawNotificationRequest) (*traits.DrawNotificationSupport, error) {
	child, err := r.GetPowerSupplyInfoClient(request.Name)
	if err != nil {
		return nil, err
	}

	return child.DescribeDrawNotification(ctx, request)
}