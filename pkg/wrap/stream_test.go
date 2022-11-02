package wrap

import (
	"context"
	"errors"
	"github.com/smart-core-os/sc-api/go/traits"
	"reflect"
	"testing"
	"time"
)

func Test_clientSend(t *testing.T) {
	ctx := context.Background()
	clientServer := NewClientServerStream(ctx)
	client := clientServer.Client()
	server := clientServer.Server()

	sentMessage := &traits.Brightness{
		LevelPercent: 12,
	}
	receivedMessage := &traits.Brightness{}

	go client.SendMsg(sentMessage)
	server.RecvMsg(receivedMessage)

	if !reflect.DeepEqual(sentMessage, receivedMessage) {
		t.Errorf("%v != %v", sentMessage, receivedMessage)
	}
}

func Test_serverSend(t *testing.T) {
	ctx := context.Background()
	clientServer := NewClientServerStream(ctx)
	client := clientServer.Client()
	server := clientServer.Server()

	sentMessage := &traits.Brightness{
		LevelPercent: 12,
	}
	receivedMessage := &traits.Brightness{}

	go server.SendMsg(sentMessage)
	client.RecvMsg(receivedMessage)

	if !reflect.DeepEqual(sentMessage, receivedMessage) {
		t.Errorf("%v != %v", sentMessage, receivedMessage)
	}
}

func TestClientServerStream_headerReturnsOnClose(t *testing.T) {
	t.Run("nil err", func(t *testing.T) {
		assertHeaderErrOnClose(t, nil)
	})
	t.Run("non-nil err", func(t *testing.T) {
		assertHeaderErrOnClose(t, errors.New("early closed"))
	})
}

func assertHeaderErrOnClose(t *testing.T, wantErr error) {
	ctx := context.Background()
	cs := NewClientServerStream(ctx)
	client := cs.Client()

	go cs.Close(wantErr)

	var gotErr error
	headerDone := make(chan struct{})
	go func() {
		_, gotErr = client.Header()
		close(headerDone)
	}()

	select {
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for Client.Header()")
	case <-headerDone:
	}
	if gotErr != wantErr {
		t.Fatalf("Header err want %v, got %v", wantErr, gotErr)
	}
}
