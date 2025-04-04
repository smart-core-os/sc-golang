// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package fanspeedpb

import (
	traits "github.com/smart-core-os/sc-api/go/traits"
	wrap "github.com/smart-core-os/sc-golang/pkg/wrap"
	grpc "google.golang.org/grpc"
)

// WrapInfo	adapts a traits.FanSpeedInfoServer	and presents it as a traits.FanSpeedInfoClient
func WrapInfo(server traits.FanSpeedInfoServer) *InfoWrapper {
	conn := wrap.ServerToClient(traits.FanSpeedInfo_ServiceDesc, server)
	client := traits.NewFanSpeedInfoClient(conn)
	return &InfoWrapper{
		FanSpeedInfoClient: client,
		server:             server,
		conn:               conn,
		desc:               traits.FanSpeedInfo_ServiceDesc,
	}
}

type InfoWrapper struct {
	traits.FanSpeedInfoClient

	server traits.FanSpeedInfoServer
	conn   grpc.ClientConnInterface
	desc   grpc.ServiceDesc
}

// UnwrapServer returns the underlying server instance.
func (w *InfoWrapper) UnwrapServer() traits.FanSpeedInfoServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *InfoWrapper) Unwrap() any {
	return w.UnwrapServer()
}

func (w *InfoWrapper) UnwrapService() (grpc.ClientConnInterface, grpc.ServiceDesc) {
	return w.conn, w.desc
}
