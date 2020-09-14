package memory

import (
	"context"
	"sync"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	"git.vanti.co.uk/smartcore/sc-api/go/types"
	"github.com/olebedev/emitter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type OnOff struct {
	traits.UnimplementedOnOffServer
	mu             sync.Mutex // guards the lastKnownState
	lastKnownState types.OnOffState

	bus *emitter.Emitter
}

func NewOnOff(initialState types.OnOffState) *OnOff {
	return &OnOff{
		lastKnownState: initialState,
		bus:            &emitter.Emitter{},
	}
}

func (o *OnOff) Register(server *grpc.Server) {
	traits.RegisterOnOffServer(server, o)
}

func (o *OnOff) On(ctx context.Context, request *traits.OnRequest) (*traits.OnReply, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.lastKnownState != types.OnOffState_ON {
		o.lastKnownState = types.OnOffState_ON
		o.bus.Emit("change:on-off", o.lastKnownState)
	}
	return &traits.OnReply{}, nil
}

func (o *OnOff) Off(ctx context.Context, request *traits.OffRequest) (*traits.OffReply, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.lastKnownState != types.OnOffState_OFF {
		o.lastKnownState = types.OnOffState_OFF
		o.bus.Emit("change:on-off", o.lastKnownState)
	}
	return &traits.OffReply{}, nil
}

func (o *OnOff) GetOnOffState(ctx context.Context, request *traits.GetOnOffStateRequest) (*traits.GetOnOffStateResponse, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return &traits.GetOnOffStateResponse{State: o.lastKnownState}, nil
}

func (o *OnOff) Pull(request *traits.OnOffPullRequest, server traits.OnOff_PullServer) error {
	changes := o.bus.On("change:on-off")
	defer o.bus.Off("change:on-off", changes)

	name := request.Name
	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case change, ok := <-changes:
			if !ok {
				return nil
			}
			state := change.Args[0].(types.OnOffState)
			if err := server.Send(&traits.OnOffPullResponse{Changes: []*traits.OnOffChange{
				{Name: name, State: state, CreateTime: serverTimestamp()},
			}}); err != nil {
				return err
			}
		}
	}
}
