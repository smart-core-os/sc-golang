// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package count

import (
	traits "github.com/smart-core-os/sc-api/go/traits"
	wrap "github.com/smart-core-os/sc-golang/pkg/wrap"
	grpc "google.golang.org/grpc"
)

// WrapInfo	adapts a traits.CountInfoServer	and presents it as a traits.CountInfoClient
func WrapInfo(server traits.CountInfoServer) *InfoWrapper {
	conn := wrap.ServerToClient(traits.CountInfo_ServiceDesc, server)
	client := traits.NewCountInfoClient(conn)
	return &InfoWrapper{
		CountInfoClient: client,
		server:          server,
		conn:            conn,
		desc:            traits.CountInfo_ServiceDesc,
	}
}

type InfoWrapper struct {
	traits.CountInfoClient

	server traits.CountInfoServer
	conn   grpc.ClientConnInterface
	desc   grpc.ServiceDesc
}

// UnwrapServer returns the underlying server instance.
func (w *InfoWrapper) UnwrapServer() traits.CountInfoServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *InfoWrapper) Unwrap() any {
	return w.UnwrapServer()
}

func (w *InfoWrapper) UnwrapService() (grpc.ClientConnInterface, grpc.ServiceDesc) {
	return w.conn, w.desc
}
