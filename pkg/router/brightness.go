package router

import (
	"context"
	"fmt"
	"io"
	"sync"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BrightnessApiRouter is a BrightnessApiServer that allows routing named requests to specific BrightnessApiClients
type BrightnessApiRouter struct {
	traits.UnimplementedBrightnessApiServer

	mu       sync.Mutex
	registry map[string]traits.BrightnessApiClient
}

// compile time check that we implement the interface we need
var _ traits.BrightnessApiServer = &BrightnessApiRouter{}

func (b *BrightnessApiRouter) Add(name string, client traits.BrightnessApiClient) traits.BrightnessApiClient {
	b.mu.Lock()
	defer b.mu.Unlock()
	old := b.registry[name]
	b.registry[name] = client
	return old
}

func (b *BrightnessApiRouter) Remove(name string) traits.BrightnessApiClient {
	b.mu.Lock()
	defer b.mu.Unlock()
	old := b.registry[name]
	delete(b.registry, name)
	return old
}

func (b *BrightnessApiRouter) Has(name string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, exists := b.registry[name]
	return exists
}

func (b *BrightnessApiRouter) UpdateRangeValue(ctx context.Context, request *traits.UpdateBrightnessRequest) (*traits.Brightness, error) {
	b.mu.Lock()
	child, exists := b.registry[request.Name]
	b.mu.Unlock()
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("device: %v", request.Name))
	}

	return child.UpdateRangeValue(ctx, request)
}

func (b *BrightnessApiRouter) GetBrightness(ctx context.Context, request *traits.GetBrightnessRequest) (*traits.Brightness, error) {
	b.mu.Lock()
	child, exists := b.registry[request.Name]
	b.mu.Unlock()
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("device: %v", request.Name))
	}

	return child.GetBrightness(ctx, request)
}

func (b *BrightnessApiRouter) PullBrightness(request *traits.PullBrightnessRequest, server traits.BrightnessApi_PullBrightnessServer) error {
	b.mu.Lock()
	child, exists := b.registry[request.Name]
	b.mu.Unlock()
	if !exists {
		return status.Error(codes.NotFound, fmt.Sprintf("device: %v", request.Name))
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
