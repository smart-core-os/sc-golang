package router

import (
	"context"
	"io"
	"sync"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LightApiRouter is a LightApiServer that allows routing named requests to specific LightApiClients
type LightApiRouter struct {
	traits.UnimplementedLightApiServer

	mu       sync.Mutex
	registry map[string]traits.LightApiClient
	// Factory can be used to dynamically create api clients if requests come in for devices we haven't seen.
	Factory func(string) (traits.LightApiClient, error)
}

// compile time check that we implement the interface we need
var _ traits.LightApiServer = &LightApiRouter{}

func NewLightApiRouter() *LightApiRouter {
	return &LightApiRouter{
		registry: make(map[string]traits.LightApiClient),
	}
}

func (r *LightApiRouter) Register(server *grpc.Server) {
	traits.RegisterLightApiServer(server, r)
}

func (r *LightApiRouter) Add(name string, client traits.LightApiClient) traits.LightApiClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	r.registry[name] = client
	return old
}

func (r *LightApiRouter) Remove(name string) traits.LightApiClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	delete(r.registry, name)
	return old
}

func (r *LightApiRouter) Has(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, exists := r.registry[name]
	return exists
}

func (r *LightApiRouter) Get(name string) (traits.LightApiClient, error) {
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

func (r *LightApiRouter) UpdateBrightness(ctx context.Context, request *traits.UpdateBrightnessRequest) (*traits.Brightness, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.UpdateBrightness(ctx, request)
}

func (r *LightApiRouter) GetBrightness(ctx context.Context, request *traits.GetBrightnessRequest) (*traits.Brightness, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.GetBrightness(ctx, request)
}

func (r *LightApiRouter) PullBrightness(request *traits.PullBrightnessRequest, server traits.LightApi_PullBrightnessServer) error {
	child, err := r.Get(request.Name)
	if err != nil {
		return err
	}

	// so we can cancel our forwarding request if we can't send responses to our caller
	reqCtx, reqDone := context.WithCancel(server.Context())
	// issue the request
	stream, err := child.PullBrightness(reqCtx, request)
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

		var msg *traits.PullBrightnessResponse
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
