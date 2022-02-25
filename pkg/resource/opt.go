package resource

import (
	"io"
	"math/rand"
	"time"

	"github.com/smart-core-os/sc-golang/pkg/cmp"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// Option configures a resource value or collection.
type Option interface {
	apply(s *config)
}

// WithClock configures the clock used when time is needed.
// Defaults to a Clock backed by the time package.
func WithClock(c Clock) Option {
	return optionFunc(func(s *config) {
		s.clock = c
	})
}

// WithEquivalence configures how consecutive emissions are compared, equivalent emissions are not emitted.
// Defaults to nil, no equivalence checking is performed, all events will be emitted.
func WithEquivalence(e Comparer) Option {
	return optionFunc(func(s *config) {
		s.equivalence = e
	})
}

// WithMessageEquivalence is like WithEquivalence but using a cmp.Message.
func WithMessageEquivalence(e cmp.Message) Option {
	return WithEquivalence(ComparerFunc(e))
}

// WithNoDuplicates is like WithMessageEquivalence(cmp.Equal()).
func WithNoDuplicates() Option {
	return WithMessageEquivalence(cmp.Equal())
}

// WithRNG configures the source of randomness for the resource.
// Defaults to rand.Rand with a time seed.
func WithRNG(rng io.Reader) Option {
	return optionFunc(func(s *config) {
		s.rng = rng
	})
}

// WithInitialValue configures the initial value for the resource.
// Applies only to Value.
func WithInitialValue(initialValue proto.Message) Option {
	return optionFunc(func(s *config) {
		s.initialValue = initialValue
	})
}

// WithWritableFields configures write operations on the resource to accept updates to the given fields only.
// Explicit writes to fields not in this mask will fail.
func WithWritableFields(mask *fieldmaskpb.FieldMask) Option {
	return optionFunc(func(s *config) {
		s.writableFields = mask
	})
}

// WithWritablePaths is like WithWritableFields using fieldmaskpb.New.
func WithWritablePaths(m proto.Message, paths ...string) Option {
	mask, err := fieldmaskpb.New(m, paths...)
	if err != nil {
		panic(err)
	}
	return WithWritableFields(mask)
}

type config struct {
	clock          Clock
	equivalence    Comparer
	rng            io.Reader
	initialValue   proto.Message
	writableFields *fieldmaskpb.FieldMask
}

func computeConfig(opts ...Option) *config {
	c := &config{
		clock: WallClock(),
		rng:   rand.New(rand.NewSource(time.Now().Unix())),
	}
	for _, opt := range opts {
		opt.apply(c)
	}
	return c
}

type optionFunc func(s *config)

func (f optionFunc) apply(s *config) {
	f(s)
}

// ReadOption configures settings for reading data.
type ReadOption interface {
	apply(rr *readRequest)
}

// WithReadMask configures the properties that will be filled in the response value.
func WithReadMask(mask *fieldmaskpb.FieldMask) ReadOption {
	return readOptionFunc(func(rr *readRequest) {
		rr.readMask = mask
	})
}

// WithReadPaths configures the properties that will be filled in the response value.
// Panics if paths aren't part of m.
func WithReadPaths(m proto.Message, paths ...string) ReadOption {
	mask, err := fieldmaskpb.New(m, paths...)
	if err != nil {
		panic(err)
	}
	return WithReadMask(mask)
}

func computeReadConfig(opts ...ReadOption) *readRequest {
	rr := &readRequest{}
	for _, opt := range opts {
		opt.apply(rr)
	}
	return rr
}

type readRequest struct {
	readMask *fieldmaskpb.FieldMask
}

// ResponseFilter returns a masks.ResponseFilter configured using this readRequest properties.
func (rr *readRequest) ResponseFilter() *masks.ResponseFilter {
	return masks.NewResponseFilter(masks.WithFieldMask(rr.readMask))
}

// FilterClone in the equivalent of rr.ResponseFilter().FilterClone(m).
func (rr *readRequest) FilterClone(m proto.Message) proto.Message {
	return rr.ResponseFilter().FilterClone(m)
}

type readOptionFunc func(rr *readRequest)

func (r readOptionFunc) apply(rr *readRequest) {
	r(rr)
}
