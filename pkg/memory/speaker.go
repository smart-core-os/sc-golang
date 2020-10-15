package memory

import (
	"context"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	"git.vanti.co.uk/smartcore/sc-api/go/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type Speaker struct {
	traits.UnimplementedSpeakerServer
	volume *Resource
}

func NewSpeaker(initialState *types.Volume) *Speaker {
	return &Speaker{
		volume: NewResource(WithInitialValue(initialState)),
	}
}

func (s *Speaker) Register(server *grpc.Server) {
	traits.RegisterSpeakerServer(server, s)
}

func (s *Speaker) GetVolume(ctx context.Context, request *traits.GetSpeakerVolumeRequest) (*types.Volume, error) {
	return s.volume.Get().(*types.Volume), nil
}

func (s *Speaker) UpdateVolume(ctx context.Context, request *traits.UpdateSpeakerVolumeRequest) (*types.Volume, error) {
	newValue, err := s.volume.UpdateDelta(request.Volume, request.UpdateMask, func(old, change proto.Message) {
		if request.Delta {
			val := old.(*types.Volume)
			delta := change.(*types.Volume)
			if val.GetGain().GetValue() != nil && delta.GetGain().GetValue() != nil {
				delta.Gain.Value.Value += val.Gain.Value.Value
			}
		}
	})
	if err != nil {
		return nil, err
	}
	return newValue.(*types.Volume), nil
}

func (s *Speaker) PullVolume(request *traits.PullSpeakerVolumeRequest, server traits.Speaker_PullVolumeServer) error {
	changes, done := s.volume.OnUpdate(server.Context())
	defer done()

	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case change := <-changes:
			typedChange := &types.VolumeChange{
				Name:       request.Name,
				Volume:     change.Value.(*types.Volume),
				ChangeTime: change.ChangeTime,
			}
			err := server.Send(&traits.PullSpeakerVolumeResponse{
				Changes: []*types.VolumeChange{typedChange},
			})
			if err != nil {
				return err
			}
		}
	}
}
