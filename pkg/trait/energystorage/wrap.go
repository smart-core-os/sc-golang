package energystorage

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
	"google.golang.org/grpc"
)

// Wrap adapts a traits.EnergyStorageApiServer and presents it as a traits.EnergyStorageApiClient
func Wrap(server traits.EnergyStorageApiServer) traits.EnergyStorageApiClient {
	return &wrapper{server}
}

type wrapper struct {
	server traits.EnergyStorageApiServer
}

// compile time check that we implement the interface we need
var _ traits.EnergyStorageApiClient = (*wrapper)(nil)

func (w *wrapper) GetEnergyLevel(ctx context.Context, in *traits.GetEnergyLevelRequest, _ ...grpc.CallOption) (*traits.EnergyLevel, error) {
	return w.server.GetEnergyLevel(ctx, in)
}

func (w *wrapper) PullEnergyLevel(ctx context.Context, in *traits.PullEnergyLevelRequest, _ ...grpc.CallOption) (traits.EnergyStorageApi_PullEnergyLevelClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullEnergyLevelServerWrapper{stream.Server()}
	client := &pullEnergyLevelClientWrapper{stream.Client()}
	go func() {
		err := w.server.PullEnergyLevel(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullEnergyLevelClientWrapper struct {
	grpc.ClientStream
}

func (c *pullEnergyLevelClientWrapper) Recv() (*traits.PullEnergyLevelResponse, error) {
	m := new(traits.PullEnergyLevelResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullEnergyLevelServerWrapper struct {
	grpc.ServerStream
}

func (s *pullEnergyLevelServerWrapper) Send(response *traits.PullEnergyLevelResponse) error {
	return s.ServerStream.SendMsg(response)
}

func (w *wrapper) Charge(ctx context.Context, in *traits.ChargeRequest, _ ...grpc.CallOption) (*traits.ChargeResponse, error) {
	return w.server.Charge(ctx, in)
}
