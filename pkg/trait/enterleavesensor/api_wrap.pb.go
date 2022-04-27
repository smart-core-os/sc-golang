// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package enterleavesensor

import (
	context "context"
	traits "github.com/smart-core-os/sc-api/go/traits"
	wrap "github.com/smart-core-os/sc-golang/pkg/wrap"
	grpc "google.golang.org/grpc"
)

// WrapApi	adapts a traits.EnterLeaveSensorApiServer	and presents it as a traits.EnterLeaveSensorApiClient
func WrapApi(server traits.EnterLeaveSensorApiServer) traits.EnterLeaveSensorApiClient {
	return &apiWrapper{server}
}

type apiWrapper struct {
	server traits.EnterLeaveSensorApiServer
}

// compile time check that we implement the interface we need
var _ traits.EnterLeaveSensorApiClient = (*apiWrapper)(nil)

// UnwrapServer returns the underlying server instance.
func (w *apiWrapper) UnwrapServer() traits.EnterLeaveSensorApiServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *apiWrapper) Unwrap() interface{} {
	return w.UnwrapServer()
}

func (w *apiWrapper) PullEnterLeaveEvents(ctx context.Context, in *traits.PullEnterLeaveEventsRequest, opts ...grpc.CallOption) (traits.EnterLeaveSensorApi_PullEnterLeaveEventsClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullEnterLeaveEventsApiServerWrapper{stream.Server()}
	client := &pullEnterLeaveEventsApiClientWrapper{stream.Client()}
	go func() {
		err := w.server.PullEnterLeaveEvents(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullEnterLeaveEventsApiClientWrapper struct {
	grpc.ClientStream
}

func (w *pullEnterLeaveEventsApiClientWrapper) Recv() (*traits.PullEnterLeaveEventsResponse, error) {
	m := new(traits.PullEnterLeaveEventsResponse)
	if err := w.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullEnterLeaveEventsApiServerWrapper struct {
	grpc.ServerStream
}

func (s *pullEnterLeaveEventsApiServerWrapper) Send(response *traits.PullEnterLeaveEventsResponse) error {
	return s.ServerStream.SendMsg(response)
}
