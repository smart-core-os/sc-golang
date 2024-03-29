// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package onoff

import (
	context "context"
	traits "github.com/smart-core-os/sc-api/go/traits"
	wrap "github.com/smart-core-os/sc-golang/pkg/wrap"
	grpc "google.golang.org/grpc"
)

// WrapApi	adapts a traits.OnOffApiServer	and presents it as a traits.OnOffApiClient
func WrapApi(server traits.OnOffApiServer) traits.OnOffApiClient {
	return &apiWrapper{server}
}

type apiWrapper struct {
	server traits.OnOffApiServer
}

// compile time check that we implement the interface we need
var _ traits.OnOffApiClient = (*apiWrapper)(nil)

// UnwrapServer returns the underlying server instance.
func (w *apiWrapper) UnwrapServer() traits.OnOffApiServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *apiWrapper) Unwrap() any {
	return w.UnwrapServer()
}

func (w *apiWrapper) GetOnOff(ctx context.Context, req *traits.GetOnOffRequest, _ ...grpc.CallOption) (*traits.OnOff, error) {
	return w.server.GetOnOff(ctx, req)
}

func (w *apiWrapper) UpdateOnOff(ctx context.Context, req *traits.UpdateOnOffRequest, _ ...grpc.CallOption) (*traits.OnOff, error) {
	return w.server.UpdateOnOff(ctx, req)
}

func (w *apiWrapper) PullOnOff(ctx context.Context, in *traits.PullOnOffRequest, opts ...grpc.CallOption) (traits.OnOffApi_PullOnOffClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullOnOffApiServerWrapper{stream.Server()}
	client := &pullOnOffApiClientWrapper{stream.Client()}
	go func() {
		err := w.server.PullOnOff(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullOnOffApiClientWrapper struct {
	grpc.ClientStream
}

func (w *pullOnOffApiClientWrapper) Recv() (*traits.PullOnOffResponse, error) {
	m := new(traits.PullOnOffResponse)
	if err := w.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullOnOffApiServerWrapper struct {
	grpc.ServerStream
}

func (s *pullOnOffApiServerWrapper) Send(response *traits.PullOnOffResponse) error {
	return s.ServerStream.SendMsg(response)
}
