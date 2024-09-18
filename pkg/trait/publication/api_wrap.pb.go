// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package publication

import (
	traits "github.com/smart-core-os/sc-api/go/traits"
	wrap "github.com/smart-core-os/sc-golang/pkg/wrap"
	grpc "google.golang.org/grpc"
)

// WrapApi	adapts a traits.PublicationApiServer	and presents it as a traits.PublicationApiClient
func WrapApi(server traits.PublicationApiServer) *ApiWrapper {
	conn := wrap.ServerToClient(traits.PublicationApi_ServiceDesc, server)
	client := traits.NewPublicationApiClient(conn)
	return &ApiWrapper{
		PublicationApiClient: client,
		server:               server,
		conn:                 conn,
		desc:                 traits.PublicationApi_ServiceDesc,
	}
}

type ApiWrapper struct {
	traits.PublicationApiClient

	server traits.PublicationApiServer
	conn   grpc.ClientConnInterface
	desc   grpc.ServiceDesc
}

// UnwrapServer returns the underlying server instance.
func (w *ApiWrapper) UnwrapServer() traits.PublicationApiServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *ApiWrapper) Unwrap() any {
	return w.UnwrapServer()
}

func (w *ApiWrapper) UnwrapService() (grpc.ClientConnInterface, grpc.ServiceDesc) {
	return w.conn, w.desc
}
