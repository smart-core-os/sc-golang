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

// ThermostatRouter is a ThermostatServer that allows routing named requests to specific ThermostatClients
type ThermostatRouter struct {
	traits.UnimplementedThermostatServer

	mu       sync.Mutex
	registry map[string]traits.ThermostatClient
}

// compile time check that we implement the interface we need
var _ traits.ThermostatServer = &ThermostatRouter{}

func (b *ThermostatRouter) Add(name string, client traits.ThermostatClient) traits.ThermostatClient {
	b.mu.Lock()
	defer b.mu.Unlock()
	old := b.registry[name]
	b.registry[name] = client
	return old
}

func (b *ThermostatRouter) Remove(name string) traits.ThermostatClient {
	b.mu.Lock()
	defer b.mu.Unlock()
	old := b.registry[name]
	delete(b.registry, name)
	return old
}

func (b *ThermostatRouter) Has(name string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, exists := b.registry[name]
	return exists
}

func (b *ThermostatRouter) UpdateState(ctx context.Context, request *traits.UpdateThermostatStateRequest) (*traits.ThermostatState, error) {
	b.mu.Lock()
	child, exists := b.registry[request.Name]
	b.mu.Unlock()
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("device: %v", request.Name))
	}

	return child.UpdateState(ctx, request)
}

func (b *ThermostatRouter) GetState(ctx context.Context, request *traits.GetThermostatStateRequest) (*traits.ThermostatState, error) {
	b.mu.Lock()
	child, exists := b.registry[request.Name]
	b.mu.Unlock()
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("device: %v", request.Name))
	}

	return child.GetState(ctx, request)
}

func (b *ThermostatRouter) PullState(request *traits.PullThermostatStateRequest, server traits.Thermostat_PullStateServer) error {
	b.mu.Lock()
	child, exists := b.registry[request.Name]
	b.mu.Unlock()
	if !exists {
		return status.Error(codes.NotFound, fmt.Sprintf("device: %v", request.Name))
	}

	// so we can cancel our forwarding request if we can't send responses to our caller
	reqCtx, reqDone := context.WithCancel(server.Context())
	// issue the request
	stream, err := child.PullState(reqCtx, request)
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

		var msg *traits.PullThermostatStateResponse
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
