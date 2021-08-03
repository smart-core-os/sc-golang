package onoff

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
	"google.golang.org/grpc"
)

// Wrap adapts a traits.OnOffApiServer and presents it as a traits.OnOffApiClient
func Wrap(server traits.OnOffApiServer) traits.OnOffApiClient {
	return &wrapper{server}
}

type wrapper struct {
	server traits.OnOffApiServer
}

// compile time check that we implement the interface we need
var _ traits.OnOffApiClient = (*wrapper)(nil)

func (c *wrapper) GetOnOff(ctx context.Context, in *traits.GetOnOffRequest, _ ...grpc.CallOption) (*traits.OnOff, error) {
	return c.server.GetOnOff(ctx, in)
}

func (c *wrapper) UpdateOnOff(ctx context.Context, in *traits.UpdateOnOffRequest, _ ...grpc.CallOption) (*traits.OnOff, error) {
	return c.server.UpdateOnOff(ctx, in)
}

func (c *wrapper) PullOnOff(ctx context.Context, in *traits.PullOnOffRequest, _ ...grpc.CallOption) (traits.OnOffApi_PullOnOffClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullOnOffServerWrapper{stream.Server()}
	client := &pullOnOffClientWrapper{stream.Client()}
	go func() {
		err := c.server.PullOnOff(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullOnOffClientWrapper struct {
	grpc.ClientStream
}

func (c *pullOnOffClientWrapper) Recv() (*traits.PullOnOffResponse, error) {
	m := new(traits.PullOnOffResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullOnOffServerWrapper struct {
	grpc.ServerStream
}

func (s *pullOnOffServerWrapper) Send(response *traits.PullOnOffResponse) error {
	return s.ServerStream.SendMsg(response)
}
