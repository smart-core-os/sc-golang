package airtemperature

import (
	"context"

	"github.com/smart-core-os/sc-golang/pkg/wrap"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc"
)

// Wrap adapts a traits.AirTemperatureApiServer and presents it as a traits.AirTemperatureApiClient
func Wrap(server traits.AirTemperatureApiServer) traits.AirTemperatureApiClient {
	return &wrapper{server}
}

type wrapper struct {
	server traits.AirTemperatureApiServer
}

// compile time check that we implement the interface we need
var _ traits.AirTemperatureApiClient = (*wrapper)(nil)

func (b *wrapper) GetAirTemperature(ctx context.Context, in *traits.GetAirTemperatureRequest, _ ...grpc.CallOption) (*traits.AirTemperature, error) {
	return b.server.GetAirTemperature(ctx, in)
}

func (b *wrapper) UpdateAirTemperature(ctx context.Context, in *traits.UpdateAirTemperatureRequest, _ ...grpc.CallOption) (*traits.AirTemperature, error) {
	return b.server.UpdateAirTemperature(ctx, in)
}

func (b *wrapper) PullAirTemperature(ctx context.Context, in *traits.PullAirTemperatureRequest, _ ...grpc.CallOption) (traits.AirTemperatureApi_PullAirTemperatureClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullAirTemperatureServerWrapper{stream.Server()}
	client := &pullAirTemperatureClientWrapper{stream.Client()}
	go func() {
		err := b.server.PullAirTemperature(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullAirTemperatureClientWrapper struct {
	grpc.ClientStream
}

func (c *pullAirTemperatureClientWrapper) Recv() (*traits.PullAirTemperatureResponse, error) {
	m := new(traits.PullAirTemperatureResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullAirTemperatureServerWrapper struct {
	grpc.ServerStream
}

func (s *pullAirTemperatureServerWrapper) Send(response *traits.PullAirTemperatureResponse) error {
	return s.ServerStream.SendMsg(response)
}
