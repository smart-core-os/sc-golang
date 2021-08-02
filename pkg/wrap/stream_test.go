package wrap

import (
	"context"
	"reflect"
	"testing"

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
