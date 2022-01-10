// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package airtemperature

import (
	context "context"
	traits "github.com/smart-core-os/sc-api/go/traits"
	grpc "google.golang.org/grpc"
)

// Wrap Info	adapts a traits.AirTemperatureInfoServer	and presents it as a traits.AirTemperatureInfoClient
func WrapInfo(server traits.AirTemperatureInfoServer) traits.AirTemperatureInfoClient {
	return &infoWrapper{server}
}

type infoWrapper struct {
	server traits.AirTemperatureInfoServer
}

// compile time check that we implement the interface we need
var _ traits.AirTemperatureInfoClient = (*infoWrapper)(nil)

func (w *infoWrapper) DescribeAirTemperature(ctx context.Context, req *traits.DescribeAirTemperatureRequest, _ ...grpc.CallOption) (*traits.AirTemperatureSupport, error) {
	return w.server.DescribeAirTemperature(ctx, req)
}
