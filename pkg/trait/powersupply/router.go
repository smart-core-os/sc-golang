package powersupply

import (
	"context"
	"io"
	"sync"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Router is a PowerSupplyApiServer that allows routing named requests to specific PowerSupplyApiClients
type Router struct {
	traits.UnimplementedPowerSupplyApiServer

	mu       sync.Mutex
	registry map[string]traits.PowerSupplyApiClient
	// Factory can be used to dynamically create api clients if requests come in for devices we haven't seen.
	Factory func(string) (traits.PowerSupplyApiClient, error)
}

// compile time check that we implement the interface we need
var _ traits.PowerSupplyApiServer = (*Router)(nil)

func NewRouter() *Router {
	return &Router{
		registry: make(map[string]traits.PowerSupplyApiClient),
	}
}

func (r *Router) Register(server *grpc.Server) {
	traits.RegisterPowerSupplyApiServer(server, r)
}

func (r *Router) Add(name string, client traits.PowerSupplyApiClient) traits.PowerSupplyApiClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	r.registry[name] = client
	return old
}

func (r *Router) Remove(name string) traits.PowerSupplyApiClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	delete(r.registry, name)
	return old
}

func (r *Router) Has(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, exists := r.registry[name]
	return exists
}

func (r *Router) Get(name string) (traits.PowerSupplyApiClient, error) {
	r.mu.Lock()
	child, exists := r.registry[name]
	defer r.mu.Unlock()
	if !exists {
		if r.Factory != nil {
			child, err := r.Factory(name)
			if err != nil {
				return nil, err
			}
			r.registry[name] = child
			return child, nil
		}
		return nil, status.Error(codes.NotFound, name)
	}
	return child, nil
}

func (r *Router) GetPowerCapacity(ctx context.Context, request *traits.GetPowerCapacityRequest) (*traits.PowerCapacity, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.GetPowerCapacity(ctx, request)
}

func (r *Router) PullPowerCapacity(request *traits.PullPowerCapacityRequest, server traits.PowerSupplyApi_PullPowerCapacityServer) error {
	child, err := r.Get(request.Name)
	if err != nil {
		return err
	}

	// so we can cancel our forwarding request if we can't send responses to our caller
	reqCtx, reqDone := context.WithCancel(server.Context())
	// issue the request
	stream, err := child.PullPowerCapacity(reqCtx, request)
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

		var msg *traits.PullPowerCapacityResponse
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

func (r *Router) CreateDrawNotification(ctx context.Context, request *traits.CreateDrawNotificationRequest) (*traits.DrawNotification, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.CreateDrawNotification(ctx, request)
}

func (r *Router) UpdateDrawNotification(ctx context.Context, request *traits.UpdateDrawNotificationRequest) (*traits.DrawNotification, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.UpdateDrawNotification(ctx, request)
}

func (r *Router) DeleteDrawNotification(ctx context.Context, request *traits.DeleteDrawNotificationRequest) (*emptypb.Empty, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.DeleteDrawNotification(ctx, request)
}
