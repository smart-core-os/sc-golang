// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package lockunlock

import (
	traits "github.com/smart-core-os/sc-api/go/traits"
	wrap "github.com/smart-core-os/sc-golang/pkg/wrap"
	grpc "google.golang.org/grpc"
)

// WrapInfo	adapts a traits.LockUnlockInfoServer	and presents it as a traits.LockUnlockInfoClient
func WrapInfo(server traits.LockUnlockInfoServer) *InfoWrapper {
	conn := wrap.ServerToClient(traits.LockUnlockInfo_ServiceDesc, server)
	client := traits.NewLockUnlockInfoClient(conn)
	return &InfoWrapper{
		LockUnlockInfoClient: client,
		server:               server,
		conn:                 conn,
		desc:                 traits.LockUnlockInfo_ServiceDesc,
	}
}

type InfoWrapper struct {
	traits.LockUnlockInfoClient

	server traits.LockUnlockInfoServer
	conn   grpc.ClientConnInterface
	desc   grpc.ServiceDesc
}

// UnwrapServer returns the underlying server instance.
func (w *InfoWrapper) UnwrapServer() traits.LockUnlockInfoServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *InfoWrapper) Unwrap() any {
	return w.UnwrapServer()
}

func (w *InfoWrapper) UnwrapService() (grpc.ClientConnInterface, grpc.ServiceDesc) {
	return w.conn, w.desc
}
