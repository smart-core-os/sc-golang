// Code generated by protoc-gen-router. DO NOT EDIT.

package inputselect

import (
	context "context"
	fmt "fmt"
	traits "github.com/smart-core-os/sc-api/go/traits"
	router "github.com/smart-core-os/sc-golang/pkg/router"
	grpc "google.golang.org/grpc"
	io "io"
)

// ApiRouter is a traits.InputSelectApiServer that allows routing named requests to specific traits.InputSelectApiClient
type ApiRouter struct {
	traits.UnimplementedInputSelectApiServer

	router.Router
}

// compile time check that we implement the interface we need
var _ traits.InputSelectApiServer = (*ApiRouter)(nil)

func NewApiRouter(opts ...router.Option) *ApiRouter {
	return &ApiRouter{
		Router: router.NewRouter(opts...),
	}
}

// WithInputSelectApiClientFactory instructs the router to create a new
// client the first time Get is called for that name.
func WithInputSelectApiClientFactory(f func(name string) (traits.InputSelectApiClient, error)) router.Option {
	return router.WithFactory(func(name string) (interface{}, error) {
		return f(name)
	})
}

func (r *ApiRouter) Register(server *grpc.Server) {
	traits.RegisterInputSelectApiServer(server, r)
}

// Add extends Router.Add to panic if client is not of type traits.InputSelectApiClient.
func (r *ApiRouter) Add(name string, client interface{}) interface{} {
	if !r.HoldsType(client) {
		panic(fmt.Sprintf("not correct type: client of type %T is not a traits.InputSelectApiClient", client))
	}
	return r.Router.Add(name, client)
}

func (r *ApiRouter) HoldsType(client interface{}) bool {
	_, ok := client.(traits.InputSelectApiClient)
	return ok
}

func (r *ApiRouter) AddInputSelectApiClient(name string, client traits.InputSelectApiClient) traits.InputSelectApiClient {
	res := r.Add(name, client)
	if res == nil {
		return nil
	}
	return res.(traits.InputSelectApiClient)
}

func (r *ApiRouter) RemoveInputSelectApiClient(name string) traits.InputSelectApiClient {
	res := r.Remove(name)
	if res == nil {
		return nil
	}
	return res.(traits.InputSelectApiClient)
}

func (r *ApiRouter) GetInputSelectApiClient(name string) (traits.InputSelectApiClient, error) {
	res, err := r.Get(name)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.(traits.InputSelectApiClient), nil
}

func (r *ApiRouter) UpdateInput(ctx context.Context, request *traits.UpdateInputRequest) (*traits.Input, error) {
	child, err := r.GetInputSelectApiClient(request.Name)
	if err != nil {
		return nil, err
	}

	return child.UpdateInput(ctx, request)
}

func (r *ApiRouter) GetInput(ctx context.Context, request *traits.GetInputRequest) (*traits.Input, error) {
	child, err := r.GetInputSelectApiClient(request.Name)
	if err != nil {
		return nil, err
	}

	return child.GetInput(ctx, request)
}

func (r *ApiRouter) PullInput(request *traits.PullInputRequest, server traits.InputSelectApi_PullInputServer) error {
	child, err := r.GetInputSelectApiClient(request.Name)
	if err != nil {
		return err
	}

	// so we can cancel our forwarding request if we can't send responses to our caller
	reqCtx, reqDone := context.WithCancel(server.Context())
	// issue the request
	stream, err := child.PullInput(reqCtx, request)
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

		var msg *traits.PullInputResponse
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
