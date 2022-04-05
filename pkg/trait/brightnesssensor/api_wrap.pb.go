// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package brightnesssensor

import (
	context "context"
	traits "github.com/smart-core-os/sc-api/go/traits"
	wrap "github.com/smart-core-os/sc-golang/pkg/wrap"
	grpc "google.golang.org/grpc"
)

// WrapApi	adapts a traits.BrightnessSensorApiServer	and presents it as a traits.BrightnessSensorApiClient
func WrapApi(server traits.BrightnessSensorApiServer) traits.BrightnessSensorApiClient {
	return &apiWrapper{server}
}

type apiWrapper struct {
	server traits.BrightnessSensorApiServer
}

// compile time check that we implement the interface we need
var _ traits.BrightnessSensorApiClient = (*apiWrapper)(nil)

// UnwrapServer returns the underlying server instance.
func (w *apiWrapper) UnwrapServer() traits.BrightnessSensorApiServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *apiWrapper) Unwrap() interface{} {
	return w.UnwrapServer()
}

func (w *apiWrapper) GetAmbientBrightness(ctx context.Context, req *traits.GetAmbientBrightnessRequest, _ ...grpc.CallOption) (*traits.AmbientBrightness, error) {
	return w.server.GetAmbientBrightness(ctx, req)
}

func (w *apiWrapper) PullAmbientBrightness(ctx context.Context, in *traits.PullAmbientBrightnessRequest, opts ...grpc.CallOption) (traits.BrightnessSensorApi_PullAmbientBrightnessClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullAmbientBrightnessApiServerWrapper{stream.Server()}
	client := &pullAmbientBrightnessApiClientWrapper{stream.Client()}
	go func() {
		err := w.server.PullAmbientBrightness(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullAmbientBrightnessApiClientWrapper struct {
	grpc.ClientStream
}

func (w *pullAmbientBrightnessApiClientWrapper) Recv() (*traits.PullAmbientBrightnessResponse, error) {
	m := new(traits.PullAmbientBrightnessResponse)
	if err := w.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullAmbientBrightnessApiServerWrapper struct {
	grpc.ServerStream
}

func (s *pullAmbientBrightnessApiServerWrapper) Send(response *traits.PullAmbientBrightnessResponse) error {
	return s.ServerStream.SendMsg(response)
}
