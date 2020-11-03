package client

import (
	"context"
	"net/url"
	"time"

	"git.vanti.co.uk/smartcore/sc-api/go/info"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Client struct {
	conn        *grpc.ClientConn
	retryPolicy []grpc.CallOption
	logger      *zap.Logger
}

func NewClient(processorAddr *url.URL, auth AuthCredentials, logger *zap.Logger) Client {
	// connect to processor
	conn, err := grpc.Dial(
		processorAddr.Host,
		grpc.WithTransportCredentials(auth.Creds),
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor()),
	)
	if err != nil {
		logger.Fatal("Could not create connection to server", zap.Error(err))
	}

	return Client{
		conn,
		[]grpc.CallOption{
			grpc_retry.WithMax(5),
			grpc_retry.WithPerRetryTimeout(2 * time.Second),
			grpc_retry.WithBackoff(grpc_retry.BackoffExponentialWithJitter(100*time.Millisecond, 0.01)),
		},
		logger,
	}
}

// Shutdown this connection to the client
func (c *Client) Shutdown() error {
	return c.conn.Close()
}

// Info service functions
func (c *Client) GetDeviceList(ctx context.Context) ([]*info.Device, error) {
	i := info.NewInfoClient(c.conn)
	resp, err := i.ListDevices(ctx, &info.ListDevicesRequest{})
	if err != nil {
		return nil, err
	}
	devices := resp.Devices
	// loop through to get all known devices for this controller
	for resp.NextPageToken != "" {
		resp, err = i.ListDevices(ctx, &info.ListDevicesRequest{PageToken: resp.NextPageToken})
		if err != nil {
			return nil, err
		}
		devices = append(devices, resp.Devices...)
	}

	return devices, nil
}

/*func (c *Client) Devices(ctx context.Context) (<- chan *info.Device, error) {

}*/

// Health service functions
