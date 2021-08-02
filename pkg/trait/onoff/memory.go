package onoff

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/memory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type MemoryDevice struct {
	traits.UnimplementedOnOffApiServer
	onOff *memory.Resource
}

func NewMemoryDevice(initialState traits.OnOff_State) *MemoryDevice {
	return &MemoryDevice{
		onOff: memory.NewResource(memory.WithInitialValue(&traits.OnOff{
			State: initialState,
		})),
	}
}

func (t *MemoryDevice) Register(server *grpc.Server) {
	traits.RegisterOnOffApiServer(server, t)
}

func (t *MemoryDevice) GetOnOff(_ context.Context, _ *traits.GetOnOffRequest) (*traits.OnOff, error) {
	return t.onOff.Get().(*traits.OnOff), nil
}

func (t *MemoryDevice) UpdateOnOff(_ context.Context, request *traits.UpdateOnOffRequest) (*traits.OnOff, error) {
	res, err := t.onOff.Set(request.OnOff)
	return res.(*traits.OnOff), err
}

func (t *MemoryDevice) PullOnOff(request *traits.PullOnOffRequest, server traits.OnOffApi_PullOnOffServer) error {
	changes, done := t.onOff.OnUpdate(server.Context())
	defer done()

	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case event := <-changes:
			change := &traits.PullOnOffResponse_Change{
				Name:       request.Name,
				OnOff:      event.Value.(*traits.OnOff),
				ChangeTime: event.ChangeTime,
			}
			err := server.Send(&traits.PullOnOffResponse{
				Changes: []*traits.PullOnOffResponse_Change{change},
			})
			if err != nil {
				return err
			}
		}
	}
}
