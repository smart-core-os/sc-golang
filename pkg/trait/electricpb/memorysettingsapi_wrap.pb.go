// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package electricpb

import (
	wrap "github.com/smart-core-os/sc-golang/pkg/wrap"
	grpc "google.golang.org/grpc"
)

// WrapMemorySettingsApi	adapts a MemorySettingsApiServer	and presents it as a MemorySettingsApiClient
func WrapMemorySettingsApi(server MemorySettingsApiServer) *MemorySettingsApiWrapper {
	conn := wrap.ServerToClient(MemorySettingsApi_ServiceDesc, server)
	client := NewMemorySettingsApiClient(conn)
	return &MemorySettingsApiWrapper{
		MemorySettingsApiClient: client,
		server:                  server,
		conn:                    conn,
		desc:                    MemorySettingsApi_ServiceDesc,
	}
}

type MemorySettingsApiWrapper struct {
	MemorySettingsApiClient

	server MemorySettingsApiServer
	conn   grpc.ClientConnInterface
	desc   grpc.ServiceDesc
}

// UnwrapServer returns the underlying server instance.
func (w *MemorySettingsApiWrapper) UnwrapServer() MemorySettingsApiServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *MemorySettingsApiWrapper) Unwrap() any {
	return w.UnwrapServer()
}

func (w *MemorySettingsApiWrapper) UnwrapService() (grpc.ClientConnInterface, grpc.ServiceDesc) {
	return w.conn, w.desc
}
