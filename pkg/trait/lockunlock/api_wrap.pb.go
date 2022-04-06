// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package lockunlock

import (
	context "context"
	traits "github.com/smart-core-os/sc-api/go/traits"
	wrap "github.com/smart-core-os/sc-golang/pkg/wrap"
	grpc "google.golang.org/grpc"
)

// WrapApi	adapts a traits.LockUnlockApiServer	and presents it as a traits.LockUnlockApiClient
func WrapApi(server traits.LockUnlockApiServer) traits.LockUnlockApiClient {
	return &apiWrapper{server}
}

type apiWrapper struct {
	server traits.LockUnlockApiServer
}

// compile time check that we implement the interface we need
var _ traits.LockUnlockApiClient = (*apiWrapper)(nil)

// UnwrapServer returns the underlying server instance.
func (w *apiWrapper) UnwrapServer() traits.LockUnlockApiServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *apiWrapper) Unwrap() interface{} {
	return w.UnwrapServer()
}

func (w *apiWrapper) GetLockUnlock(ctx context.Context, req *traits.GetLockUnlockRequest, _ ...grpc.CallOption) (*traits.LockUnlock, error) {
	return w.server.GetLockUnlock(ctx, req)
}

func (w *apiWrapper) UpdateLockUnlock(ctx context.Context, req *traits.UpdateLockUnlockRequest, _ ...grpc.CallOption) (*traits.LockUnlock, error) {
	return w.server.UpdateLockUnlock(ctx, req)
}

func (w *apiWrapper) PullLockUnlock(ctx context.Context, in *traits.PullLockUnlockRequest, opts ...grpc.CallOption) (traits.LockUnlockApi_PullLockUnlockClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullLockUnlockApiServerWrapper{stream.Server()}
	client := &pullLockUnlockApiClientWrapper{stream.Client()}
	go func() {
		err := w.server.PullLockUnlock(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullLockUnlockApiClientWrapper struct {
	grpc.ClientStream
}

func (w *pullLockUnlockApiClientWrapper) Recv() (*traits.PullLockUnlockResponse, error) {
	m := new(traits.PullLockUnlockResponse)
	if err := w.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullLockUnlockApiServerWrapper struct {
	grpc.ServerStream
}

func (s *pullLockUnlockApiServerWrapper) Send(response *traits.PullLockUnlockResponse) error {
	return s.ServerStream.SendMsg(response)
}
