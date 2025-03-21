// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package temperaturepb

import (
	traits "github.com/smart-core-os/sc-api/go/traits"
	wrap "github.com/smart-core-os/sc-golang/pkg/wrap"
	grpc "google.golang.org/grpc"
)

// WrapApi	adapts a traits.TemperatureApiServer	and presents it as a traits.TemperatureApiClient
func WrapApi(server traits.TemperatureApiServer) *ApiWrapper {
	conn := wrap.ServerToClient(traits.TemperatureApi_ServiceDesc, server)
	client := traits.NewTemperatureApiClient(conn)
	return &ApiWrapper{
		TemperatureApiClient: client,
		server:               server,
		conn:                 conn,
		desc:                 traits.TemperatureApi_ServiceDesc,
	}
}

type ApiWrapper struct {
	traits.TemperatureApiClient

	server traits.TemperatureApiServer
	conn   grpc.ClientConnInterface
	desc   grpc.ServiceDesc
}

// UnwrapServer returns the underlying server instance.
func (w *ApiWrapper) UnwrapServer() traits.TemperatureApiServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *ApiWrapper) Unwrap() any {
	return w.UnwrapServer()
}

func (w *ApiWrapper) UnwrapService() (grpc.ClientConnInterface, grpc.ServiceDesc) {
	return w.conn, w.desc
}
