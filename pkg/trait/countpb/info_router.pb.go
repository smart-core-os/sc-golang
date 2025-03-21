// Code generated by protoc-gen-router. DO NOT EDIT.

package countpb

import (
	context "context"
	fmt "fmt"
	traits "github.com/smart-core-os/sc-api/go/traits"
	router "github.com/smart-core-os/sc-golang/pkg/router"
	grpc "google.golang.org/grpc"
)

// InfoRouter is a traits.CountInfoServer that allows routing named requests to specific traits.CountInfoClient
type InfoRouter struct {
	traits.UnimplementedCountInfoServer

	router.Router
}

// compile time check that we implement the interface we need
var _ traits.CountInfoServer = (*InfoRouter)(nil)

func NewInfoRouter(opts ...router.Option) *InfoRouter {
	return &InfoRouter{
		Router: router.NewRouter(opts...),
	}
}

// WithCountInfoClientFactory instructs the router to create a new
// client the first time Get is called for that name.
func WithCountInfoClientFactory(f func(name string) (traits.CountInfoClient, error)) router.Option {
	return router.WithFactory(func(name string) (any, error) {
		return f(name)
	})
}

func (r *InfoRouter) Register(server grpc.ServiceRegistrar) {
	traits.RegisterCountInfoServer(server, r)
}

// Add extends Router.Add to panic if client is not of type traits.CountInfoClient.
func (r *InfoRouter) Add(name string, client any) any {
	if !r.HoldsType(client) {
		panic(fmt.Sprintf("not correct type: client of type %T is not a traits.CountInfoClient", client))
	}
	return r.Router.Add(name, client)
}

func (r *InfoRouter) HoldsType(client any) bool {
	_, ok := client.(traits.CountInfoClient)
	return ok
}

func (r *InfoRouter) AddCountInfoClient(name string, client traits.CountInfoClient) traits.CountInfoClient {
	res := r.Add(name, client)
	if res == nil {
		return nil
	}
	return res.(traits.CountInfoClient)
}

func (r *InfoRouter) RemoveCountInfoClient(name string) traits.CountInfoClient {
	res := r.Remove(name)
	if res == nil {
		return nil
	}
	return res.(traits.CountInfoClient)
}

func (r *InfoRouter) GetCountInfoClient(name string) (traits.CountInfoClient, error) {
	res, err := r.Get(name)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.(traits.CountInfoClient), nil
}

func (r *InfoRouter) DescribeCount(ctx context.Context, request *traits.DescribeCountRequest) (*traits.CountSupport, error) {
	child, err := r.GetCountInfoClient(request.Name)
	if err != nil {
		return nil, err
	}

	return child.DescribeCount(ctx, request)
}
