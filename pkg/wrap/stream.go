package wrap

import (
	"context"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"io"
)

// ClientServerStream combines both a grpc.ServerStream and grpc.ClientStream
type ClientServerStream struct {
	ctx context.Context

	header  metadata.MD
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
		return nil, c.closeErr
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
		return c.closeErr
	case c.clientSend <- m:
		return nil
	}
}

func (c *clientStream) RecvMsg(m any) error {
	select {
	case <-c.Context().Done():
		return c.closeErr
	case val, ok := <-c.serverSend:
		if !ok {
			if c.closeErr != nil {
				return c.closeErr
			}
			return io.EOF
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
	s.header = metadata.Join(s.header, md)
	return s.broadcastHeaderSent()
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
		return s.closeErr
	case s.serverSend <- m:
		return nil
	}
}

func (s *serverStream) RecvMsg(m any) error {
	s.sendHeaderIfNeeded()
	select {
	case <-s.Context().Done():
		return s.closeErr
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

func (s *serverStream) broadcastHeaderSent() (err error) {
	defer func() {
		// I don't like using panic/recover here but the headerC chan already deals with locking of the close
		// and I don't want to have to do that here too
		recover()
		err = errors.New("headers already sent")
	}()
	close(s.headerC)
	return nil
}

func (s *serverStream) sendHeaderIfNeeded() {
	select {
	case <-s.headerC:
	// already sent header
	default:
		_ = s.SendHeader(nil)
	}
}
