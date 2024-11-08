package openclose

import (
	"strconv"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// DefaultModelOptions holds the default options for the model.
var DefaultModelOptions = []resource.Option{
	WithInitialOpenClosePositions(), // no positions
}

// ModelOption defined the base type for all options that apply to this traits model.
type ModelOption interface {
	resource.Option
	applyModel(args *modelArgs)
}

// WithOpenClosePositionsOption configures the positions resource of the model.
func WithOpenClosePositionsOption(opts ...resource.Option) resource.Option {
	return modelOptionFunc(func(args *modelArgs) {
		args.positionsOpts = append(args.positionsOpts, opts...)
	})
}

// WithInitialOpenClosePositions returns an option that configures the model to initialise with the given positions.
// This option should only be used once per model.
func WithInitialOpenClosePositions(positions ...*traits.OpenClosePosition) resource.Option {
	var opts []resource.Option
	for i, state := range positions {
		opts = append(opts, resource.WithInitialRecord(strconv.Itoa(i), state))
	}
	return WithOpenClosePositionsOption(resource.Options(opts))
}

// WithPreset returns an option that configures the model with the given preset.
func WithPreset(desc *traits.OpenClosePositions_Preset, positions ...*traits.OpenClosePosition) resource.Option {
	sortPositions(positions)
	return modelOptionFunc(func(args *modelArgs) {
		args.presets = append(args.presets, preset{desc: desc, positions: positions})
	})
}

func calcModelArgs(opts ...resource.Option) modelArgs {
	args := new(modelArgs)
	args.apply(DefaultModelOptions...)
	args.apply(opts...)
	return *args
}

type modelArgs struct {
	positionsOpts []resource.Option
	presets       []preset
}

func (a *modelArgs) apply(opts ...resource.Option) {
	for _, opt := range opts {
		if v, ok := opt.(ModelOption); ok {
			v.applyModel(a)
			continue
		}
		a.positionsOpts = append(a.positionsOpts, opt)
	}
}

func modelOptionFunc(fn func(args *modelArgs)) ModelOption {
	return modelOption{resource.EmptyOption{}, fn}
}

type modelOption struct {
	resource.Option
	fn func(args *modelArgs)
}

func (m modelOption) applyModel(args *modelArgs) {
	m.fn(args)
}
