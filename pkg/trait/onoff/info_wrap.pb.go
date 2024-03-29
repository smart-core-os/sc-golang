// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package onoff

import (
	context "context"
	traits "github.com/smart-core-os/sc-api/go/traits"
	grpc "google.golang.org/grpc"
)

// WrapInfo	adapts a traits.OnOffInfoServer	and presents it as a traits.OnOffInfoClient
func WrapInfo(server traits.OnOffInfoServer) traits.OnOffInfoClient {
	return &infoWrapper{server}
}

type infoWrapper struct {
	server traits.OnOffInfoServer
}

// compile time check that we implement the interface we need
var _ traits.OnOffInfoClient = (*infoWrapper)(nil)

// UnwrapServer returns the underlying server instance.
func (w *infoWrapper) UnwrapServer() traits.OnOffInfoServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *infoWrapper) Unwrap() any {
	return w.UnwrapServer()
}

func (w *infoWrapper) DescribeOnOff(ctx context.Context, req *traits.DescribeOnOffRequest, _ ...grpc.CallOption) (*traits.OnOffSupport, error) {
	return w.server.DescribeOnOff(ctx, req)
}
