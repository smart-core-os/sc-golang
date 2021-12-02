package electric

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
	"google.golang.org/grpc"
)

// Wrap adapts a traits.ElectricApiServer and presents it as a traits.ElectricApiClient
func Wrap(server traits.ElectricApiServer) traits.ElectricApiClient {
	return &wrapper{server}
}

type wrapper struct {
	server traits.ElectricApiServer
}

// compile time check that we implement the interface we need
var _ traits.ElectricApiClient = (*wrapper)(nil)

func (c *wrapper) GetDemand(ctx context.Context, in *traits.GetDemandRequest, opts ...grpc.CallOption) (*traits.ElectricDemand, error) {
	return c.server.GetDemand(ctx, in)
}

func (c *wrapper) PullDemand(ctx context.Context, in *traits.PullDemandRequest, opts ...grpc.CallOption) (traits.ElectricApi_PullDemandClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullDemandServerWrapper{stream.Server()}
	client := &pullDemandClientWrapper{stream.Client()}
	go func() {
		err := c.server.PullDemand(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullDemandClientWrapper struct {
	grpc.ClientStream
}

func (c *pullDemandClientWrapper) Recv() (*traits.PullDemandResponse, error) {
	m := new(traits.PullDemandResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullDemandServerWrapper struct {
	grpc.ServerStream
}

func (s *pullDemandServerWrapper) Send(response *traits.PullDemandResponse) error {
	return s.ServerStream.SendMsg(response)
}

func (c *wrapper) GetActiveMode(ctx context.Context, in *traits.GetActiveModeRequest, opts ...grpc.CallOption) (*traits.ElectricMode, error) {
	return c.server.GetActiveMode(ctx, in)
}

func (c *wrapper) UpdateActiveMode(ctx context.Context, in *traits.UpdateActiveModeRequest, opts ...grpc.CallOption) (*traits.ElectricMode, error) {
	return c.server.UpdateActiveMode(ctx, in)
}

func (c *wrapper) ClearActiveMode(ctx context.Context, in *traits.ClearActiveModeRequest, opts ...grpc.CallOption) (*traits.ElectricMode, error) {
	return c.server.ClearActiveMode(ctx, in)
}

func (c *wrapper) PullActiveMode(ctx context.Context, in *traits.PullActiveModeRequest, opts ...grpc.CallOption) (traits.ElectricApi_PullActiveModeClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullActiveModeServerWrapper{stream.Server()}
	client := &pullActiveModeClientWrapper{stream.Client()}
	go func() {
		err := c.server.PullActiveMode(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullActiveModeClientWrapper struct {
	grpc.ClientStream
}

func (c *pullActiveModeClientWrapper) Recv() (*traits.PullActiveModeResponse, error) {
	m := new(traits.PullActiveModeResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullActiveModeServerWrapper struct {
	grpc.ServerStream
}

func (s *pullActiveModeServerWrapper) Send(response *traits.PullActiveModeResponse) error {
	return s.ServerStream.SendMsg(response)
}

func (c *wrapper) ListModes(ctx context.Context, in *traits.ListModesRequest, opts ...grpc.CallOption) (*traits.ListModesResponse, error) {
	return c.server.ListModes(ctx, in)
}

func (c *wrapper) PullModes(ctx context.Context, in *traits.PullModesRequest, opts ...grpc.CallOption) (traits.ElectricApi_PullModesClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullModesServerWrapper{stream.Server()}
	client := &pullModesClientWrapper{stream.Client()}
	go func() {
		err := c.server.PullModes(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullModesClientWrapper struct {
	grpc.ClientStream
}

func (c *pullModesClientWrapper) Recv() (*traits.PullModesResponse, error) {
	m := new(traits.PullModesResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullModesServerWrapper struct {
	grpc.ServerStream
}

func (s *pullModesServerWrapper) Send(response *traits.PullModesResponse) error {
	return s.ServerStream.SendMsg(response)
}
