package resource

import (
	"io"
	"math/rand"
	"time"

	"github.com/smart-core-os/sc-golang/pkg/cmp"
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
