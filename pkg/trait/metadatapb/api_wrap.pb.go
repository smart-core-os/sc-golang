// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package metadatapb

import (
	traits "github.com/smart-core-os/sc-api/go/traits"
	wrap "github.com/smart-core-os/sc-golang/pkg/wrap"
	grpc "google.golang.org/grpc"
)

// WrapApi	adapts a traits.MetadataApiServer	and presents it as a traits.MetadataApiClient
func WrapApi(server traits.MetadataApiServer) *ApiWrapper {
	conn := wrap.ServerToClient(traits.MetadataApi_ServiceDesc, server)
	client := traits.NewMetadataApiClient(conn)
	return &ApiWrapper{
		MetadataApiClient: client,
		server:            server,
		conn:              conn,
		desc:              traits.MetadataApi_ServiceDesc,
	}
}

type ApiWrapper struct {
	traits.MetadataApiClient

	server traits.MetadataApiServer
	conn   grpc.ClientConnInterface
	desc   grpc.ServiceDesc
}

// UnwrapServer returns the underlying server instance.
func (w *ApiWrapper) UnwrapServer() traits.MetadataApiServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *ApiWrapper) Unwrap() any {
	return w.UnwrapServer()
}

func (w *ApiWrapper) UnwrapService() (grpc.ClientConnInterface, grpc.ServiceDesc) {
	return w.conn, w.desc
}
