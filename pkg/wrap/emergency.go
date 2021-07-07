package wrap

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc"
)

// EmergencyApiServer adapts a traits.EmergencyApiServer and presents it as a traits.EmergencyApiClient
func EmergencyApiServer(server traits.EmergencyApiServer) traits.EmergencyApiClient {
	return &emergencyApiServerClient{server}
}

type emergencyApiServerClient struct {
	server traits.EmergencyApiServer
}

// compile time check that we implement the interface we need
var _ traits.EmergencyApiClient = &emergencyApiServerClient{}

func (b *emergencyApiServerClient) GetEmergency(ctx context.Context, in *traits.GetEmergencyRequest, opts ...grpc.CallOption) (*traits.Emergency, error) {
	return b.server.GetEmergency(ctx, in)
}

func (b *emergencyApiServerClient) UpdateEmergency(ctx context.Context, in *traits.UpdateEmergencyRequest, opts ...grpc.CallOption) (*traits.Emergency, error) {
	return b.server.UpdateEmergency(ctx, in)
}

func (b *emergencyApiServerClient) PullEmergency(ctx context.Context, in *traits.PullEmergencyRequest, opts ...grpc.CallOption) (traits.EmergencyApi_PullEmergencyClient, error) {
	stream := newClientServerStream(ctx)
	server := &emergencyApiPullEmergencyServer{stream.Server()}
	client := &emergencyApiPullEmergencyClient{stream.Client()}
	go func() {
		err := b.server.PullEmergency(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type emergencyApiPullEmergencyClient struct {
	grpc.ClientStream
}

func (c *emergencyApiPullEmergencyClient) Recv() (*traits.PullEmergencyResponse, error) {
	m := new(traits.PullEmergencyResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type emergencyApiPullEmergencyServer struct {
	grpc.ServerStream
}

func (s *emergencyApiPullEmergencyServer) Send(response *traits.PullEmergencyResponse) error {
	return s.ServerStream.SendMsg(response)
}
