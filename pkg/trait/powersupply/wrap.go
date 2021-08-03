package powersupply

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Wrap adapts a traits.PowerSupplyApiServer and presents it as a traits.PowerSupplyApiClient.
func Wrap(server traits.PowerSupplyApiServer) traits.PowerSupplyApiClient {
	return &wrapper{server}
}

type wrapper struct {
	server traits.PowerSupplyApiServer
}

// compile time check that we implement the interface we need
var _ traits.PowerSupplyApiClient = (*wrapper)(nil)

func (w *wrapper) GetPowerCapacity(ctx context.Context, in *traits.GetPowerCapacityRequest, _ ...grpc.CallOption) (*traits.PowerCapacity, error) {
	return w.server.GetPowerCapacity(ctx, in)
}

func (w *wrapper) PullPowerCapacity(ctx context.Context, in *traits.PullPowerCapacityRequest, _ ...grpc.CallOption) (traits.PowerSupplyApi_PullPowerCapacityClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullPowerCapacityServerWrapper{stream.Server()}
	client := &pullPowerCapacityClientWrapper{stream.Client()}
	go func() {
		err := w.server.PullPowerCapacity(in, server)
		stream.Close(err)
	}()
	return client, nil
}

func (w *wrapper) CreateDrawNotification(ctx context.Context, in *traits.CreateDrawNotificationRequest, _ ...grpc.CallOption) (*traits.DrawNotification, error) {
	return w.server.CreateDrawNotification(ctx, in)
}

func (w *wrapper) UpdateDrawNotification(ctx context.Context, in *traits.UpdateDrawNotificationRequest, _ ...grpc.CallOption) (*traits.DrawNotification, error) {
	return w.server.UpdateDrawNotification(ctx, in)
}

func (w *wrapper) DeleteDrawNotification(ctx context.Context, in *traits.DeleteDrawNotificationRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return w.server.DeleteDrawNotification(ctx, in)
}

type pullPowerCapacityClientWrapper struct {
	grpc.ClientStream
}

func (c *pullPowerCapacityClientWrapper) Recv() (*traits.PullPowerCapacityResponse, error) {
	m := new(traits.PullPowerCapacityResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullPowerCapacityServerWrapper struct {
	grpc.ServerStream
}

func (s *pullPowerCapacityServerWrapper) Send(response *traits.PullPowerCapacityResponse) error {
	return s.ServerStream.SendMsg(response)
}
