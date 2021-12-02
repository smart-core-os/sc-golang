package electric

import (
	"context"
	"io"
	"sync"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Router is a ElectricApiServer that allows routing named requests to specific ElectricApiClients
type Router struct {
	traits.UnimplementedElectricApiServer

	mu       sync.Mutex
	registry map[string]traits.ElectricApiClient
	// Factory can be used to dynamically create api clients if requests come in for devices we haven't seen.
	Factory func(string) (traits.ElectricApiClient, error)
}

// compile time check that we implement the interface we need
var _ traits.ElectricApiServer = (*Router)(nil)

func NewRouter() *Router {
	return &Router{
		registry: make(map[string]traits.ElectricApiClient),
	}
}

func (r *Router) Register(server *grpc.Server) {
	traits.RegisterElectricApiServer(server, r)
}

func (r *Router) Add(name string, client traits.ElectricApiClient) traits.ElectricApiClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	r.registry[name] = client
	return old
}

func (r *Router) Remove(name string) traits.ElectricApiClient {
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

func (r *Router) Get(name string) (traits.ElectricApiClient, error) {
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

func (r *Router) GetDemand(ctx context.Context, request *traits.GetDemandRequest) (*traits.ElectricDemand, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.GetDemand(ctx, request)
}

func (r *Router) PullDemand(request *traits.PullDemandRequest, server traits.ElectricApi_PullDemandServer) error {
	child, err := r.Get(request.Name)
	if err != nil {
		return err
	}

	// so we can cancel our forwarding request if we can't send responses to our caller
	reqCtx, reqDone := context.WithCancel(server.Context())
	// issue the request
	stream, err := child.PullDemand(reqCtx, request)
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

		var msg *traits.PullDemandResponse
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

func (r *Router) GetActiveMode(ctx context.Context, request *traits.GetActiveModeRequest) (*traits.ElectricMode, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.GetActiveMode(ctx, request)
}

func (r *Router) UpdateActiveMode(ctx context.Context, request *traits.UpdateActiveModeRequest) (*traits.ElectricMode, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.UpdateActiveMode(ctx, request)
}

func (r *Router) ClearActiveMode(ctx context.Context, request *traits.ClearActiveModeRequest) (*traits.ElectricMode, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.ClearActiveMode(ctx, request)
}

func (r *Router) PullActiveMode(request *traits.PullActiveModeRequest, server traits.ElectricApi_PullActiveModeServer) error {
	child, err := r.Get(request.Name)
	if err != nil {
		return err
	}

	// so we can cancel our forwarding request if we can't send responses to our caller
	reqCtx, reqDone := context.WithCancel(server.Context())
	// issue the request
	stream, err := child.PullActiveMode(reqCtx, request)
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

		var msg *traits.PullActiveModeResponse
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

func (r *Router) ListModes(ctx context.Context, request *traits.ListModesRequest) (*traits.ListModesResponse, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.ListModes(ctx, request)
}

func (r *Router) PullModes(request *traits.PullModesRequest, server traits.ElectricApi_PullModesServer) error {
	child, err := r.Get(request.Name)
	if err != nil {
		return err
	}

	// so we can cancel our forwarding request if we can't send responses to our caller
	reqCtx, reqDone := context.WithCancel(server.Context())
	// issue the request
	stream, err := child.PullModes(reqCtx, request)
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

		var msg *traits.PullModesResponse
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
