// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package channel

import (
	context "context"
	traits "github.com/smart-core-os/sc-api/go/traits"
	wrap "github.com/smart-core-os/sc-golang/pkg/wrap"
	grpc "google.golang.org/grpc"
)

// WrapApi	adapts a traits.ChannelApiServer	and presents it as a traits.ChannelApiClient
func WrapApi(server traits.ChannelApiServer) traits.ChannelApiClient {
	return &apiWrapper{server}
}

type apiWrapper struct {
	server traits.ChannelApiServer
}

// compile time check that we implement the interface we need
var _ traits.ChannelApiClient = (*apiWrapper)(nil)

// UnwrapServer returns the underlying server instance.
func (w *apiWrapper) UnwrapServer() traits.ChannelApiServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *apiWrapper) Unwrap() interface{} {
	return w.UnwrapServer()
}

func (w *apiWrapper) GetChosenChannel(ctx context.Context, req *traits.GetChosenChannelRequest, _ ...grpc.CallOption) (*traits.Channel, error) {
	return w.server.GetChosenChannel(ctx, req)
}

func (w *apiWrapper) ChooseChannel(ctx context.Context, req *traits.ChooseChannelRequest, _ ...grpc.CallOption) (*traits.Channel, error) {
	return w.server.ChooseChannel(ctx, req)
}

func (w *apiWrapper) AdjustChannel(ctx context.Context, req *traits.AdjustChannelRequest, _ ...grpc.CallOption) (*traits.Channel, error) {
	return w.server.AdjustChannel(ctx, req)
}

func (w *apiWrapper) ReturnChannel(ctx context.Context, req *traits.ReturnChannelRequest, _ ...grpc.CallOption) (*traits.Channel, error) {
	return w.server.ReturnChannel(ctx, req)
}

func (w *apiWrapper) PullChosenChannel(ctx context.Context, in *traits.PullChosenChannelRequest, opts ...grpc.CallOption) (traits.ChannelApi_PullChosenChannelClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullChosenChannelApiServerWrapper{stream.Server()}
	client := &pullChosenChannelApiClientWrapper{stream.Client()}
	go func() {
		err := w.server.PullChosenChannel(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullChosenChannelApiClientWrapper struct {
	grpc.ClientStream
}

func (w *pullChosenChannelApiClientWrapper) Recv() (*traits.PullChosenChannelResponse, error) {
	m := new(traits.PullChosenChannelResponse)
	if err := w.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullChosenChannelApiServerWrapper struct {
	grpc.ServerStream
}

func (s *pullChosenChannelApiServerWrapper) Send(response *traits.PullChosenChannelResponse) error {
	return s.ServerStream.SendMsg(response)
}
