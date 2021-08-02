package light

import (
	"context"

	"github.com/smart-core-os/sc-golang/pkg/wrap"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc"
)

// Wrap adapts a traits.LightApiServer and presents it as a traits.LightApiClient
func Wrap(server traits.LightApiServer) traits.LightApiClient {
	return &wrapper{server}
}

type wrapper struct {
	server traits.LightApiServer
}

// compile time check that we implement the interface we need
var _ traits.LightApiClient = &wrapper{}

func (c *wrapper) UpdateBrightness(ctx context.Context, in *traits.UpdateBrightnessRequest, opts ...grpc.CallOption) (*traits.Brightness, error) {
	return c.server.UpdateBrightness(ctx, in)
}

func (c *wrapper) GetBrightness(ctx context.Context, in *traits.GetBrightnessRequest, opts ...grpc.CallOption) (*traits.Brightness, error) {
	return c.server.GetBrightness(ctx, in)
}

func (c *wrapper) PullBrightness(ctx context.Context, in *traits.PullBrightnessRequest, opts ...grpc.CallOption) (traits.LightApi_PullBrightnessClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullBrightnessServerWrapper{stream.Server()}
	client := &pullBrightnessClientWrapper{stream.Client()}
	go func() {
		err := c.server.PullBrightness(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullBrightnessClientWrapper struct {
	grpc.ClientStream
}

func (c *pullBrightnessClientWrapper) Recv() (*traits.PullBrightnessResponse, error) {
	m := new(traits.PullBrightnessResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullBrightnessServerWrapper struct {
	grpc.ServerStream
}

func (s *pullBrightnessServerWrapper) Send(response *traits.PullBrightnessResponse) error {
	return s.ServerStream.SendMsg(response)
}
