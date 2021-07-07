package wrap

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc"
)

// AirTemperatureApiServer adapts a traits.AirTemperatureApiServer and presents it as a traits.AirTemperatureApiClient
func AirTemperatureApiServer(server traits.AirTemperatureApiServer) traits.AirTemperatureApiClient {
	return &airTemperatureApiServerClient{server}
}

type airTemperatureApiServerClient struct {
	server traits.AirTemperatureApiServer
}

// compile time check that we implement the interface we need
var _ traits.AirTemperatureApiClient = &airTemperatureApiServerClient{}

func (b *airTemperatureApiServerClient) GetAirTemperature(ctx context.Context, in *traits.GetAirTemperatureRequest, _ ...grpc.CallOption) (*traits.AirTemperature, error) {
	return b.server.GetAirTemperature(ctx, in)
}

func (b *airTemperatureApiServerClient) UpdateAirTemperature(ctx context.Context, in *traits.UpdateAirTemperatureRequest, _ ...grpc.CallOption) (*traits.AirTemperature, error) {
	return b.server.UpdateAirTemperature(ctx, in)
}

func (b *airTemperatureApiServerClient) PullAirTemperature(ctx context.Context, in *traits.PullAirTemperatureRequest, _ ...grpc.CallOption) (traits.AirTemperatureApi_PullAirTemperatureClient, error) {
	stream := newClientServerStream(ctx)
	server := &thermostatPullAirTemperatureServer{stream.Server()}
	client := &thermostatPullAirTemperatureClient{stream.Client()}
	go func() {
		err := b.server.PullAirTemperature(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type thermostatPullAirTemperatureClient struct {
	grpc.ClientStream
}

func (c *thermostatPullAirTemperatureClient) Recv() (*traits.PullAirTemperatureResponse, error) {
	m := new(traits.PullAirTemperatureResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type thermostatPullAirTemperatureServer struct {
	grpc.ServerStream
}

func (s *thermostatPullAirTemperatureServer) Send(response *traits.PullAirTemperatureResponse) error {
	return s.ServerStream.SendMsg(response)
}
