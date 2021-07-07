package router

import (
	"context"
	"io"
	"sync"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SpeakerApiRouter is a SpeakerApiServer that allows routing named requests to specific SpeakerApiClients
type SpeakerApiRouter struct {
	traits.UnimplementedSpeakerApiServer

	mu       sync.Mutex
	registry map[string]traits.SpeakerApiClient
	// Factory can be used to dynamically create api clients if requests come in for devices we haven't seen.
	Factory func(string) (traits.SpeakerApiClient, error)
}

// compile time check that we implement the interface we need
var _ traits.SpeakerApiServer = &SpeakerApiRouter{}

func NewSpeakerApiRouter() *SpeakerApiRouter {
	return &SpeakerApiRouter{
		registry: make(map[string]traits.SpeakerApiClient),
	}
}

func (r *SpeakerApiRouter) Register(server *grpc.Server) {
	traits.RegisterSpeakerApiServer(server, r)
}

func (r *SpeakerApiRouter) Add(name string, client traits.SpeakerApiClient) traits.SpeakerApiClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	r.registry[name] = client
	return old
}

func (r *SpeakerApiRouter) Remove(name string) traits.SpeakerApiClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	delete(r.registry, name)
	return old
}

func (r *SpeakerApiRouter) Has(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, exists := r.registry[name]
	return exists
}

func (r *SpeakerApiRouter) Get(name string) (traits.SpeakerApiClient, error) {
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

func (r *SpeakerApiRouter) GetVolume(ctx context.Context, request *traits.GetSpeakerVolumeRequest) (*types.AudioLevel, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.GetVolume(ctx, request)
}

func (r *SpeakerApiRouter) UpdateVolume(ctx context.Context, request *traits.UpdateSpeakerVolumeRequest) (*types.AudioLevel, error) {
	child, err := r.Get(request.Name)
	if err != nil {
		return nil, err
	}

	return child.UpdateVolume(ctx, request)
}

func (r *SpeakerApiRouter) PullVolume(request *traits.PullSpeakerVolumeRequest, server traits.SpeakerApi_PullVolumeServer) error {
	child, err := r.Get(request.Name)
	if err != nil {
		return err
	}

	// so we can cancel our forwarding request if we can't send responses to our caller
	reqCtx, reqDone := context.WithCancel(server.Context())
	// issue the request
	stream, err := child.PullVolume(reqCtx, request)
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

		var msg *traits.PullSpeakerVolumeResponse
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
