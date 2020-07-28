package server

import (
	"google.golang.org/grpc/credentials"
)

type AuthProvider struct {
	Creds credentials.TransportCredentials
}
