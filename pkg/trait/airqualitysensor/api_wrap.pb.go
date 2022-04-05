// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package airqualitysensor

import (
	context "context"
	traits "github.com/smart-core-os/sc-api/go/traits"
	wrap "github.com/smart-core-os/sc-golang/pkg/wrap"
	grpc "google.golang.org/grpc"
)

// WrapApi	adapts a traits.AirQualitySensorApiServer	and presents it as a traits.AirQualitySensorApiClient
func WrapApi(server traits.AirQualitySensorApiServer) traits.AirQualitySensorApiClient {
	return &apiWrapper{server}
}

type apiWrapper struct {
	server traits.AirQualitySensorApiServer
}

// compile time check that we implement the interface we need
var _ traits.AirQualitySensorApiClient = (*apiWrapper)(nil)

// UnwrapServer returns the underlying server instance.
func (w *apiWrapper) UnwrapServer() traits.AirQualitySensorApiServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *apiWrapper) Unwrap() interface{} {
	return w.UnwrapServer()
}

func (w *apiWrapper) GetAirQuality(ctx context.Context, req *traits.GetAirQualityRequest, _ ...grpc.CallOption) (*traits.AirQuality, error) {
	return w.server.GetAirQuality(ctx, req)
}

func (w *apiWrapper) PullAirQuality(ctx context.Context, in *traits.PullAirQualityRequest, opts ...grpc.CallOption) (traits.AirQualitySensorApi_PullAirQualityClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullAirQualityApiServerWrapper{stream.Server()}
	client := &pullAirQualityApiClientWrapper{stream.Client()}
	go func() {
		err := w.server.PullAirQuality(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullAirQualityApiClientWrapper struct {
	grpc.ClientStream
}

func (w *pullAirQualityApiClientWrapper) Recv() (*traits.PullAirQualityResponse, error) {
	m := new(traits.PullAirQualityResponse)
	if err := w.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullAirQualityApiServerWrapper struct {
	grpc.ServerStream
}

func (s *pullAirQualityApiServerWrapper) Send(response *traits.PullAirQualityResponse) error {
	return s.ServerStream.SendMsg(response)
}
