package wrap

import (
	"context"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	"google.golang.org/grpc"
)

// OnOffClientFromServer adapts a OnOffServer and presents it as a OnOffClient
func OnOffClientFromServer(server traits.OnOffServer) traits.OnOffClient {
	return &onOffServerClient{server}
}

type onOffServerClient struct {
	server traits.OnOffServer
}

// compile time check that we implement the interface we need
var _ traits.OnOffClient = &onOffServerClient{}

func (c *onOffServerClient) On(ctx context.Context, in *traits.OnRequest, opts ...grpc.CallOption) (*traits.OnReply, error) {
	return c.server.On(ctx, in)
}

func (c *onOffServerClient) Off(ctx context.Context, in *traits.OffRequest, opts ...grpc.CallOption) (*traits.OffReply, error) {
	return c.server.Off(ctx, in)
}

func (c *onOffServerClient) GetOnOffState(ctx context.Context, in *traits.GetOnOffStateRequest, opts ...grpc.CallOption) (*traits.GetOnOffStateResponse, error) {
	return c.server.GetOnOffState(ctx, in)
}

func (c *onOffServerClient) Pull(ctx context.Context, in *traits.OnOffPullRequest, opts ...grpc.CallOption) (traits.OnOff_PullClient, error) {
	stream := newClientServerStream(ctx)
	server := &onOffPullOnOffStateServer{stream.Server()}
	client := &onOffPullOnOffStateClient{stream.Client()}
	go func() {
		err := c.server.Pull(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type onOffPullOnOffStateClient struct {
	grpc.ClientStream
}

func (c *onOffPullOnOffStateClient) Recv() (*traits.OnOffPullResponse, error) {
	m := new(traits.OnOffPullResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type onOffPullOnOffStateServer struct {
	grpc.ServerStream
}

func (s *onOffPullOnOffStateServer) Send(response *traits.OnOffPullResponse) error {
	return s.ServerStream.SendMsg(response)
}
