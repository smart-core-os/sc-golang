package wrap

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// PowerSupplyApiServer adapts a traits.PowerSupplyApiServer and presents it as a traits.PowerSupplyApiClient
func PowerSupplyApiServer(server traits.PowerSupplyApiServer) traits.PowerSupplyApiClient {
	return &powerSupplyApiServerClient{server}
}

type powerSupplyApiServerClient struct {
	server traits.PowerSupplyApiServer
}

// compile time check that we implement the interface we need
var _ traits.PowerSupplyApiClient = &powerSupplyApiServerClient{}

func (b *powerSupplyApiServerClient) GetPowerCapacity(ctx context.Context, in *traits.GetPowerCapacityRequest, _ ...grpc.CallOption) (*traits.PowerCapacity, error) {
	return b.server.GetPowerCapacity(ctx, in)
}

func (b *powerSupplyApiServerClient) PullPowerCapacity(ctx context.Context, in *traits.PullPowerCapacityRequest, _ ...grpc.CallOption) (traits.PowerSupplyApi_PullPowerCapacityClient, error) {
	stream := newClientServerStream(ctx)
	server := &powerSupplyApiPullPowerCapacityServer{stream.Server()}
	client := &powerSupplyApiPullPowerCapacityClient{stream.Client()}
	go func() {
		err := b.server.PullPowerCapacity(in, server)
		stream.Close(err)
	}()
	return client, nil
}

func (b *powerSupplyApiServerClient) CreateDrawNotification(ctx context.Context, in *traits.CreateDrawNotificationRequest, _ ...grpc.CallOption) (*traits.DrawNotification, error) {
	return b.server.CreateDrawNotification(ctx, in)
}

func (b *powerSupplyApiServerClient) UpdateDrawNotification(ctx context.Context, in *traits.UpdateDrawNotificationRequest, _ ...grpc.CallOption) (*traits.DrawNotification, error) {
	return b.server.UpdateDrawNotification(ctx, in)
}

func (b *powerSupplyApiServerClient) DeleteDrawNotification(ctx context.Context, in *traits.DeleteDrawNotificationRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return b.server.DeleteDrawNotification(ctx, in)
}

type powerSupplyApiPullPowerCapacityClient struct {
	grpc.ClientStream
}

func (c *powerSupplyApiPullPowerCapacityClient) Recv() (*traits.PullPowerCapacityResponse, error) {
	m := new(traits.PullPowerCapacityResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type powerSupplyApiPullPowerCapacityServer struct {
	grpc.ServerStream
}

func (s *powerSupplyApiPullPowerCapacityServer) Send(response *traits.PullPowerCapacityResponse) error {
	return s.ServerStream.SendMsg(response)
}
