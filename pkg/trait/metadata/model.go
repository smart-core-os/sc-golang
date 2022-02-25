package metadata

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func (m *Model) UpdateTraitMetadata(traitMetadata *traits.TraitMetadata, opts ...resource.WriteOption) (*traits.Metadata, error) {
	// update traits and merge equivalently named traits more metadata field.
	opts = append([]resource.WriteOption{resource.WithUpdatePaths("traits"), resource.InterceptBefore(func(old, new proto.Message) {
		oldT := old.(*traits.Metadata)
		newT := new.(*traits.Metadata)
		newT.Traits = oldT.Traits
		for _, trait := range newT.Traits {
			if trait.Name == traitMetadata.Name {
				if trait.More == nil {
					trait.More = make(map[string]string)
				}
				for k, v := range traitMetadata.More {
					trait.More[k] = v
				}
				return
			}
		}
		// trait doesn't exist, add it
		newT.Traits = append(newT.Traits, traitMetadata)
	})}, opts...)
	return m.UpdateMetadata(&traits.Metadata{Traits: []*traits.TraitMetadata{traitMetadata}}, opts...)
}

func (m *Model) PullMetadata(ctx context.Context) <-chan *traits.PullMetadataResponse_Change {
	send := make(chan *traits.PullMetadataResponse_Change)

	// when ctx is cancelled, then the resource will close recv for us
	recv := m.metadata.Pull(ctx)
	go func() {
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
