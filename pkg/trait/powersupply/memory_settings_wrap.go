package powersupply

import (
	"context"

	"github.com/smart-core-os/sc-golang/pkg/wrap"
	"google.golang.org/grpc"
)

// WrapMemorySettings adapts a MemorySettingsApiServer and presents it as a MemorySettingsApiClient.
func WrapMemorySettings(server MemorySettingsApiServer) MemorySettingsApiClient {
	return &memorySettingsWrapper{server}
}

type memorySettingsWrapper struct {
	server MemorySettingsApiServer
}

var _ MemorySettingsApiClient = (*memorySettingsWrapper)(nil) // compiler check for implementing the interface

func (w *memorySettingsWrapper) GetSettings(ctx context.Context, in *GetMemorySettingsReq, opts ...grpc.CallOption) (*MemorySettings, error) {
	return w.server.GetSettings(ctx, in)
}

func (w *memorySettingsWrapper) UpdateSettings(ctx context.Context, in *UpdateMemorySettingsReq, opts ...grpc.CallOption) (*MemorySettings, error) {
	return w.server.UpdateSettings(ctx, in)
}

func (w *memorySettingsWrapper) PullSettings(ctx context.Context, in *PullMemorySettingsReq, opts ...grpc.CallOption) (MemorySettingsApi_PullSettingsClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullSettingsServerWrapper{stream.Server()}
	client := &pullSettingsClientWrapper{stream.Client()}
	go func() {
		err := w.server.PullSettings(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullSettingsClientWrapper struct {
	grpc.ClientStream
}

func (c *pullSettingsClientWrapper) Recv() (*PullMemorySettingsRes, error) {
	m := new(PullMemorySettingsRes)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullSettingsServerWrapper struct {
	grpc.ServerStream
}

func (s *pullSettingsServerWrapper) Send(response *PullMemorySettingsRes) error {
	return s.ServerStream.SendMsg(response)
}
