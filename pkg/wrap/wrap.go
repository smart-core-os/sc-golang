package wrap

import (
	"context"
	"fmt"
	"reflect"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ServerToClient(desc grpc.ServiceDesc, srv any) grpc.ClientConnInterface {
	// check that srv is the right type to be a server for desc
	expectType := reflect.TypeOf(desc.HandlerType).Elem()
	if !reflect.TypeOf(srv).Implements(expectType) {
		panic(fmt.Sprintf("grpcdynamic: srv must be of type %v", expectType))
	}

	return &wrapper{
		desc: desc,
		srv:  srv,
	}
}

type wrapper struct {
	desc grpc.ServiceDesc
	srv  any
}

func (w *wrapper) Invoke(ctx context.Context, method string, args any, reply any, opts ...grpc.CallOption) error {
	var matched *grpc.MethodDesc
	for _, m := range w.desc.Methods {
		methodName := fmt.Sprintf("/%s/%s", w.desc.ServiceName, m.MethodName)
		if methodName == method {
			matched = &m
			break
		}
	}
	if matched == nil {
		return status.Error(codes.Unimplemented, "method not found")
	}

	clientServerStream := NewClientServerStream(ctx)
	ss := clientServerStream.Server()
	go func() {
		res, err := matched.Handler(w.srv, ctx, func(dst any) error {
			return ss.RecvMsg(dst)
		}, nil)
		if err != nil {
			clientServerStream.Close(err)
			return
		}
		err = ss.SendMsg(res)
		clientServerStream.Close(err)
	}()

	cs := clientServerStream.Client()
	if err := cs.SendMsg(args); err != nil {
		return err
	}
	return cs.RecvMsg(reply)
}

func (w *wrapper) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	var matched *grpc.StreamDesc
	for _, s := range w.desc.Streams {
		streamName := fmt.Sprintf("/%s/%s", w.desc.ServiceName, s.StreamName)
		if streamName == method {
			matched = &s
			break
		}
	}
	if matched == nil {
		return nil, status.Error(codes.Unimplemented, "method not found")
	}

	if matched.ServerStreams != desc.ServerStreams || matched.ClientStreams != desc.ClientStreams {
		return nil, status.Error(codes.Internal, "method stream shape mismatch")
	}

	clientServerStream := NewClientServerStream(ctx)
	go func() {
		err := matched.Handler(w.srv, clientServerStream.Server())
		clientServerStream.Close(err)
	}()

	return clientServerStream.Client(), nil
}
