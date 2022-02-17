// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package energystorage

import (
	context "context"
	traits "github.com/smart-core-os/sc-api/go/traits"
	grpc "google.golang.org/grpc"
)

// WrapInfo	adapts a traits.EnergyStorageInfoServer	and presents it as a traits.EnergyStorageInfoClient
func WrapInfo(server traits.EnergyStorageInfoServer) traits.EnergyStorageInfoClient {
	return &infoWrapper{server}
}

type infoWrapper struct {
	server traits.EnergyStorageInfoServer
}

// compile time check that we implement the interface we need
var _ traits.EnergyStorageInfoClient = (*infoWrapper)(nil)

// UnwrapServer returns the underlying server instance.
func (w *infoWrapper) UnwrapServer() traits.EnergyStorageInfoServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *infoWrapper) Unwrap() interface{} {
	return w.UnwrapServer()
}

func (w *infoWrapper) DescribeEnergyLevel(ctx context.Context, req *traits.DescribeEnergyLevelRequest, _ ...grpc.CallOption) (*traits.EnergyLevelSupport, error) {
	return w.server.DescribeEnergyLevel(ctx, req)
}
