package group

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

var ctx = context.Background()
var streamTimout = 500 * time.Millisecond

func checkErr(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%v returned an error: %v", msg, err)
	}
}

func dial(lis *bufconn.Listener) (*grpc.ClientConn, error) {
	dialler := func(ctx context.Context, s string) (net.Conn, error) {
		return lis.Dial()
	}
	return grpc.DialContext(ctx, "test", grpc.WithContextDialer(dialler), grpc.WithInsecure())
}
