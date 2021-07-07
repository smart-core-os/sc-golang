package wrap

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc"
)

// BookingApiServer adapts a traits.BookingApiServer and presents it as a traits.BookingApiClient
func BookingApiServer(server traits.BookingApiServer) traits.BookingApiClient {
	return &bookingApiServerClient{server}
}

type bookingApiServerClient struct {
	server traits.BookingApiServer
}

// compile time check that we implement the interface we need
var _ traits.BookingApiClient = &bookingApiServerClient{}

func (b *bookingApiServerClient) ListBookings(ctx context.Context, in *traits.ListBookingsRequest, opts ...grpc.CallOption) (*traits.ListBookingsResponse, error) {
	return b.server.ListBookings(ctx, in)
}

func (b *bookingApiServerClient) CheckInBooking(ctx context.Context, in *traits.CheckInBookingRequest, opts ...grpc.CallOption) (*traits.CheckInBookingResponse, error) {
	return b.server.CheckInBooking(ctx, in)
}

func (b *bookingApiServerClient) CheckOutBooking(ctx context.Context, in *traits.CheckOutBookingRequest, opts ...grpc.CallOption) (*traits.CheckOutBookingResponse, error) {
	return b.server.CheckOutBooking(ctx, in)
}

func (b *bookingApiServerClient) CreateBooking(ctx context.Context, in *traits.CreateBookingRequest, opts ...grpc.CallOption) (*traits.CreateBookingResponse, error) {
	return b.server.CreateBooking(ctx, in)
}

func (b *bookingApiServerClient) UpdateBooking(ctx context.Context, in *traits.UpdateBookingRequest, opts ...grpc.CallOption) (*traits.UpdateBookingResponse, error) {
	return b.server.UpdateBooking(ctx, in)
}

func (b *bookingApiServerClient) PullBookings(ctx context.Context, in *traits.ListBookingsRequest, opts ...grpc.CallOption) (traits.BookingApi_PullBookingsClient, error) {
	stream := newClientServerStream(ctx)
	server := &bookingApiPullBookingsServer{stream.Server()}
	client := &bookingApiPullBookingsClient{stream.Client()}
	go func() {
		err := b.server.PullBookings(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type bookingApiPullBookingsClient struct {
	grpc.ClientStream
}

func (c *bookingApiPullBookingsClient) Recv() (*traits.PullBookingsResponse, error) {
	m := new(traits.PullBookingsResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type bookingApiPullBookingsServer struct {
	grpc.ServerStream
}

func (s *bookingApiPullBookingsServer) Send(response *traits.PullBookingsResponse) error {
	return s.ServerStream.SendMsg(response)
}
