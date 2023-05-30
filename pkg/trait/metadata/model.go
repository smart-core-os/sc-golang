package metadata

import (
	"context"
	"sort"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	metadata *resource.Value // of traits.Metadata
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&traits.Metadata{})}
	return &Model{
		metadata: resource.NewValue(append(defaultOpts, opts...)...),
	}
}

func (m *Model) GetMetadata(opts ...resource.ReadOption) (*traits.Metadata, error) {
	res := m.metadata.Get(opts...)
	return res.(*traits.Metadata), nil
}

func (m *Model) UpdateMetadata(metadata *traits.Metadata, opts ...resource.WriteOption) (*traits.Metadata, error) {
	res, err := m.metadata.Set(metadata, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*traits.Metadata), nil
}

// MergeMetadata writes any present fields in metadata to the existing data.
// Traits that exist in the given metadata are merged with existing traits using
func (m *Model) MergeMetadata(metadata *traits.Metadata, opts ...resource.WriteOption) (*traits.Metadata, error) {
	newOpts := make([]resource.WriteOption, 1, len(opts)+1)
	newOpts[0] = resource.InterceptBefore(func(old, new proto.Message) {
		clean := proto.Clone(metadata).(*traits.Metadata)
		// handle trait updates specially
		cleanTraits := clean.Traits
		clean.Traits = nil

		proto.Merge(new, old)   // copy all the original values into new
		proto.Merge(new, clean) // then copy our updates - excluding Traits - on top

		// finally merge traits
		// The default proto.Merge logic is to append src slices to dst slices.
		// Instead we want to treat the Traits slice as if it were a map keyed by TraitMetadata.Name,
		// so we have to do it ourselves.
		oldVal := old.(*traits.Metadata)
		newVal := new.(*traits.Metadata)
		newVal.Traits = oldVal.Traits
		for _, trait := range cleanTraits {
			newVal.Traits = mergeTraitMetadata(newVal.Traits, trait)
		}
		// make the output consistent
		sort.Slice(newVal.Traits, func(i, j int) bool {
			return newVal.Traits[i].Name < newVal.Traits[j].Name
		})
	})
	newOpts = append(newOpts, opts...)
	return m.UpdateMetadata(metadata, newOpts...)
}

func (m *Model) UpdateTraitMetadata(traitMetadata *traits.TraitMetadata, opts ...resource.WriteOption) (*traits.Metadata, error) {
	return m.MergeMetadata(&traits.Metadata{Traits: []*traits.TraitMetadata{traitMetadata}}, opts...)
}

// mergeTraitMetadata merged tmd into tmds and returns the updated slice.
// If a trait with tmd.Name already exists in tmds then tmd will be merged into it.
// Otherwise tmd will be added appended to the slice.
func mergeTraitMetadata(tmds []*traits.TraitMetadata, tmd *traits.TraitMetadata) []*traits.TraitMetadata {
	// todo: this would be more efficient if tmds were sorted by Name, so figure out how to do that.

	if len(tmds) == 0 {
		return []*traits.TraitMetadata{tmd}
	}
	for _, trait := range tmds {
		if trait.Name == tmd.Name {
			proto.Merge(trait, tmd)
			return tmds
		}
	}
	// trait doesn't exist, add it
	return append(tmds, tmd)
}

func (m *Model) PullMetadata(ctx context.Context, opts ...resource.ReadOption) <-chan *traits.PullMetadataResponse_Change {
	send := make(chan *traits.PullMetadataResponse_Change)

	// when ctx is cancelled, then the resource will close recv for us
	recv := m.metadata.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			select {
			case <-ctx.Done():
				return
			case send <- metadataChangeToProto(change):
			}
		}
	}()

	return send
}
func metadataChangeToProto(change *resource.ValueChange) *traits.PullMetadataResponse_Change {
	return &traits.PullMetadataResponse_Change{
		ChangeTime: timestamppb.New(change.ChangeTime),
		Metadata:   change.Value.(*traits.Metadata),
	}
}
