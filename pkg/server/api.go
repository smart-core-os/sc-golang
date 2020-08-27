package server

import "google.golang.org/grpc"

type GrpcApi interface {
	Register(server *grpc.Server)
}
