package wrap

import (
	"context"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	"git.vanti.co.uk/smartcore/sc-api/go/types"
	"google.golang.org/grpc"
)

// SpeakerClientFromServer adapts a SpeakerServer and presents it as a SpeakerClient
func SpeakerClientFromServer(server traits.SpeakerServer) traits.SpeakerClient {
	return &speakerServerClient{server}
}

type speakerServerClient struct {
	server traits.SpeakerServer
}

// compile time check that we implement the interface we need
var _ traits.SpeakerClient = &speakerServerClient{}

func (b *speakerServerClient) GetVolume(ctx context.Context, in *traits.GetSpeakerVolumeRequest, opts ...grpc.CallOption) (*types.Volume, error) {
	return b.server.GetVolume(ctx, in)
}

func (b *speakerServerClient) UpdateVolume(ctx context.Context, in *traits.UpdateSpeakerVolumeRequest, opts ...grpc.CallOption) (*types.Volume, error) {
	return b.server.UpdateVolume(ctx, in)
}

func (b *speakerServerClient) PullVolume(ctx context.Context, in *traits.PullSpeakerVolumeRequest, opts ...grpc.CallOption) (traits.Speaker_PullVolumeClient, error) {
	stream := newClientServerStream(ctx)
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
