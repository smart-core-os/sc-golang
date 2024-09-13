// Code generated by protoc-gen-router. DO NOT EDIT.

package booking

import (
	context "context"
	fmt "fmt"
	traits "github.com/smart-core-os/sc-api/go/traits"
	router "github.com/smart-core-os/sc-golang/pkg/router"
	grpc "google.golang.org/grpc"
)

// InfoRouter is a traits.BookingInfoServer that allows routing named requests to specific traits.BookingInfoClient
type InfoRouter struct {
	traits.UnimplementedBookingInfoServer

	router.Router
}

// compile time check that we implement the interface we need
var _ traits.BookingInfoServer = (*InfoRouter)(nil)

func NewInfoRouter(opts ...router.Option) *InfoRouter {
	return &InfoRouter{
		Router: router.NewRouter(opts...),
	}
}

// WithBookingInfoClientFactory instructs the router to create a new
// client the first time Get is called for that name.
func WithBookingInfoClientFactory(f func(name string) (traits.BookingInfoClient, error)) router.Option {
	return router.WithFactory(func(name string) (any, error) {
		return f(name)
	})
}

func (r *InfoRouter) Register(server grpc.ServiceRegistrar) {
	traits.RegisterBookingInfoServer(server, r)
}

// Add extends Router.Add to panic if client is not of type traits.BookingInfoClient.
func (r *InfoRouter) Add(name string, client any) any {
	if !r.HoldsType(client) {
		panic(fmt.Sprintf("not correct type: client of type %T is not a traits.BookingInfoClient", client))
	}
	return r.Router.Add(name, client)
}

func (r *InfoRouter) HoldsType(client any) bool {
	_, ok := client.(traits.BookingInfoClient)
	return ok
}

func (r *InfoRouter) AddBookingInfoClient(name string, client traits.BookingInfoClient) traits.BookingInfoClient {
	res := r.Add(name, client)
	if res == nil {
		return nil
	}
	return res.(traits.BookingInfoClient)
}

func (r *InfoRouter) RemoveBookingInfoClient(name string) traits.BookingInfoClient {
	res := r.Remove(name)
	if res == nil {
		return nil
	}
	return res.(traits.BookingInfoClient)
}

func (r *InfoRouter) GetBookingInfoClient(name string) (traits.BookingInfoClient, error) {
	res, err := r.Get(name)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.(traits.BookingInfoClient), nil
}

func (r *InfoRouter) DescribeBooking(ctx context.Context, request *traits.DescribeBookingRequest) (*traits.BookingSupport, error) {
	child, err := r.GetBookingInfoClient(request.Name)
	if err != nil {
		return nil, err
	}

	return child.DescribeBooking(ctx, request)
}
