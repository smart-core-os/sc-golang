package wrap

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"google.golang.org/grpc"
)

// SpeakerApiServer adapts a traits.SpeakerApiServer and presents it as a traits.SpeakerApiClient
func SpeakerApiServer(server traits.SpeakerApiServer) traits.SpeakerApiClient {
	return &speakerApiServerClient{server}
}

type speakerApiServerClient struct {
	server traits.SpeakerApiServer
}

// compile time check that we implement the interface we need
var _ traits.SpeakerApiClient = &speakerApiServerClient{}

func (b *speakerApiServerClient) GetVolume(ctx context.Context, in *traits.GetSpeakerVolumeRequest, _ ...grpc.CallOption) (*types.AudioLevel, error) {
	return b.server.GetVolume(ctx, in)
}

func (b *speakerApiServerClient) UpdateVolume(ctx context.Context, in *traits.UpdateSpeakerVolumeRequest, _ ...grpc.CallOption) (*types.AudioLevel, error) {
	return b.server.UpdateVolume(ctx, in)
}

func (b *speakerApiServerClient) PullVolume(ctx context.Context, in *traits.PullSpeakerVolumeRequest, _ ...grpc.CallOption) (traits.SpeakerApi_PullVolumeClient, error) {
	stream := NewClientServerStream(ctx)
	server := &speakerPullVolumeServer{stream.Server()}
	client := &speakerPullVolumeClient{stream.Client()}
	go func() {
		err := b.server.PullVolume(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type speakerPullVolumeClient struct {
	grpc.ClientStream
}

func (c *speakerPullVolumeClient) Recv() (*traits.PullSpeakerVolumeResponse, error) {
	m := new(traits.PullSpeakerVolumeResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type speakerPullVolumeServer struct {
	grpc.ServerStream
}

func (s *speakerPullVolumeServer) Send(response *traits.PullSpeakerVolumeResponse) error {
	return s.ServerStream.SendMsg(response)
}
