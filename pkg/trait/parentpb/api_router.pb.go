// Code generated by protoc-gen-router. DO NOT EDIT.

package parentpb

import (
	context "context"
	fmt "fmt"
	traits "github.com/smart-core-os/sc-api/go/traits"
	router "github.com/smart-core-os/sc-golang/pkg/router"
	grpc "google.golang.org/grpc"
	io "io"
)

// ApiRouter is a traits.ParentApiServer that allows routing named requests to specific traits.ParentApiClient
type ApiRouter struct {
	traits.UnimplementedParentApiServer

	router.Router
}

// compile time check that we implement the interface we need
var _ traits.ParentApiServer = (*ApiRouter)(nil)

func NewApiRouter(opts ...router.Option) *ApiRouter {
	return &ApiRouter{
		Router: router.NewRouter(opts...),
	}
}

// WithParentApiClientFactory instructs the router to create a new
// client the first time Get is called for that name.
func WithParentApiClientFactory(f func(name string) (traits.ParentApiClient, error)) router.Option {
	return router.WithFactory(func(name string) (any, error) {
		return f(name)
	})
}

func (r *ApiRouter) Register(server grpc.ServiceRegistrar) {
	traits.RegisterParentApiServer(server, r)
}

// Add extends Router.Add to panic if client is not of type traits.ParentApiClient.
func (r *ApiRouter) Add(name string, client any) any {
	if !r.HoldsType(client) {
		panic(fmt.Sprintf("not correct type: client of type %T is not a traits.ParentApiClient", client))
	}
	return r.Router.Add(name, client)
}

func (r *ApiRouter) HoldsType(client any) bool {
	_, ok := client.(traits.ParentApiClient)
	return ok
}

func (r *ApiRouter) AddParentApiClient(name string, client traits.ParentApiClient) traits.ParentApiClient {
	res := r.Add(name, client)
	if res == nil {
		return nil
	}
	return res.(traits.ParentApiClient)
}

func (r *ApiRouter) RemoveParentApiClient(name string) traits.ParentApiClient {
	res := r.Remove(name)
	if res == nil {
		return nil
	}
	return res.(traits.ParentApiClient)
}

func (r *ApiRouter) GetParentApiClient(name string) (traits.ParentApiClient, error) {
	res, err := r.Get(name)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.(traits.ParentApiClient), nil
}

func (r *ApiRouter) ListChildren(ctx context.Context, request *traits.ListChildrenRequest) (*traits.ListChildrenResponse, error) {
	child, err := r.GetParentApiClient(request.Name)
	if err != nil {
		return nil, err
	}

	return child.ListChildren(ctx, request)
}

func (r *ApiRouter) PullChildren(request *traits.PullChildrenRequest, server traits.ParentApi_PullChildrenServer) error {
	child, err := r.GetParentApiClient(request.Name)
	if err != nil {
		return err
	}

	// so we can cancel our forwarding request if we can't send responses to our caller
	reqCtx, reqDone := context.WithCancel(server.Context())
	// issue the request
	stream, err := child.PullChildren(reqCtx, request)
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

		var msg *traits.PullChildrenResponse
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
