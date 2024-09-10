package wrap

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/smart-core-os/sc-api/go/traits"
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
		assertHeaderErrOnClose(t, nil, nil)
	})
	t.Run("non-nil err", func(t *testing.T) {
		err := errors.New("early closed")
		assertHeaderErrOnClose(t, err, nil)
	})
}

func TestClientStream_Header(t *testing.T) {
	// check that closing the stream causes Header to nil,nil as documented on the interface
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := NewClientServerStream(ctx)
	s.Close(errors.New("some error"))
	h, err := s.Client().Header()
	if h != nil || err != nil {
		t.Errorf("Header = %v, %v; want nil, nil", h, err)
	}
}

func assertHeaderErrOnClose(t *testing.T, closeErr, wantErr error) {
	ctx := context.Background()
	cs := NewClientServerStream(ctx)
	client := cs.Client()

	go cs.Close(closeErr)

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

func TestClientServerStream_RecvMsg_errOnClose(t *testing.T) {
	t.Run("nil err", func(t *testing.T) {
		assertRecvMsgErrOnClose(t, nil, io.EOF)
	})
	t.Run("non-nil err", func(t *testing.T) {
		err := errors.New("early closed")
		assertRecvMsgErrOnClose(t, err, err)
	})
}

func assertRecvMsgErrOnClose(t *testing.T, closeErr, wantErr error) {
	ctx := context.Background()
	cs := NewClientServerStream(ctx)
	client := cs.Client()

	go cs.Close(closeErr)

	var gotErr error
	gotMsg := &emptypb.Empty{}
	recvDone := make(chan struct{})
	go func() {
		gotErr = client.RecvMsg(gotMsg)
		close(recvDone)
	}()

	select {
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for Client.RecvMsg()")
	case <-recvDone:
	}
	if gotErr != wantErr {
		t.Fatalf("RecvMsg err want %v, got %v", wantErr, gotErr)
	}
}
