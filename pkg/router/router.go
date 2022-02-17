package router

import (
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Router tracks a registry of gRPC clients.
// Typically used by code generated via the protoc-gen-router plugin.
type Router interface {
	// Add adds a named client to this Router.
	// Add returns the old client associated with this name, or nil if there wasn't one.
	// If HoldsType returns false for the given client, this will panic.
	Add(name string, client interface{}) interface{}
	// HoldsType returns true if this Router holds clients of the specified type.
	HoldsType(client interface{}) bool
	// Remove removes and returns a named client.
	Remove(name string) interface{}
	// Has returns true if this Router has a client with the given name.
	Has(name string) bool
	// Get returns the client for the given name.
	// An error will be returned if no such client exists.
	Get(name string) (interface{}, error)
}

type router struct {
	mu       sync.Mutex
	registry map[string]interface{} // of type MyServiceClient
	factory  Factory
	fallback Factory
}
type Factory func(string) (interface{}, error) // returns the type MyServiceClient

// NewRouter creates a new instance of Router with the given options.
func NewRouter(opts ...Option) Router {
	r := &router{
		registry: make(map[string]interface{}),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *router) Add(name string, client interface{}) interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	r.registry[name] = client
	return old
}

func (r *router) HoldsType(_ interface{}) bool {
	return true
}

func (r *router) Remove(name string) interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.registry[name]
	delete(r.registry, name)
	return old
}

func (r *router) Has(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, exists := r.registry[name]
	return exists
}

func (r *router) Get(name string) (child interface{}, err error) {
	r.mu.Lock()
	child, exists := r.registry[name]
	defer r.mu.Unlock()
	if !exists {
		child, exists, err = invoke(name, r.fallback)
	}
	if !exists {
		child, exists, err = invoke(name, r.factory)
		if exists {
			r.registry[name] = child
		}
	}

	if !exists {
		return nil, status.Error(codes.NotFound, name)
	}
	return
}
func invoke(name string, f Factory) (interface{}, bool, error) {
	if f == nil {
		return nil, false, nil
	}
	child, err := f(name)
	return child, child != nil && err == nil, err
}

type Option func(r *router)

// WithFactory configures a Router to call the given function when Get is called and no existing client is known.
// Prefer using the generated WithMyServiceClientFactory methods in the trait packages.
func WithFactory(f Factory) Option {
	return func(r *router) {
		r.factory = f
	}
}

// WithFallback configures a Router to ask the given function when Get is called and no existing client is known.
// If WithFallback and WithFactory are both configured, WithFallback will be called first, only using WithFactory if
// WithFallback returns nil or an error.
func WithFallback(f Factory) Option {
	return func(r *router) {
		r.fallback = f
	}
}
