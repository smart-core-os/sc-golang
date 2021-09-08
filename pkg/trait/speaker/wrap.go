package speaker

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
	"google.golang.org/grpc"
)

// Wrap adapts a traits.SpeakerApiServer and presents it as a traits.SpeakerApiClient
func Wrap(server traits.SpeakerApiServer) traits.SpeakerApiClient {
	return &wrapper{server}
}

type wrapper struct {
	server traits.SpeakerApiServer
}

// compile time check that we implement the interface we need
var _ traits.SpeakerApiClient = (*wrapper)(nil)

func (b *wrapper) GetVolume(ctx context.Context, in *traits.GetSpeakerVolumeRequest, _ ...grpc.CallOption) (*types.AudioLevel, error) {
	return b.server.GetVolume(ctx, in)
}

func (b *wrapper) UpdateVolume(ctx context.Context, in *traits.UpdateSpeakerVolumeRequest, _ ...grpc.CallOption) (*types.AudioLevel, error) {
	return b.server.UpdateVolume(ctx, in)
}

func (b *wrapper) PullVolume(ctx context.Context, in *traits.PullSpeakerVolumeRequest, _ ...grpc.CallOption) (traits.SpeakerApi_PullVolumeClient, error) {
	stream := wrap.NewClientServerStream(ctx)
	server := &pullVolumeServerWrapper{stream.Server()}
	client := &pullVolumeClientWrapper{stream.Client()}
	go func() {
		err := b.server.PullVolume(in, server)
		stream.Close(err)
	}()
	return client, nil
}

type pullVolumeClientWrapper struct {
	grpc.ClientStream
}

func (c *pullVolumeClientWrapper) Recv() (*traits.PullSpeakerVolumeResponse, error) {
	m := new(traits.PullSpeakerVolumeResponse)
	if err := c.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type pullVolumeServerWrapper struct {
	grpc.ServerStream
}

func (s *pullVolumeServerWrapper) Send(response *traits.PullSpeakerVolumeResponse) error {
	return s.ServerStream.SendMsg(response)
}
