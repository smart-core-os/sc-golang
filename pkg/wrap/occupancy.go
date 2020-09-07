package wrap

import (
	"context"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

// OccupancyApiClientFromServer adapts a OccupancyApiServer and presents it as a OccupancyApiClient
func OccupancyApiClientFromServer(server traits.OccupancyApiServer) traits.OccupancyApiClient {
	return &occupancyApiServerClient{server}
}

type occupancyApiServerClient struct {
	server traits.OccupancyApiServer
}

// compile time check that we implement the interface we need
var _ traits.OccupancyApiClient = &occupancyApiServerClient{}

func (c *occupancyApiServerClient) GetOccupancy(ctx context.Context, in *traits.GetOccupancyRequest, _ ...grpc.CallOption) (*traits.Occupancy, error) {
	return c.server.GetOccupancy(ctx, in)
}

func (c *occupancyApiServerClient) PullOccupancy(ctx context.Context, in *traits.PullOccupancyRequest, _ ...grpc.CallOption) (traits.OccupancyApi_PullOccupancyClient, error) {
	stream := newClientServerStream(ctx)
	server := &occupancyApiPullOccupancyServer{stream.Server()}
	client := &occupancyApiPullOccupancyClient{stream.Client()}
	go func() {
		err := c.server.PullOccupancy(in, server)
		stream.Close(err)
	}()
	return client, nil
}

func (c *occupancyApiServerClient) CreateOccupancyOverride(ctx context.Context, in *traits.CreateOccupancyOverrideRequest, _ ...grpc.CallOption) (*traits.OccupancyOverride, error) {
	return c.server.CreateOccupancyOverride(ctx, in)
}

func (c *occupancyApiServerClient) UpdateOccupancyOverride(ctx context.Context, in *traits.UpdateOccupancyOverrideRequest, _ ...grpc.CallOption) (*traits.OccupancyOverride, error) {
	return c.server.UpdateOccupancyOverride(ctx, in)
}

func (c *occupancyApiServerClient) DeleteOccupancyOverride(ctx context.Context, in *traits.DeleteOccupancyOverrideRequest, _ ...grpc.CallOption) (*empty.Empty, error) {
	return c.server.DeleteOccupancyOverride(ctx, in)
}

func (c *occupancyApiServerClient) GetOccupancyOverride(ctx context.Context, in *traits.GetOccupancyOverrideRequest, _ ...grpc.CallOption) (*traits.OccupancyOverride, error) {
	return c.server.GetOccupancyOverride(ctx, in)
}

func (c *occupancyApiServerClient) ListOccupancyOverrides(ctx context.Context, in *traits.ListOccupancyOverridesRequest, _ ...grpc.CallOption) (*traits.ListOccupancyOverridesResponse, error) {
	return c.server.ListOccupancyOverrides(ctx, in)
}

type occupancyApiPullOccupancyClient struct {
	grpc.ClientStream
}

func (c *occupancyApiPullOccupancyClient) Recv() (*traits.PullOccupancyResponse, error) {
	m := new(traits.PullOccupancyResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type occupancyApiPullOccupancyServer struct {
	grpc.ServerStream
}

func (s *occupancyApiPullOccupancyServer) Send(response *traits.PullOccupancyResponse) error {
	return s.ServerStream.SendMsg(response)
}
