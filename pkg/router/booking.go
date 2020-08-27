package router

import (
	"context"
	"fmt"
	"io"
	"sync"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BookingApiRouter is a BookingApiServer that allows routing named requests to specific BookingApiClients
type BookingApiRouter struct {
	traits.UnimplementedBookingApiServer

	mu       sync.Mutex
	registry map[string]traits.BookingApiClient
}

// compile time check that we implement the interface we need
var _ traits.BookingApiServer = &BookingApiRouter{}

func (b *BookingApiRouter) Add(name string, client traits.BookingApiClient) traits.BookingApiClient {
	b.mu.Lock()
	defer b.mu.Unlock()
	old := b.registry[name]
	b.registry[name] = client
	return old
}

func (b *BookingApiRouter) Remove(name string) traits.BookingApiClient {
	b.mu.Lock()
	defer b.mu.Unlock()
	old := b.registry[name]
	delete(b.registry, name)
	return old
}

func (b *BookingApiRouter) Has(name string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, exists := b.registry[name]
	return exists
}

func (b *BookingApiRouter) ListBookings(ctx context.Context, request *traits.ListBookingsRequest) (*traits.ListBookingsResponse, error) {
	b.mu.Lock()
	child, exists := b.registry[request.Name]
	b.mu.Unlock()
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("device: %v", request.Name))
	}
	return child.ListBookings(ctx, request)
}

func (b *BookingApiRouter) CheckInBooking(ctx context.Context, request *traits.CheckInBookingRequest) (*traits.CheckInBookingResponse, error) {
	b.mu.Lock()
	child, exists := b.registry[request.Name]
	b.mu.Unlock()
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("device: %v", request.Name))
	}
	return child.CheckInBooking(ctx, request)
}

func (b *BookingApiRouter) CheckOutBooking(ctx context.Context, request *traits.CheckOutBookingRequest) (*traits.CheckOutBookingResponse, error) {
	b.mu.Lock()
	child, exists := b.registry[request.Name]
	b.mu.Unlock()
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("device: %v", request.Name))
	}
	return child.CheckOutBooking(ctx, request)
}

func (b *BookingApiRouter) CreateBooking(ctx context.Context, request *traits.CreateBookingRequest) (*traits.CreateBookingResponse, error) {
	b.mu.Lock()
	child, exists := b.registry[request.Name]
	b.mu.Unlock()
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("device: %v", request.Name))
	}
	return child.CreateBooking(ctx, request)
}

func (b *BookingApiRouter) UpdateBooking(ctx context.Context, request *traits.UpdateBookingRequest) (*traits.UpdateBookingResponse, error) {
	b.mu.Lock()
	child, exists := b.registry[request.Name]
	b.mu.Unlock()
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("device: %v", request.Name))
	}
	return child.UpdateBooking(ctx, request)
}

func (b *BookingApiRouter) PullBookings(request *traits.ListBookingsRequest, server traits.BookingApi_PullBookingsServer) error {
	b.mu.Lock()
	child, exists := b.registry[request.Name]
	b.mu.Unlock()
	if !exists {
		return status.Error(codes.NotFound, fmt.Sprintf("device: %v", request.Name))
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
