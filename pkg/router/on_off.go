package router

import (
	"context"
	"io"
	"sync"

	"git.vanti.co.uk/smartcore/sc-api/go/traits"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OnOffApiRouter is a OnOffApiServer that allows routing named requests to specific OnOffApiClients
type OnOffApiRouter struct {
	traits.UnimplementedOnOffApiServer

	mu       sync.Mutex
	registry map[string]traits.OnOffApiClient
	// Factory can be used to dynamically create api clients if requests come in for devices we haven't seen.
	Factory func(string) (traits.OnOffApiClient, error)
}

// compile time check that we implement the interface we need
var _ traits.OnOffApiServer = &OnOffApiRouter{}

func NewOnOffApiRouter() *OnOffApiRouter {
	return &OnOffApiRouter{
		registry: make(map[string]traits.OnOffApiClient),
	}
}

func (r *OnOffApiRouter) Register(server *grpc.Server) {
	traits.RegisterOnOffApiServer(server, r)
}

func (r *OnOffApiRouter) Add(name string, client traits.OnOffApiClient) traits.OnOffApiClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	r.registry[name] = client
	return old
}

func (r *OnOffApiRouter) Remove(name string) traits.OnOffApiClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	delete(r.registry, name)
	return old
}

func (r *OnOffApiRouter) Has(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, exists := r.registry[name]
	return exists
}

func (r *OnOffApiRouter) Get(name string) (traits.OnOffApiClient, error) {
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

func (r *OnOffApiRouter) GetOnOff(ctx context.Context, request *traits.GetOnOffRequest) (*traits.OnOff, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.GetOnOff(ctx, request)
}

func (r *OnOffApiRouter) UpdateOnOff(ctx context.Context, request *traits.UpdateOnOffRequest) (*traits.OnOff, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.UpdateOnOff(ctx, request)
}

func (r *OnOffApiRouter) PullOnOff(request *traits.PullOnOffRequest, server traits.OnOffApi_PullOnOffServer) error {
	child, err := r.Get(request.Name)
	if err != nil {
		return err
	}

	// so we can cancel our forwarding request if we can't send responses to our caller
	reqCtx, reqDone := context.WithCancel(server.Context())
	// issue the request
	stream, err := child.PullOnOff(reqCtx, request)
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

		var msg *traits.PullOnOffResponse
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
