package booking

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
	"google.golang.org/grpc"
)

// Wrap adapts a traits.BookingApiServer and presents it as a traits.BookingApiClient
func Wrap(server traits.BookingApiServer) traits.BookingApiClient {
	return &wrapper{server}
}

type wrapper struct {
	server traits.BookingApiServer
}

// compile time check that we implement the interface we need
var _ traits.BookingApiClient = (*wrapper)(nil)

func (b *wrapper) ListBookings(ctx context.Context, in *traits.ListBookingsRequest, opts ...grpc.CallOption) (*traits.ListBookingsResponse, error) {
	return b.server.ListBookings(ctx, in)
}

func (b *wrapper) CheckInBooking(ctx context.Context, in *traits.CheckInBookingRequest, opts ...grpc.CallOption) (*traits.CheckInBookingResponse, error) {
	return b.server.CheckInBooking(ctx, in)
}

func (b *wrapper) CheckOutBooking(ctx context.Context, in *traits.CheckOutBookingRequest, opts ...grpc.CallOption) (*traits.CheckOutBookingResponse, error) {
	return b.server.CheckOutBooking(ctx, in)
}

func (b *wrapper) CreateBooking(ctx context.Context, in *traits.CreateBookingRequest, opts ...grpc.CallOption) (*traits.CreateBookingResponse, error) {
	return b.server.CreateBooking(ctx, in)
}

func (b *wrapper) UpdateBooking(ctx context.Context, in *traits.UpdateBookingRequest, opts ...grpc.CallOption) (*traits.UpdateBookingResponse, error) {
	return b.server.UpdateBooking(ctx, in)
}

func (b *wrapper) PullBookings(ctx context.Context, in *traits.ListBookingsRequest, opts ...grpc.CallOption) (traits.BookingApi_PullBookingsClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullBookingsServerWrapper{stream.Server()}
	client := &pullBookingsClientWrapper{stream.Client()}
	go func() {
		err := b.server.PullBookings(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullBookingsClientWrapper struct {
	grpc.ClientStream
}

func (c *pullBookingsClientWrapper) Recv() (*traits.PullBookingsResponse, error) {
	m := new(traits.PullBookingsResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullBookingsServerWrapper struct {
	grpc.ServerStream
}

func (s *pullBookingsServerWrapper) Send(response *traits.PullBookingsResponse) error {
	return s.ServerStream.SendMsg(response)
}
