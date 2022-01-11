package router

import (
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Router tracks a registry of gRPC clients.
// Typically used by code generated via the protoc-gen-router plugin.
type Router struct {
	mu       sync.Mutex
	registry map[string]interface{} // of type MyServiceClient
	factory  Factory
}
type Factory func(string) (interface{}, error) // returns the type MyServiceClient

// NewRouter creates a new instance of Router with the given options.
func NewRouter(opts ...Option) *Router {
	r := &Router{
		registry: make(map[string]interface{}),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *Router) Add(name string, client interface{}) interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	r.registry[name] = client
	return old
}

func (r *Router) Remove(name string) interface{} {
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

func (r *Router) Get(name string) (interface{}, error) {
	r.mu.Lock()
	child, exists := r.registry[name]
	defer r.mu.Unlock()
	if !exists {
		if r.factory != nil {
			child, err := r.factory(name)
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

type Option func(r *Router)

// WithFactory configures a Router to call the given function when Get is called and no existing client is known.
// Prefer using the generated WithMyServiceClientFactory methods in the trait packages.
func WithFactory(f Factory) Option {
	return func(r *Router) {
		r.factory = f
	}
}
