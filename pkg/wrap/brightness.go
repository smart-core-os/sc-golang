package wrap

import (
	"context"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	"google.golang.org/grpc"
)

// BrightnessApiClientFromServer adapts a BrightnessApiServer and presents it as a BrightnessApiClient
func BrightnessApiClientFromServer(server traits.BrightnessApiServer) traits.BrightnessApiClient {
	return &brightnessApiServerClient{server}
}

type brightnessApiServerClient struct {
	server traits.BrightnessApiServer
}

// compile time check that we implement the interface we need
var _ traits.BrightnessApiClient = &brightnessApiServerClient{}

func (b *brightnessApiServerClient) UpdateRangeValue(ctx context.Context, in *traits.UpdateBrightnessRequest, opts ...grpc.CallOption) (*traits.Brightness, error) {
	return b.server.UpdateRangeValue(ctx, in)
}

func (b *brightnessApiServerClient) GetBrightness(ctx context.Context, in *traits.GetBrightnessRequest, opts ...grpc.CallOption) (*traits.Brightness, error) {
	return b.server.GetBrightness(ctx, in)
}

func (b *brightnessApiServerClient) PullBrightness(ctx context.Context, in *traits.PullBrightnessRequest, opts ...grpc.CallOption) (traits.BrightnessApi_PullBrightnessClient, error) {
	stream := newClientServerStream(ctx)
	server := &brightnessApiPullBrightnessServer{stream.Server()}
	client := &brightnessApiPullBrightnessClient{stream.Client()}
	go func() {
		err := b.server.PullBrightness(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type brightnessApiPullBrightnessClient struct {
	grpc.ClientStream
}

func (c *brightnessApiPullBrightnessClient) Recv() (*traits.PullBrightnessResponse, error) {
	m := new(traits.PullBrightnessResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type brightnessApiPullBrightnessServer struct {
	grpc.ServerStream
}

func (s *brightnessApiPullBrightnessServer) Send(response *traits.PullBrightnessResponse) error {
	return s.ServerStream.SendMsg(response)
}
