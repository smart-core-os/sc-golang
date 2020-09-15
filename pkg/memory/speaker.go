package memory

import (
	"context"
	"sync"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	"git.vanti.co.uk/smartcore/sc-api/go/types"
	fieldMaskUtils "github.com/mennanov/fieldmask-utils"
	"github.com/olebedev/emitter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"git.vanti.co.uk/smartcore/sc-golang/pkg/masks"
)

// SpeakerVolumeWritableFields are all fields that can be updated
var SpeakerVolumeWritableFields *fieldmaskpb.FieldMask

type Speaker struct {
	traits.UnimplementedSpeakerServer
	volumeMu sync.RWMutex // guards volume
	volume   *types.Volume

	bus *emitter.Emitter
}

func NewSpeaker(initialState *types.Volume) *Speaker {
	return &Speaker{
		volume: initialState,
		bus:    &emitter.Emitter{},
	}
}

func (s *Speaker) Register(server *grpc.Server) {
	traits.RegisterSpeakerServer(server, s)
}

func (s *Speaker) GetVolume(ctx context.Context, request *traits.GetSpeakerVolumeRequest) (*types.Volume, error) {
	s.volumeMu.Lock()
	defer s.volumeMu.Unlock()
	return s.volume, nil
}

func (s *Speaker) UpdateVolume(ctx context.Context, request *traits.UpdateSpeakerVolumeRequest) (*types.Volume, error) {
	// make sure they can only write the fields we want
	mask, err := masks.ValidWritableMask(SpeakerVolumeWritableFields, request.UpdateMask, request.Volume)
	if err != nil {
		return nil, err
	}

	_, newValue, err := applyChange(
		&s.volumeMu,
		func() (proto.Message, bool) {
			return s.volume, true
		},
		func(message proto.Message) error {
			if mask != nil {
				// apply only selected fields
				return fieldMaskUtils.StructToStruct(mask, request.Volume, message.(*types.Volume))
			} else {
				// replace the booking data
				proto.Reset(message)
				proto.Merge(message, request.Volume)
				return nil
			}
		},
		func(message proto.Message) {
			s.volume = message.(*types.Volume)
		},
	)

	if err != nil {
		return nil, err
	}

	s.bus.Emit("update:volume", &types.VolumeChange{
		Name:       request.Name,
		Volume:     newValue.(*types.Volume),
		ChangeTime: serverTimestamp(),
	})

	return newValue.(*types.Volume), err
}

func (s *Speaker) PullVolume(request *traits.PullSpeakerVolumeRequest, server traits.Speaker_PullVolumeServer) error {
	changes := s.bus.On("update:volume")
	defer s.bus.Off("update:volume", changes)

	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case event := <-changes:
			change := event.Args[0].(*types.VolumeChange)
			err := server.Send(&traits.PullSpeakerVolumeResponse{
				Changes: []*types.VolumeChange{change},
			})
			if err != nil {
				return err
			}
		}
	}
}
