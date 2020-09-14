package router

import (
	"context"
	"io"
	"sync"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OnOffRouter is a OnOffServer that allows routing named requests to specific OnOffClients
type OnOffRouter struct {
	traits.UnimplementedOnOffServer

	mu       sync.Mutex
	registry map[string]traits.OnOffClient
	// Factory can be used to dynamically create api clients if requests come in for devices we haven't seen.
	Factory func(string) (traits.OnOffClient, error)
}

// compile time check that we implement the interface we need
var _ traits.OnOffServer = &OnOffRouter{}

func NewOnOffRouter() *OnOffRouter {
	return &OnOffRouter{
		registry: make(map[string]traits.OnOffClient),
	}
}

func (r *OnOffRouter) Register(server *grpc.Server) {
	traits.RegisterOnOffServer(server, r)
}

func (r *OnOffRouter) Add(name string, client traits.OnOffClient) traits.OnOffClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	r.registry[name] = client
	return old
}

func (r *OnOffRouter) Remove(name string) traits.OnOffClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	delete(r.registry, name)
	return old
}

func (r *OnOffRouter) Has(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, exists := r.registry[name]
	return exists
}

func (r *OnOffRouter) Get(name string) (traits.OnOffClient, error) {
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

func (r *OnOffRouter) On(ctx context.Context, request *traits.OnRequest) (*traits.OnReply, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.On(ctx, request)
}

func (r *OnOffRouter) Off(ctx context.Context, request *traits.OffRequest) (*traits.OffReply, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.Off(ctx, request)
}

func (r *OnOffRouter) GetOnOffState(ctx context.Context, request *traits.GetOnOffStateRequest) (*traits.GetOnOffStateResponse, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.GetOnOffState(ctx, request)
}

func (r *OnOffRouter) Pull(request *traits.OnOffPullRequest, server traits.OnOff_PullServer) error {
	child, err := r.Get(request.Name)
	if err != nil {
		return err
	}

	// so we can cancel our forwarding request if we can't send responses to our caller
	reqCtx, reqDone := context.WithCancel(server.Context())
	// issue the request
	stream, err := child.Pull(reqCtx, request)
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

		var msg *traits.OnOffPullResponse
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
