package wrap

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc"
)

// OccupancySensorApiServer adapts a traits.OccupancySensorApiServer and presents it as a traits.OccupancySensorApiClient
func OccupancySensorApiServer(server traits.OccupancySensorApiServer) traits.OccupancySensorApiClient {
	return &occupancySensorApiServerClient{server}
}

type occupancySensorApiServerClient struct {
	server traits.OccupancySensorApiServer
}

// compile time check that we implement the interface we need
var _ traits.OccupancySensorApiClient = &occupancySensorApiServerClient{}

func (c *occupancySensorApiServerClient) GetOccupancy(ctx context.Context, in *traits.GetOccupancyRequest, _ ...grpc.CallOption) (*traits.Occupancy, error) {
	return c.server.GetOccupancy(ctx, in)
}

func (c *occupancySensorApiServerClient) PullOccupancy(ctx context.Context, in *traits.PullOccupancyRequest, _ ...grpc.CallOption) (traits.OccupancySensorApi_PullOccupancyClient, error) {
	stream := newClientServerStream(ctx)
	server := &occupancySensorApiPullOccupancyServer{stream.Server()}
	client := &occupancySensorApiPullOccupancyClient{stream.Client()}
	go func() {
		err := c.server.PullOccupancy(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type occupancySensorApiPullOccupancyClient struct {
	grpc.ClientStream
}

func (c *occupancySensorApiPullOccupancyClient) Recv() (*traits.PullOccupancyResponse, error) {
	m := new(traits.PullOccupancyResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type occupancySensorApiPullOccupancyServer struct {
	grpc.ServerStream
}

func (s *occupancySensorApiPullOccupancyServer) Send(response *traits.PullOccupancyResponse) error {
	return s.ServerStream.SendMsg(response)
}
