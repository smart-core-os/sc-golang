package wrap

import (
	"context"
	"errors"
	"io"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

// ClientServerStream combines both a grpc.ServerStream and grpc.ClientStream
type ClientServerStream struct {
	ctx context.Context

	header  metadata.MD
	headerM sync.Mutex    // guards closing of headerC
	headerC chan struct{} // closed once calls to clientStream.Header should return

	serverSend chan any
	clientSend chan any
	trailer    metadata.MD
	closed     context.CancelFunc
	closeErr   error
}

func NewClientServerStream(ctx context.Context) *ClientServerStream {
	newCtx, closed := context.WithCancel(ctx)
	return &ClientServerStream{
		ctx:        newCtx,
		closed:     closed,
		headerC:    make(chan struct{}),
		serverSend: make(chan any),
		clientSend: make(chan any),
	}
}

func (s *ClientServerStream) Close(err error) {
	s.closeErr = err
	close(s.serverSend)
	s.closed()
}

// safe to call if s.serverSend is closed
func (s *ClientServerStream) closeErrLocked() error {
	if s.closeErr == nil {
		return io.EOF
	}
	return s.closeErr
}

func (s *ClientServerStream) Client() grpc.ClientStream {
	return &clientStream{s}
}

func (s *ClientServerStream) Server() grpc.ServerStream {
	return &serverStream{s}
}

type clientStream struct {
	*ClientServerStream
}

func (c *clientStream) Header() (metadata.MD, error) {
	select {
	case <-c.ctx.Done():
		select {
		case <-c.headerC:
			// we should still return the headers if we have them, even if the context is done
			return c.header, nil
		default:
			// when the stream is terminated without headers, ClientStream should return a nil error
			return nil, nil
		}
	case <-c.headerC:
		return c.header, nil
	}
}

func (c *clientStream) Trailer() metadata.MD {
	return c.trailer
}

func (c *clientStream) CloseSend() error {
	close(c.clientSend)
	return nil
}

func (c *clientStream) Context() context.Context {
	return c.ctx
}

func (c *clientStream) SendMsg(m any) error {
	select {
	case <-c.ctx.Done():
		return c.closeErrLocked()
	case c.clientSend <- m:
		return nil
	}
}

func (c *clientStream) RecvMsg(m any) error {
	select {
	case <-c.Context().Done():
		// closeErr may or may not be available depending on why the context has ended
		//  1. if c.closed() was called by c.Close(...), then c.closeErr will be available
		//  2. if the parent context was cancelled, then c.closeErr may not be available
		select {
		case _, ok := <-c.serverSend:
			if !ok {
				return c.closeErrLocked()
			}
		default:
		}
		return c.Context().Err()
	case val, ok := <-c.serverSend:
		if !ok {
			return c.closeErrLocked()
		}
		return permissiveProtoMerge(m.(proto.Message), val.(proto.Message))
	}
}

type serverStream struct {
	*ClientServerStream
}

func (s *serverStream) SetHeader(md metadata.MD) error {
	s.header = metadata.Join(s.header, md)
	return nil
}

func (s *serverStream) SendHeader(md metadata.MD) error {
	s.headerM.Lock()
	defer s.headerM.Unlock()

	select {
	case <-s.headerC:
		return errors.New("headers already sent")
	default:
	}
	s.header = metadata.Join(s.header, md)
	close(s.headerC)
	return nil
}

func (s *serverStream) SetTrailer(md metadata.MD) {
	s.trailer = metadata.Join(s.trailer, md)
}

func (s *serverStream) Context() context.Context {
	return s.ctx
}

func (s *serverStream) SendMsg(m any) error {
	s.sendHeaderIfNeeded()
	select {
	case <-s.ctx.Done():
		return s.closeErrLocked()
	case s.serverSend <- m:
		return nil
	}
}

func (s *serverStream) RecvMsg(m any) error {
	select {
	case <-s.Context().Done():
		return s.closeErrLocked()
	case val, ok := <-s.clientSend:
		if !ok {
			return io.EOF
		}
		return permissiveProtoMerge(m.(proto.Message), val.(proto.Message))
	}
}

func (s *serverStream) sendHeaderIfNeeded() {
	// ignore error, SendHeader has no side effects if the headers have already been sent
	_ = s.SendHeader(nil)
}

// works like proto.Merge but allows messages with different descriptors by performing a marshal/unmarshal
//
// This is the recommended way to do it: https://github.com/golang/protobuf/issues/1163#issuecomment-654334690
func permissiveProtoMerge(dst, src proto.Message) error {
	if dst.ProtoReflect().Descriptor() == src.ProtoReflect().Descriptor() {
		// easy case, where proto.Merge can be used
		proto.Merge(dst, src)
		return nil
	}
	// mismatched descriptors, so we need to marshal/unmarshal
	encoded, err := proto.Marshal(src)
	if err != nil {
		return err
	}
	return proto.UnmarshalOptions{Merge: true}.Unmarshal(encoded, dst)
}
