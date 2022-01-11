// Code generated by protoc-gen-router. DO NOT EDIT.

package booking

import (
	context "context"
	traits "github.com/smart-core-os/sc-api/go/traits"
	router "github.com/smart-core-os/sc-golang/pkg/router"
	grpc "google.golang.org/grpc"
)

// InfoRouter is a traits.BookingInfoServer that allows routing named requests to specific traits.BookingInfoClient
type InfoRouter struct {
	traits.UnimplementedBookingInfoServer

	router *router.Router
}

// compile time check that we implement the interface we need
var _ traits.BookingInfoServer = (*InfoRouter)(nil)

func NewInfoRouter(opts ...router.Option) *InfoRouter {
	return &InfoRouter{
		router: router.NewRouter(opts...),
	}
}

// WithBookingInfoClientFactory instructs the router to create a new
// client the first time Get is called for that name.
func WithBookingInfoClientFactory(f func(name string) (traits.BookingInfoClient, error)) router.Option {
	return router.WithFactory(func(name string) (interface{}, error) {
		return f(name)
	})
}

func (r *InfoRouter) Register(server *grpc.Server) {
	traits.RegisterBookingInfoServer(server, r)
}

func (r *InfoRouter) Add(name string, client traits.BookingInfoClient) traits.BookingInfoClient {
	res := r.router.Add(name, client)
	if res == nil {
		return nil
	}
	return res.(traits.BookingInfoClient)
}

func (r *InfoRouter) Remove(name string) traits.BookingInfoClient {
	res := r.router.Remove(name)
	if res == nil {
		return nil
	}
	return res.(traits.BookingInfoClient)
}

func (r *InfoRouter) Has(name string) bool {
	return r.router.Has(name)
}

func (r *InfoRouter) Get(name string) (traits.BookingInfoClient, error) {
	res, err := r.router.Get(name)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.(traits.BookingInfoClient), nil
}

func (r *InfoRouter) DescribeBooking(ctx context.Context, request *traits.DescribeBookingRequest) (*traits.BookingSupport, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.DescribeBooking(ctx, request)
}
