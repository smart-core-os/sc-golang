// Code generated by protoc-gen-router. DO NOT EDIT.

package openclose

import (
	context "context"
	fmt "fmt"
	traits "github.com/smart-core-os/sc-api/go/traits"
	router "github.com/smart-core-os/sc-golang/pkg/router"
	grpc "google.golang.org/grpc"
)

// InfoRouter is a traits.OpenCloseInfoServer that allows routing named requests to specific traits.OpenCloseInfoClient
type InfoRouter struct {
	traits.UnimplementedOpenCloseInfoServer

	router.Router
}

// compile time check that we implement the interface we need
var _ traits.OpenCloseInfoServer = (*InfoRouter)(nil)

func NewInfoRouter(opts ...router.Option) *InfoRouter {
	return &InfoRouter{
		Router: router.NewRouter(opts...),
	}
}

// WithOpenCloseInfoClientFactory instructs the router to create a new
// client the first time Get is called for that name.
func WithOpenCloseInfoClientFactory(f func(name string) (traits.OpenCloseInfoClient, error)) router.Option {
	return router.WithFactory(func(name string) (interface{}, error) {
		return f(name)
	})
}

func (r *InfoRouter) Register(server *grpc.Server) {
	traits.RegisterOpenCloseInfoServer(server, r)
}

// Add extends Router.Add to panic if client is not of type traits.OpenCloseInfoClient.
func (r *InfoRouter) Add(name string, client interface{}) interface{} {
	if !r.HoldsType(client) {
		panic(fmt.Sprintf("not correct type: client of type %T is not a traits.OpenCloseInfoClient", client))
	}
	return r.Router.Add(name, client)
}

func (r *InfoRouter) HoldsType(client interface{}) bool {
	_, ok := client.(traits.OpenCloseInfoClient)
	return ok
}

func (r *InfoRouter) AddOpenCloseInfoClient(name string, client traits.OpenCloseInfoClient) traits.OpenCloseInfoClient {
	res := r.Add(name, client)
	if res == nil {
		return nil
	}
	return res.(traits.OpenCloseInfoClient)
}

func (r *InfoRouter) RemoveOpenCloseInfoClient(name string) traits.OpenCloseInfoClient {
	res := r.Remove(name)
	if res == nil {
		return nil
	}
	return res.(traits.OpenCloseInfoClient)
}

func (r *InfoRouter) GetOpenCloseInfoClient(name string) (traits.OpenCloseInfoClient, error) {
	res, err := r.Get(name)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.(traits.OpenCloseInfoClient), nil
}

func (r *InfoRouter) DescribePositions(ctx context.Context, request *traits.DescribePositionsRequest) (*traits.PositionsSupport, error) {
	child, err := r.GetOpenCloseInfoClient(request.Name)
	if err != nil {
		return nil, err
	}

	return child.DescribePositions(ctx, request)
}
