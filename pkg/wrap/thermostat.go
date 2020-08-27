package wrap

import (
	"context"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	"google.golang.org/grpc"
)

// ThermostatClientFromServer adapts a ThermostatServer and presents it as a ThermostatClient
func ThermostatClientFromServer(server traits.ThermostatServer) traits.ThermostatClient {
	return &thermostatServerClient{server}
}

type thermostatServerClient struct {
	server traits.ThermostatServer
}

// compile time check that we implement the interface we need
var _ traits.ThermostatClient = &thermostatServerClient{}

func (b *thermostatServerClient) GetState(ctx context.Context, in *traits.GetThermostatStateRequest, opts ...grpc.CallOption) (*traits.ThermostatState, error) {
	return b.server.GetState(ctx, in)
}

func (b *thermostatServerClient) UpdateState(ctx context.Context, in *traits.UpdateThermostatStateRequest, opts ...grpc.CallOption) (*traits.ThermostatState, error) {
	return b.server.UpdateState(ctx, in)
}

func (b *thermostatServerClient) PullState(ctx context.Context, in *traits.PullThermostatStateRequest, opts ...grpc.CallOption) (traits.Thermostat_PullStateClient, error) {
	stream := newClientServerStream(ctx)
	server := &thermostatPullStateServer{stream.Server()}
	client := &thermostatPullStateClient{stream.Client()}
	go func() {
		err := b.server.PullState(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type thermostatPullStateClient struct {
	grpc.ClientStream
}

func (c *thermostatPullStateClient) Recv() (*traits.PullThermostatStateResponse, error) {
	m := new(traits.PullThermostatStateResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type thermostatPullStateServer struct {
	grpc.ServerStream
}

func (s *thermostatPullStateServer) Send(response *traits.PullThermostatStateResponse) error {
	return s.ServerStream.SendMsg(response)
}
