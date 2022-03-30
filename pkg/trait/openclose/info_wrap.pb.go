// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package openclose

import (
	context "context"
	traits "github.com/smart-core-os/sc-api/go/traits"
	grpc "google.golang.org/grpc"
)

// WrapInfo	adapts a traits.OpenCloseInfoServer	and presents it as a traits.OpenCloseInfoClient
func WrapInfo(server traits.OpenCloseInfoServer) traits.OpenCloseInfoClient {
	return &infoWrapper{server}
}

type infoWrapper struct {
	server traits.OpenCloseInfoServer
}

// compile time check that we implement the interface we need
var _ traits.OpenCloseInfoClient = (*infoWrapper)(nil)

// UnwrapServer returns the underlying server instance.
func (w *infoWrapper) UnwrapServer() traits.OpenCloseInfoServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *infoWrapper) Unwrap() interface{} {
	return w.UnwrapServer()
}

func (w *infoWrapper) DescribePositions(ctx context.Context, req *traits.DescribePositionsRequest, _ ...grpc.CallOption) (*traits.PositionsSupport, error) {
	return w.server.DescribePositions(ctx, req)
}
