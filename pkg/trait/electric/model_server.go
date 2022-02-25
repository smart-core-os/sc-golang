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

	"github.com/smart-core-os/sc-golang/pkg/resource"
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

func (s *ModelServer) Unwrap() interface{} {
	return s.model
}

func (s *ModelServer) Register(server *grpc.Server) {
	traits.RegisterElectricApiServer(server, s)
	RegisterMemorySettingsApiServer(server, s)
}

func (s *ModelServer) GetDemand(_ context.Context, request *traits.GetDemandRequest) (*traits.ElectricDemand, error) {
	return s.model.Demand(resource.WithGetMask(request.ReadMask)), nil
}

func (s *ModelServer) PullDemand(request *traits.PullDemandRequest, server traits.ElectricApi_PullDemandServer) error {
	updates, done := s.model.PullDemand(server.Context(), request.ReadMask)
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

func (s *ModelServer) GetActiveMode(_ context.Context, request *traits.GetActiveModeRequest) (*traits.ElectricMode, error) {
	return s.model.ActiveMode(resource.WithGetMask(request.ReadMask)), nil
}

func (s *ModelServer) UpdateActiveMode(_ context.Context, request *traits.UpdateActiveModeRequest) (*traits.ElectricMode, error) {
	mode := request.GetActiveMode()
	// hydrate the mode using the list of known modes (by id)
	id := mode.GetId()
	if id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Id should be present during update")
	}

	return s.model.ChangeActiveMode(id)
}

func (s *ModelServer) ClearActiveMode(_ context.Context, _ *traits.ClearActiveModeRequest) (*traits.ElectricMode, error) {
	return s.model.ChangeToNormalMode()
}

func (s *ModelServer) PullActiveMode(request *traits.PullActiveModeRequest, server traits.ElectricApi_PullActiveModeServer) error {
	updates, done := s.model.PullActiveMode(server.Context(), request.ReadMask)
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

func (s *ModelServer) ListModes(_ context.Context, request *traits.ListModesRequest) (*traits.ListModesResponse, error) {
	pageToken := &types.PageToken{}
	if err := decodePageToken(request.PageToken, pageToken); err != nil {
		return nil, err
	}

	lastKey := pageToken.GetLastResourceName() // the key() of the last item we sent
	pageSize := capPageSize(int(request.GetPageSize()))

	sortedModes := s.model.Modes(request.ReadMask)
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

func (s *ModelServer) PullModes(request *traits.PullModesRequest, server traits.ElectricApi_PullModesServer) error {
	ctx, done := context.WithCancel(server.Context())
	changes := s.model.PullModes(ctx, request.ReadMask)
	defer done()

	// watch for changes to the modes list and emit when one matches our query
	for {
		select {
		case <-ctx.Done():
			return status.FromContextError(ctx.Err()).Err()
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
