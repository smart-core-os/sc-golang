// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package ptz

import (
	context "context"
	traits "github.com/smart-core-os/sc-api/go/traits"
	grpc "google.golang.org/grpc"
)

// WrapInfo	adapts a traits.PtzInfoServer	and presents it as a traits.PtzInfoClient
func WrapInfo(server traits.PtzInfoServer) traits.PtzInfoClient {
	return &infoWrapper{server}
}

type infoWrapper struct {
	server traits.PtzInfoServer
}

// compile time check that we implement the interface we need
var _ traits.PtzInfoClient = (*infoWrapper)(nil)

// UnwrapServer returns the underlying server instance.
func (w *infoWrapper) UnwrapServer() traits.PtzInfoServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *infoWrapper) Unwrap() any {
	return w.UnwrapServer()
}

func (w *infoWrapper) DescribePtz(ctx context.Context, req *traits.DescribePtzRequest, _ ...grpc.CallOption) (*traits.PtzSupport, error) {
	return w.server.DescribePtz(ctx, req)
}
