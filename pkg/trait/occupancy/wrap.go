package occupancy

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
	"google.golang.org/grpc"
)

// Wrap adapts a traits.OccupancySensorApiServer and presents it as a traits.OccupancySensorApiClient
func Wrap(server traits.OccupancySensorApiServer) traits.OccupancySensorApiClient {
	return &wrapper{server}
}

type wrapper struct {
	server traits.OccupancySensorApiServer
}

// compile time check that we implement the interface we need
var _ traits.OccupancySensorApiClient = (*wrapper)(nil)

func (c *wrapper) GetOccupancy(ctx context.Context, in *traits.GetOccupancyRequest, _ ...grpc.CallOption) (*traits.Occupancy, error) {
	return c.server.GetOccupancy(ctx, in)
}

func (c *wrapper) PullOccupancy(ctx context.Context, in *traits.PullOccupancyRequest, _ ...grpc.CallOption) (traits.OccupancySensorApi_PullOccupancyClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullOccupancyServerWrapper{stream.Server()}
	client := &pullOccupancyClientWrapper{stream.Client()}
	go func() {
		err := c.server.PullOccupancy(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullOccupancyClientWrapper struct {
	grpc.ClientStream
}

func (c *pullOccupancyClientWrapper) Recv() (*traits.PullOccupancyResponse, error) {
	m := new(traits.PullOccupancyResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullOccupancyServerWrapper struct {
	grpc.ServerStream
}

func (s *pullOccupancyServerWrapper) Send(response *traits.PullOccupancyResponse) error {
	return s.ServerStream.SendMsg(response)
}
