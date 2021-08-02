package wrap

import (
	"context"
	"io"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

// ClientServerStream combines both a grpc.ServerStream and grpc.ClientStream
type ClientServerStream struct {
	ctx        context.Context
	header     metadata.MD
	headerCond *sync.Cond
	headerSent bool
	serverSend chan interface{}
	clientSend chan interface{}
	trailer    metadata.MD
	closed     context.CancelFunc
	closeErr   error
}

func NewClientServerStream(ctx context.Context) *ClientServerStream {
	newCtx, closed := context.WithCancel(ctx)
	return &ClientServerStream{
		ctx:        newCtx,
		closed:     closed,
		headerCond: sync.NewCond(&sync.Mutex{}),
		serverSend: make(chan interface{}),
		clientSend: make(chan interface{}),
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
	c.headerCond.L.Lock()
	for !c.headerSent {
		c.headerCond.Wait()
	}
	c.headerCond.L.Unlock()
	return c.header, nil
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

func (c *clientStream) SendMsg(m interface{}) error {
	select {
	case <-c.ctx.Done():
		return c.closeErr
	case c.clientSend <- m:
		return nil
	}
}

func (c *clientStream) RecvMsg(m interface{}) error {
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
	s.headerCond.L.Lock()
	s.headerSent = true
	s.headerCond.L.Unlock()
	s.headerCond.Broadcast()
	return nil
}

func (s *serverStream) SetTrailer(md metadata.MD) {
	s.trailer = metadata.Join(s.trailer, md)
}

func (s *serverStream) Context() context.Context {
	return s.ctx
}

func (s *serverStream) SendMsg(m interface{}) error {
	if !s.headerSent {
		_ = s.SendHeader(nil)
	}
	select {
	case <-s.ctx.Done():
		return s.closeErr
	case s.serverSend <- m:
		return nil
	}
}

func (s *serverStream) RecvMsg(m interface{}) error {
	if !s.headerSent {
		_ = s.SendHeader(nil)
	}
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
