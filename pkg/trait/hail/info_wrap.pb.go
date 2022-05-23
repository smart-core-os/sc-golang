// Code generated by protoc-gen-wrapper. DO NOT EDIT.

package hail

import (
	traits "github.com/smart-core-os/sc-api/go/traits"
)

// WrapInfo	adapts a traits.HailInfoServer	and presents it as a traits.HailInfoClient
func WrapInfo(server traits.HailInfoServer) traits.HailInfoClient {
	return &infoWrapper{server}
}

type infoWrapper struct {
	server traits.HailInfoServer
}

// compile time check that we implement the interface we need
var _ traits.HailInfoClient = (*infoWrapper)(nil)

// UnwrapServer returns the underlying server instance.
func (w *infoWrapper) UnwrapServer() traits.HailInfoServer {
	return w.server
}

// Unwrap implements wrap.Unwrapper and returns the underlying server instance as an unknown type.
func (w *infoWrapper) Unwrap() interface{} {
	return w.UnwrapServer()
}