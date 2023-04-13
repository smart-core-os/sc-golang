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
		return nil, c.closeErrLocked()
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
		return c.closeErrLocked()
	case val, ok := <-c.serverSend:
		if !ok {
			return c.closeErrLocked()
		}
		proto.Merge(m.(proto.Message), val.(proto.Message))
		return nil
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
	s.sendHeaderIfNeeded()
	select {
	case <-s.Context().Done():
		return s.closeErrLocked()
	case val, ok := <-s.clientSend:
		if !ok {
			// we shouldn't send any more
			close(s.serverSend)
			return io.EOF
		}
		proto.Merge(m.(proto.Message), val.(proto.Message))
		return nil
	}
}

func (s *serverStream) sendHeaderIfNeeded() {
	// ignore error, SendHeader has no side effects if the headers have already been sent
	_ = s.SendHeader(nil)
}
