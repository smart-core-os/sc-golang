// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package ptz

import (
	traits "github.com/smart-core-os/sc-api/go/traits"
	wrap "github.com/smart-core-os/sc-golang/pkg/wrap"
)

// WrapInfo	adapts a traits.PtzInfoServer	and presents it as a traits.PtzInfoClient
func WrapInfo(server traits.PtzInfoServer) traits.PtzInfoClient {
	conn := wrap.ServerToClient(traits.PtzInfo_ServiceDesc, server)
	client := traits.NewPtzInfoClient(conn)
	return &infoWrapper{
		PtzInfoClient: client,
		server:        server,
	}
}

type infoWrapper struct {
	traits.PtzInfoClient

	server traits.PtzInfoServer
}

// compile time check that we implement the interface we need
var _ traits.PtzInfoClient = (*infoWrapper)(nil)

// UnwrapServer returns the underlying server instance.
func (w *infoWrapper) UnwrapServer() traits.PtzInfoServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *infoWrapper) Unwrap() any {
	return w.UnwrapServer()
}
