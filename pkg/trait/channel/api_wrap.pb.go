// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package channel

import (
	traits "github.com/smart-core-os/sc-api/go/traits"
	wrap "github.com/smart-core-os/sc-golang/pkg/wrap"
	grpc "google.golang.org/grpc"
)

// WrapApi	adapts a traits.ChannelApiServer	and presents it as a traits.ChannelApiClient
func WrapApi(server traits.ChannelApiServer) *ApiWrapper {
	conn := wrap.ServerToClient(traits.ChannelApi_ServiceDesc, server)
	client := traits.NewChannelApiClient(conn)
	return &ApiWrapper{
		ChannelApiClient: client,
		server:           server,
		conn:             conn,
		desc:             traits.ChannelApi_ServiceDesc,
	}
}

type ApiWrapper struct {
	traits.ChannelApiClient

	server traits.ChannelApiServer
	conn   grpc.ClientConnInterface
	desc   grpc.ServiceDesc
}

// UnwrapServer returns the underlying server instance.
func (w *ApiWrapper) UnwrapServer() traits.ChannelApiServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *ApiWrapper) Unwrap() any {
	return w.UnwrapServer()
}

func (w *ApiWrapper) UnwrapService() (grpc.ClientConnInterface, grpc.ServiceDesc) {
	return w.conn, w.desc
}
