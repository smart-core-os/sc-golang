package wrap

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc"
)

// CountApiServer adapts a traits.CountApiServer and presents it as a traits.CountApiClient
func CountApiServer(server traits.CountApiServer) traits.CountApiClient {
	return &countApiServerClient{server}
}

type countApiServerClient struct {
	server traits.CountApiServer
}

// compile time check that we implement the interface we need
var _ traits.CountApiClient = &countApiServerClient{}

func (c *countApiServerClient) GetCount(ctx context.Context, in *traits.GetCountRequest, _ ...grpc.CallOption) (*traits.Count, error) {
	return c.server.GetCount(ctx, in)
}

func (c *countApiServerClient) ResetCount(ctx context.Context, in *traits.ResetCountRequest, _ ...grpc.CallOption) (*traits.Count, error) {
	return c.server.ResetCount(ctx, in)
}

func (c *countApiServerClient) UpdateCount(ctx context.Context, in *traits.UpdateCountRequest, _ ...grpc.CallOption) (*traits.Count, error) {
	return c.server.UpdateCount(ctx, in)
}

func (c *countApiServerClient) PullCounts(ctx context.Context, in *traits.PullCountsRequest, _ ...grpc.CallOption) (traits.CountApi_PullCountsClient, error) {
	stream := NewClientServerStream(ctx)
	server := &countApiPullCountsServer{stream.Server()}
	client := &countApiPullCountsClient{stream.Client()}
	go func() {
		err := c.server.PullCounts(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type countApiPullCountsClient struct {
	grpc.ClientStream
}

func (c *countApiPullCountsClient) Recv() (*traits.PullCountsResponse, error) {
	m := new(traits.PullCountsResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type countApiPullCountsServer struct {
	grpc.ServerStream
}

func (s *countApiPullCountsServer) Send(response *traits.PullCountsResponse) error {
	return s.ServerStream.SendMsg(response)
}
