// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package emergency

import (
	context "context"
	traits "github.com/smart-core-os/sc-api/go/traits"
	wrap "github.com/smart-core-os/sc-golang/pkg/wrap"
	grpc "google.golang.org/grpc"
)

// WrapApi	adapts a traits.EmergencyApiServer	and presents it as a traits.EmergencyApiClient
func WrapApi(server traits.EmergencyApiServer) traits.EmergencyApiClient {
	return &apiWrapper{server}
}

type apiWrapper struct {
	server traits.EmergencyApiServer
}

// compile time check that we implement the interface we need
var _ traits.EmergencyApiClient = (*apiWrapper)(nil)

// UnwrapServer returns the underlying server instance.
func (w *apiWrapper) UnwrapServer() traits.EmergencyApiServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *apiWrapper) Unwrap() any {
	return w.UnwrapServer()
}

func (w *apiWrapper) GetEmergency(ctx context.Context, req *traits.GetEmergencyRequest, _ ...grpc.CallOption) (*traits.Emergency, error) {
	return w.server.GetEmergency(ctx, req)
}

func (w *apiWrapper) UpdateEmergency(ctx context.Context, req *traits.UpdateEmergencyRequest, _ ...grpc.CallOption) (*traits.Emergency, error) {
	return w.server.UpdateEmergency(ctx, req)
}

func (w *apiWrapper) PullEmergency(ctx context.Context, in *traits.PullEmergencyRequest, opts ...grpc.CallOption) (traits.EmergencyApi_PullEmergencyClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullEmergencyApiServerWrapper{stream.Server()}
	client := &pullEmergencyApiClientWrapper{stream.Client()}
	go func() {
		err := w.server.PullEmergency(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullEmergencyApiClientWrapper struct {
	grpc.ClientStream
}

func (w *pullEmergencyApiClientWrapper) Recv() (*traits.PullEmergencyResponse, error) {
	m := new(traits.PullEmergencyResponse)
	if err := w.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullEmergencyApiServerWrapper struct {
	grpc.ServerStream
}

func (s *pullEmergencyApiServerWrapper) Send(response *traits.PullEmergencyResponse) error {
	return s.ServerStream.SendMsg(response)
}
