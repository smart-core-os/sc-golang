package count

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
	"google.golang.org/grpc"
)

// Wrap adapts a traits.CountApiServer and presents it as a traits.CountApiClient
func Wrap(server traits.CountApiServer) traits.CountApiClient {
	return &wrapper{server}
}

type wrapper struct {
	server traits.CountApiServer
}

// compile time check that we implement the interface we need
var _ traits.CountApiClient = (*wrapper)(nil)

func (c *wrapper) GetCount(ctx context.Context, in *traits.GetCountRequest, _ ...grpc.CallOption) (*traits.Count, error) {
	return c.server.GetCount(ctx, in)
}

func (c *wrapper) ResetCount(ctx context.Context, in *traits.ResetCountRequest, _ ...grpc.CallOption) (*traits.Count, error) {
	return c.server.ResetCount(ctx, in)
}

func (c *wrapper) UpdateCount(ctx context.Context, in *traits.UpdateCountRequest, _ ...grpc.CallOption) (*traits.Count, error) {
	return c.server.UpdateCount(ctx, in)
}

func (c *wrapper) PullCounts(ctx context.Context, in *traits.PullCountsRequest, _ ...grpc.CallOption) (traits.CountApi_PullCountsClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullCountsServerWrapper{stream.Server()}
	client := &pullCountsClientWrapper{stream.Client()}
	go func() {
		err := c.server.PullCounts(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullCountsClientWrapper struct {
	grpc.ClientStream
}

func (c *pullCountsClientWrapper) Recv() (*traits.PullCountsResponse, error) {
	m := new(traits.PullCountsResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullCountsServerWrapper struct {
	grpc.ServerStream
}

func (s *pullCountsServerWrapper) Send(response *traits.PullCountsResponse) error {
	return s.ServerStream.SendMsg(response)
}
