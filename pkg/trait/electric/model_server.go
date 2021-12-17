package electric

import (
	"context"
	"sort"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ModelServer is an implementation of ElectricApiServer and MemorySettingsApiServer backed by a Model.
type ModelServer struct {
	traits.UnimplementedElectricApiServer
	UnimplementedMemorySettingsApiServer

	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{
		model: model,
	}
}

func (d *ModelServer) Register(server *grpc.Server) {
	traits.RegisterElectricApiServer(server, d)
	RegisterMemorySettingsApiServer(server, d)
}

func (d *ModelServer) GetDemand(_ context.Context, request *traits.GetDemandRequest) (*traits.ElectricDemand, error) {
	return d.model.Demand(request.ReadMask), nil
}

func (d *ModelServer) PullDemand(request *traits.PullDemandRequest, server traits.ElectricApi_PullDemandServer) error {
	updates, done := d.model.PullDemand(server.Context(), request.ReadMask)
	defer done()

	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case update := <-updates:
			change := &traits.PullDemandResponse_Change{
				Name:       request.Name,
				ChangeTime: timestamppb.New(update.ChangeTime),
				Demand:     update.Value,
			}

			err := server.Send(&traits.PullDemandResponse{
				Changes: []*traits.PullDemandResponse_Change{change},
			})
			if err != nil {
				return err
			}
		}
	}
}

func (d *ModelServer) GetActiveMode(_ context.Context, request *traits.GetActiveModeRequest) (*traits.ElectricMode, error) {
	return d.model.ActiveMode(request.ReadMask), nil
}

func (d *ModelServer) UpdateActiveMode(_ context.Context, request *traits.UpdateActiveModeRequest) (*traits.ElectricMode, error) {
	mode := request.GetActiveMode()
	// hydrate the mode using the list of known modes (by id)
	id := mode.GetId()
	if id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Id should be present during update")
	}

	return d.model.ChangeActiveMode(id)
}

func (d *ModelServer) ClearActiveMode(_ context.Context, _ *traits.ClearActiveModeRequest) (*traits.ElectricMode, error) {
	return d.model.ChangeToNormalMode()
}

func (d *ModelServer) PullActiveMode(request *traits.PullActiveModeRequest, server traits.ElectricApi_PullActiveModeServer) error {
	updates, done := d.model.PullActiveMode(server.Context(), request.ReadMask)
	defer done()

	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case event := <-updates:
			change := &traits.PullActiveModeResponse_Change{
				Name:       request.Name,
				ActiveMode: event.ActiveMode,
				ChangeTime: timestamppb.New(event.ChangeTime),
			}
			err := server.Send(&traits.PullActiveModeResponse{
				Changes: []*traits.PullActiveModeResponse_Change{change},
			})
			if err != nil {
				return err
			}
		}
	}
}

func (d *ModelServer) ListModes(_ context.Context, request *traits.ListModesRequest) (*traits.ListModesResponse, error) {
	pageToken := &types.PageToken{}
	if err := decodePageToken(request.PageToken, pageToken); err != nil {
		return nil, err
	}

	lastKey := pageToken.GetLastResourceName() // the key() of the last item we sent
	pageSize := capPageSize(int(request.GetPageSize()))

	sortedModes := d.model.Modes(request.ReadMask)
	nextIndex := 0
	if lastKey != "" {
		nextIndex = sort.Search(len(sortedModes), func(i int) bool {
			return sortedModes[i].Id > lastKey
		})
	}

	result := &traits.ListModesResponse{
		TotalSize: int32(len(sortedModes)),
	}
	upperBound := nextIndex + pageSize
	if upperBound > len(sortedModes) {
		upperBound = len(sortedModes)
		pageToken = nil
	} else {
		pageToken.PageStart = &types.PageToken_LastResourceName{
			LastResourceName: sortedModes[upperBound-1].Id,
		}
	}

	var err error
	result.NextPageToken, err = encodePageToken(pageToken)
	if err != nil {
		return nil, err
	}
	result.Modes = sortedModes[nextIndex:upperBound]
	return result, nil
}

func (d *ModelServer) PullModes(request *traits.PullModesRequest, server traits.ElectricApi_PullModesServer) error {
	changes, done := d.model.PullModes(server.Context(), request.ReadMask)
	defer done()

	// watch for changes to the modes list and emit when one matches our query
	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case change := <-changes:
			err := server.Send(&traits.PullModesResponse{Changes: []*traits.PullModesResponse_Change{
				{
					Name:       request.Name,
					Type:       change.Type,
					NewValue:   change.NewValue,
					OldValue:   change.OldValue,
					ChangeTime: timestamppb.New(change.ChangeTime),
				},
			}})

			if err != nil {
				return err
			}
		}
	}
}
