package wrap

import (
	"context"

	"git.vanti.co.uk/smartcore/sc-api/go/traits"
	"google.golang.org/grpc"
)

// LightApiServer adapts a traits.LightApiServer and presents it as a traits.LightApiClient
func LightApiServer(server traits.LightApiServer) traits.LightApiClient {
	return &lightApiServerClient{server}
}

type lightApiServerClient struct {
	server traits.LightApiServer
}

// compile time check that we implement the interface we need
var _ traits.LightApiClient = &lightApiServerClient{}

func (b *lightApiServerClient) UpdateBrightness(ctx context.Context, in *traits.UpdateBrightnessRequest, opts ...grpc.CallOption) (*traits.Brightness, error) {
	return b.server.UpdateBrightness(ctx, in)
}

func (b *lightApiServerClient) GetBrightness(ctx context.Context, in *traits.GetBrightnessRequest, opts ...grpc.CallOption) (*traits.Brightness, error) {
	return b.server.GetBrightness(ctx, in)
}

func (b *lightApiServerClient) PullBrightness(ctx context.Context, in *traits.PullBrightnessRequest, opts ...grpc.CallOption) (traits.LightApi_PullBrightnessClient, error) {
	stream := newClientServerStream(ctx)
	server := &lightApiPullBrightnessServer{stream.Server()}
	client := &lightApiPullBrightnessClient{stream.Client()}
	go func() {
		err := b.server.PullBrightness(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type lightApiPullBrightnessClient struct {
	grpc.ClientStream
}

func (c *lightApiPullBrightnessClient) Recv() (*traits.PullBrightnessResponse, error) {
	m := new(traits.PullBrightnessResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type lightApiPullBrightnessServer struct {
	grpc.ServerStream
}

func (s *lightApiPullBrightnessServer) Send(response *traits.PullBrightnessResponse) error {
	return s.ServerStream.SendMsg(response)
}
