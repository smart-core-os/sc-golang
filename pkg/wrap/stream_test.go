package wrap

import (
	"context"
	"reflect"
	"testing"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	"git.vanti.co.uk/smartcore/sc-api/go/types"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func Test_clientSend(t *testing.T) {
	ctx := context.Background()
	clientServer := newClientServerStream(ctx)
	client := clientServer.Client()
	server := clientServer.Server()

	sentMessage := &traits.Brightness{
		Level: &types.Int32Var{
			Value: wrapperspb.Int32(12),
		},
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
	clientServer := newClientServerStream(ctx)
	client := clientServer.Client()
	server := clientServer.Server()

	sentMessage := &traits.Brightness{
		Level: &types.Int32Var{
			Value: wrapperspb.Int32(12),
		},
	}
	receivedMessage := &traits.Brightness{}

	go server.SendMsg(sentMessage)
	client.RecvMsg(receivedMessage)

	if !reflect.DeepEqual(sentMessage, receivedMessage) {
		t.Errorf("%v != %v", sentMessage, receivedMessage)
	}
}
