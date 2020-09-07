package router

import (
	"context"
	"io"
	"sync"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OccupancyApiRouter is a OccupancyApiServer that allows routing named requests to specific OccupancyApiClients
type OccupancyApiRouter struct {
	traits.UnimplementedOccupancyApiServer

	mu       sync.Mutex
	registry map[string]traits.OccupancyApiClient
}

// compile time check that we implement the interface we need
var _ traits.OccupancyApiServer = &OccupancyApiRouter{}

func NewOccupancyApiRouter() *OccupancyApiRouter {
	return &OccupancyApiRouter{
		registry: make(map[string]traits.OccupancyApiClient),
	}
}

func (r *OccupancyApiRouter) Register(server *grpc.Server) {
	traits.RegisterOccupancyApiServer(server, r)
}

func (r *OccupancyApiRouter) Add(name string, client traits.OccupancyApiClient) traits.OccupancyApiClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	r.registry[name] = client
	return old
}

func (r *OccupancyApiRouter) Remove(name string) traits.OccupancyApiClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	delete(r.registry, name)
	return old
}

func (r *OccupancyApiRouter) Has(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, exists := r.registry[name]
	return exists
}

func (r *OccupancyApiRouter) GetOccupancy(ctx context.Context, request *traits.GetOccupancyRequest) (*traits.Occupancy, error) {
	r.mu.Lock()
	child, exists := r.registry[request.Name]
	r.mu.Unlock()
	if !exists {
		return nil, status.Error(codes.NotFound, request.Name)
	}

	return child.GetOccupancy(ctx, request)
}

func (r *OccupancyApiRouter) PullOccupancy(request *traits.PullOccupancyRequest, server traits.OccupancyApi_PullOccupancyServer) error {
	r.mu.Lock()
	child, exists := r.registry[request.Name]
	r.mu.Unlock()
	if !exists {
		return status.Error(codes.NotFound, request.Name)
	}

	// so we can cancel our forwarding request if we can't send responses to our caller
	reqCtx, reqDone := context.WithCancel(server.Context())
	// issue the request
	stream, err := child.PullOccupancy(reqCtx, request)
	if err != nil {
		return err
	}

	// send the stream header
	header, err := stream.Header()
	if err != nil {
		return err
	}
	if err = server.SendHeader(header); err != nil {
		return err
	}

	// send all the messages
	// false means the error is from the child, true means the error is from the caller
	var callerError bool
	for {
		// Impl note: we could improve throughput here by issuing the Recv and Send in different goroutines, but we're doing
		// it synchronously until we have a need to change the behaviour

		var msg *traits.PullOccupancyResponse
		msg, err = stream.Recv()
		if err != nil {
			break
		}

		err = server.Send(msg)
		if err != nil {
			callerError = true
			break
		}
	}

	// err is guaranteed to be non-nil as it's the only way to exit the loop
	if callerError {
		// cancel the request
		reqDone()
		return err
	} else {
		if trailer := stream.Trailer(); trailer != nil {
			server.SetTrailer(trailer)
		}
		if err == io.EOF {
			return nil
		}
		return err
	}
}

func (r *OccupancyApiRouter) CreateOccupancyOverride(ctx context.Context, request *traits.CreateOccupancyOverrideRequest) (*traits.OccupancyOverride, error) {
	r.mu.Lock()
	child, exists := r.registry[request.Name]
	r.mu.Unlock()
	if !exists {
		return nil, status.Error(codes.NotFound, request.Name)
	}

	return child.CreateOccupancyOverride(ctx, request)
}

func (r *OccupancyApiRouter) UpdateOccupancyOverride(ctx context.Context, request *traits.UpdateOccupancyOverrideRequest) (*traits.OccupancyOverride, error) {
	r.mu.Lock()
	child, exists := r.registry[request.Name]
	r.mu.Unlock()
	if !exists {
		return nil, status.Error(codes.NotFound, request.Name)
	}

	return child.UpdateOccupancyOverride(ctx, request)
}

func (r *OccupancyApiRouter) DeleteOccupancyOverride(ctx context.Context, request *traits.DeleteOccupancyOverrideRequest) (*empty.Empty, error) {
	r.mu.Lock()
	child, exists := r.registry[request.DeviceName]
	r.mu.Unlock()
	if !exists {
		return nil, status.Error(codes.NotFound, request.DeviceName)
	}

	return child.DeleteOccupancyOverride(ctx, request)
}

func (r *OccupancyApiRouter) GetOccupancyOverride(ctx context.Context, request *traits.GetOccupancyOverrideRequest) (*traits.OccupancyOverride, error) {
	r.mu.Lock()
	child, exists := r.registry[request.DeviceName]
	r.mu.Unlock()
	if !exists {
		return nil, status.Error(codes.NotFound, request.DeviceName)
	}

	return child.GetOccupancyOverride(ctx, request)
}

func (r *OccupancyApiRouter) ListOccupancyOverrides(ctx context.Context, request *traits.ListOccupancyOverridesRequest) (*traits.ListOccupancyOverridesResponse, error) {
	r.mu.Lock()
	child, exists := r.registry[request.Name]
	r.mu.Unlock()
	if !exists {
		return nil, status.Error(codes.NotFound, request.Name)
	}

	return child.ListOccupancyOverrides(ctx, request)
}
