package wrap

import (
	"context"

	"git.vanti.co.uk/smartcore/sc-api/go/traits"
	"google.golang.org/grpc"
)

// OnOffApiServer adapts a traits.OnOffApiServer and presents it as a traits.OnOffApiClient
func OnOffApiServer(server traits.OnOffApiServer) traits.OnOffApiClient {
	return &onOffServerClient{server}
}

type onOffServerClient struct {
	server traits.OnOffApiServer
}

// compile time check that we implement the interface we need
var _ traits.OnOffApiClient = &onOffServerClient{}

func (c *onOffServerClient) GetOnOff(ctx context.Context, in *traits.GetOnOffRequest, _ ...grpc.CallOption) (*traits.OnOff, error) {
	return c.server.GetOnOff(ctx, in)
}

func (c *onOffServerClient) UpdateOnOff(ctx context.Context, in *traits.UpdateOnOffRequest, _ ...grpc.CallOption) (*traits.OnOff, error) {
	return c.server.UpdateOnOff(ctx, in)
}

func (c *onOffServerClient) PullOnOff(ctx context.Context, in *traits.PullOnOffRequest, _ ...grpc.CallOption) (traits.OnOffApi_PullOnOffClient, error) {
	stream := newClientServerStream(ctx)
	server := &onOffApiPullOnOffServer{stream.Server()}
	client := &onOffApiPullOnOffClient{stream.Client()}
	go func() {
		err := c.server.PullOnOff(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type onOffApiPullOnOffClient struct {
	grpc.ClientStream
}

func (c *onOffApiPullOnOffClient) Recv() (*traits.PullOnOffResponse, error) {
	m := new(traits.PullOnOffResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type onOffApiPullOnOffServer struct {
	grpc.ServerStream
}

func (s *onOffApiPullOnOffServer) Send(response *traits.PullOnOffResponse) error {
	return s.ServerStream.SendMsg(response)
}
