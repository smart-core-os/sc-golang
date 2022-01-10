// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package booking

import (
	context "context"
	traits "github.com/smart-core-os/sc-api/go/traits"
	wrap "github.com/smart-core-os/sc-golang/pkg/wrap"
	grpc "google.golang.org/grpc"
)

// Wrap Api	adapts a traits.BookingApiServer	and presents it as a traits.BookingApiClient
func WrapApi(server traits.BookingApiServer) traits.BookingApiClient {
	return &apiWrapper{server}
}

type apiWrapper struct {
	server traits.BookingApiServer
}

// compile time check that we implement the interface we need
var _ traits.BookingApiClient = (*apiWrapper)(nil)

func (w *apiWrapper) ListBookings(ctx context.Context, req *traits.ListBookingsRequest, _ ...grpc.CallOption) (*traits.ListBookingsResponse, error) {
	return w.server.ListBookings(ctx, req)
}

func (w *apiWrapper) CheckInBooking(ctx context.Context, req *traits.CheckInBookingRequest, _ ...grpc.CallOption) (*traits.CheckInBookingResponse, error) {
	return w.server.CheckInBooking(ctx, req)
}

func (w *apiWrapper) CheckOutBooking(ctx context.Context, req *traits.CheckOutBookingRequest, _ ...grpc.CallOption) (*traits.CheckOutBookingResponse, error) {
	return w.server.CheckOutBooking(ctx, req)
}

func (w *apiWrapper) CreateBooking(ctx context.Context, req *traits.CreateBookingRequest, _ ...grpc.CallOption) (*traits.CreateBookingResponse, error) {
	return w.server.CreateBooking(ctx, req)
}

func (w *apiWrapper) UpdateBooking(ctx context.Context, req *traits.UpdateBookingRequest, _ ...grpc.CallOption) (*traits.UpdateBookingResponse, error) {
	return w.server.UpdateBooking(ctx, req)
}

func (w *apiWrapper) PullBookings(ctx context.Context, in *traits.ListBookingsRequest, opts ...grpc.CallOption) (traits.BookingApi_PullBookingsClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullBookingsApiServerWrapper{stream.Server()}
	client := &pullBookingsApiClientWrapper{stream.Client()}
	go func() {
		err := w.server.PullBookings(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullBookingsApiClientWrapper struct {
	grpc.ClientStream
}

func (w *pullBookingsApiClientWrapper) Recv() (*traits.PullBookingsResponse, error) {
	m := new(traits.PullBookingsResponse)
	if err := w.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullBookingsApiServerWrapper struct {
	grpc.ServerStream
}

func (s *pullBookingsApiServerWrapper) Send(response *traits.PullBookingsResponse) error {
	return s.ServerStream.SendMsg(response)
}
