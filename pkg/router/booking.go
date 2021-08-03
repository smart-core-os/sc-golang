package router

import (
	"context"
	"io"
	"sync"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BookingApiRouter is a BookingApiServer that allows routing named requests to specific BookingApiClients
type BookingApiRouter struct {
	traits.UnimplementedBookingApiServer

	mu       sync.Mutex
	registry map[string]traits.BookingApiClient

	// Factory can be used to dynamically create api clients if requests come in for devices we haven't seen.
	Factory func(string) (traits.BookingApiClient, error)
}

// compile time check that we implement the interface we need
var _ traits.BookingApiServer = (*BookingApiRouter)(nil)

func NewBookingApiRouter() *BookingApiRouter {
	return &BookingApiRouter{
		registry: make(map[string]traits.BookingApiClient),
	}
}

func (r *BookingApiRouter) Register(server *grpc.Server) {
	traits.RegisterBookingApiServer(server, r)
}

func (r *BookingApiRouter) Add(name string, client traits.BookingApiClient) traits.BookingApiClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	r.registry[name] = client
	return old
}

func (r *BookingApiRouter) Remove(name string) traits.BookingApiClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	delete(r.registry, name)
	return old
}

func (r *BookingApiRouter) Has(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, exists := r.registry[name]
	return exists
}

func (r *BookingApiRouter) Get(name string) (traits.BookingApiClient, error) {
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

func (r *BookingApiRouter) ListBookings(ctx context.Context, request *traits.ListBookingsRequest) (*traits.ListBookingsResponse, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}
	return child.ListBookings(ctx, request)
}

func (r *BookingApiRouter) CheckInBooking(ctx context.Context, request *traits.CheckInBookingRequest) (*traits.CheckInBookingResponse, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}
	return child.CheckInBooking(ctx, request)
}

func (r *BookingApiRouter) CheckOutBooking(ctx context.Context, request *traits.CheckOutBookingRequest) (*traits.CheckOutBookingResponse, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}
	return child.CheckOutBooking(ctx, request)
}

func (r *BookingApiRouter) CreateBooking(ctx context.Context, request *traits.CreateBookingRequest) (*traits.CreateBookingResponse, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}
	return child.CreateBooking(ctx, request)
}

func (r *BookingApiRouter) UpdateBooking(ctx context.Context, request *traits.UpdateBookingRequest) (*traits.UpdateBookingResponse, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}
	return child.UpdateBooking(ctx, request)
}

func (r *BookingApiRouter) PullBookings(request *traits.ListBookingsRequest, server traits.BookingApi_PullBookingsServer) error {
	child, err := r.Get(request.Name)
	if err != nil {
		return err
	}

	// so we can cancel our forwarding request if we can't send responses to our caller
	reqCtx, reqDone := context.WithCancel(server.Context())
	// issue the request
	stream, err := child.PullBookings(reqCtx, request)
	if err != nil {
		return err
	}

	// send the stream header
	header, err := stream.Header()
	if err != nil {
		return err
	}
	if err = server.SendHeader(header); err != nil {
		return err
	}

	// send all the messages
	// false means the error is from the child, true means the error is from the caller
	var callerError bool
	for {
		// Impl note: we could improve throughput here by issuing the Recv and Send in different goroutines, but we're doing
		// it synchronously until we have a need to change the behaviour

		var msg *traits.PullBookingsResponse
		msg, err = stream.Recv()
		if err != nil {
			break
		}

		err = server.Send(msg)
		if err != nil {
			callerError = true
			break
		}
	}

	// err is guaranteed to be non-nil as it's the only way to exit the loop
	if callerError {
		// cancel the request
		reqDone()
		return err
	} else {
		if trailer := stream.Trailer(); trailer != nil {
			server.SetTrailer(trailer)
		}
		if err == io.EOF {
			return nil
		}
		return err
	}
}
