// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package occupancysensor

import (
	context "context"
	traits "github.com/smart-core-os/sc-api/go/traits"
	wrap "github.com/smart-core-os/sc-golang/pkg/wrap"
	grpc "google.golang.org/grpc"
)

// WrapApi	adapts a traits.OccupancySensorApiServer	and presents it as a traits.OccupancySensorApiClient
func WrapApi(server traits.OccupancySensorApiServer) traits.OccupancySensorApiClient {
	return &apiWrapper{server}
}

type apiWrapper struct {
	server traits.OccupancySensorApiServer
}

// compile time check that we implement the interface we need
var _ traits.OccupancySensorApiClient = (*apiWrapper)(nil)

// UnwrapServer returns the underlying server instance.
func (w *apiWrapper) UnwrapServer() traits.OccupancySensorApiServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *apiWrapper) Unwrap() interface{} {
	return w.UnwrapServer()
}

func (w *apiWrapper) GetOccupancy(ctx context.Context, req *traits.GetOccupancyRequest, _ ...grpc.CallOption) (*traits.Occupancy, error) {
	return w.server.GetOccupancy(ctx, req)
}

func (w *apiWrapper) PullOccupancy(ctx context.Context, in *traits.PullOccupancyRequest, opts ...grpc.CallOption) (traits.OccupancySensorApi_PullOccupancyClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullOccupancyApiServerWrapper{stream.Server()}
	client := &pullOccupancyApiClientWrapper{stream.Client()}
	go func() {
		err := w.server.PullOccupancy(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullOccupancyApiClientWrapper struct {
	grpc.ClientStream
}

func (w *pullOccupancyApiClientWrapper) Recv() (*traits.PullOccupancyResponse, error) {
	m := new(traits.PullOccupancyResponse)
	if err := w.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullOccupancyApiServerWrapper struct {
	grpc.ServerStream
}

func (s *pullOccupancyApiServerWrapper) Send(response *traits.PullOccupancyResponse) error {
	return s.ServerStream.SendMsg(response)
}
