package wrap

import (
	"context"
	"fmt"
	"reflect"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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

	ctx, clientServerStream, ss, cs := w.startStream(ctx, method)
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

	if err := cs.SendMsg(args); err != nil {
		return err
	}
	if err := cs.CloseSend(); err != nil {
		return err
	}
	err := cs.RecvMsg(reply)

	// gather headers and trailers
	for _, opt := range opts {
		switch opt := opt.(type) {
		case grpc.HeaderCallOption:
			hdr, hdrErr := cs.Header()
			if hdrErr != nil && err == nil {
				err = hdrErr
			}
			*opt.HeaderAddr = cloneMD(hdr)

		case grpc.TrailerCallOption:
			trailer := cs.Trailer()
			*opt.TrailerAddr = cloneMD(trailer)
		}
	}

	return err
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

	ctx, clientServerStream, ss, cs := w.startStream(ctx, method)
	go func() {
		err := matched.Handler(w.srv, ss)
		clientServerStream.Close(err)
	}()

	return cs, nil
}

func (w *wrapper) startStream(ctx context.Context, method string) (context.Context, *ClientServerStream, grpc.ServerStream, grpc.ClientStream) {
	// convert client's outgoing metadata to server's incoming metadata
	md, _ := metadata.FromOutgoingContext(ctx)
	md = cloneMD(md) // to prevent client from concurrently modifying the metadata

	ctx = metadata.NewIncomingContext(ctx, md)
	// attach a TransportStream to the context, so the server can send headers
	sts := &serverTransportStream{method: method}
	ctx = grpc.NewContextWithServerTransportStream(ctx, sts)

	clientServerStream := NewClientServerStream(ctx)
	ss := clientServerStream.Server()
	sts.ss = ss
	cs := clientServerStream.Client()

	return ctx, clientServerStream, ss, cs
}

// thin implementation of grpc.ServerTransportStream, only to allow grpc.SetHeader to work,
// which relies on a ServerTransportStream attached to the server context
type serverTransportStream struct {
	ss     grpc.ServerStream
	method string
}

func (ts *serverTransportStream) SetHeader(md metadata.MD) error {
	return ts.ss.SetHeader(md)
}

func (ts *serverTransportStream) SendHeader(md metadata.MD) error {
	return ts.ss.SendHeader(md)
}

func (ts *serverTransportStream) SetTrailer(md metadata.MD) error {
	ts.ss.SetTrailer(md)
	return nil
}

func (ts *serverTransportStream) Method() string {
	return ts.method
}

func cloneMD(md metadata.MD) metadata.MD {
	if md == nil {
		return nil
	}
	newMD := make(metadata.MD)
	for k, v := range md {
		newMD[k] = append([]string(nil), v...)
	}
	return newMD
}
