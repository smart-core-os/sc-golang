package resource

import (
	"google.golang.org/protobuf/proto"
)

type Message interface {
	proto.Message
	comparable
}

func zero[T Message]() T {
	var z T
	return z
}
