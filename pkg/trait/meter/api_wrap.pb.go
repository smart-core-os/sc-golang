// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package meter

import (
	traits "github.com/smart-core-os/sc-api/go/traits"
	wrap "github.com/smart-core-os/sc-golang/pkg/wrap"
	grpc "google.golang.org/grpc"
)

// WrapApi	adapts a traits.MeterApiServer	and presents it as a traits.MeterApiClient
func WrapApi(server traits.MeterApiServer) *ApiWrapper {
	conn := wrap.ServerToClient(traits.MeterApi_ServiceDesc, server)
	client := traits.NewMeterApiClient(conn)
	return &ApiWrapper{
		MeterApiClient: client,
		server:         server,
		conn:           conn,
		desc:           traits.MeterApi_ServiceDesc,
	}
}

type ApiWrapper struct {
	traits.MeterApiClient

	server traits.MeterApiServer
	conn   grpc.ClientConnInterface
	desc   grpc.ServiceDesc
}

// UnwrapServer returns the underlying server instance.
func (w *ApiWrapper) UnwrapServer() traits.MeterApiServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *ApiWrapper) Unwrap() any {
	return w.UnwrapServer()
}

func (w *ApiWrapper) UnwrapService() (grpc.ClientConnInterface, grpc.ServiceDesc) {
	return w.conn, w.desc
}
