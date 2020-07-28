package main

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"git.vanti.co.uk/smartcore/sc-api/go/device/info"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"

	"sc-golang/pkg/server"
)

func main() {

	z, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("Could not create logger: %v", err))
	}

	ctx := context.Background()

	// setup TLS creds
	c, err := credentials.NewServerTLSFromFile("test/certs/service.pem", "test/certs/service.key")
	if err != nil {
		z.Fatal("Could not create credentials", zap.Error(err))
	}
	s := server.NewServer(ctx, server.AuthProvider{c}, z)

	addr, err := url.Parse("tcp://127.0.0.1:9443")
	done := s.Startup(addr)

	s.RegisterDevice(&info.Device{
		Name:  "/test/device",
		Title: "Test Device",
	})

	// wait for termination
	select {
	case err := <-done:
		// something caused an error
		if err != nil {
			log.Printf("Shutting down: %v", err)
		} else {
			log.Println("Shutting down")
		}
	case <-ctx.Done():
		// interrupt
		log.Println("Shutting down")
	}

}
