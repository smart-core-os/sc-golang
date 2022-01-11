// Code generated by protoc-gen-router. DO NOT EDIT.

package emergency

import (
	context "context"
	traits "github.com/smart-core-os/sc-api/go/traits"
	router "github.com/smart-core-os/sc-golang/pkg/router"
	grpc "google.golang.org/grpc"
)

// InfoRouter is a traits.EmergencyInfoServer that allows routing named requests to specific traits.EmergencyInfoClient
type InfoRouter struct {
	traits.UnimplementedEmergencyInfoServer

	router *router.Router
}

// compile time check that we implement the interface we need
var _ traits.EmergencyInfoServer = (*InfoRouter)(nil)

func NewInfoRouter(opts ...router.Option) *InfoRouter {
	return &InfoRouter{
		router: router.NewRouter(opts...),
	}
}

// WithEmergencyInfoClientFactory instructs the router to create a new
// client the first time Get is called for that name.
func WithEmergencyInfoClientFactory(f func(name string) (traits.EmergencyInfoClient, error)) router.Option {
	return router.WithFactory(func(name string) (interface{}, error) {
		return f(name)
	})
}

func (r *InfoRouter) Register(server *grpc.Server) {
	traits.RegisterEmergencyInfoServer(server, r)
}

func (r *InfoRouter) Add(name string, client traits.EmergencyInfoClient) traits.EmergencyInfoClient {
	res := r.router.Add(name, client)
	if res == nil {
		return nil
	}
	return res.(traits.EmergencyInfoClient)
}

func (r *InfoRouter) Remove(name string) traits.EmergencyInfoClient {
	res := r.router.Remove(name)
	if res == nil {
		return nil
	}
	return res.(traits.EmergencyInfoClient)
}

func (r *InfoRouter) Has(name string) bool {
	return r.router.Has(name)
}

func (r *InfoRouter) Get(name string) (traits.EmergencyInfoClient, error) {
	res, err := r.router.Get(name)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.(traits.EmergencyInfoClient), nil
}

func (r *InfoRouter) DescribeEmergency(ctx context.Context, request *traits.DescribeEmergencyRequest) (*traits.EmergencySupport, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.DescribeEmergency(ctx, request)
}
