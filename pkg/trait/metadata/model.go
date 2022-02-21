package metadata

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/memory"
	"google.golang.org/protobuf/proto"
)

type Model struct {
	metadata *memory.Resource // of traits.Metadata
}

func NewModel(opts ...memory.ResourceOption) *Model {
	defaultOpts := []memory.ResourceOption{memory.WithInitialValue(&traits.Metadata{})}
	return &Model{
		metadata: memory.NewResource(append(defaultOpts, opts...)...),
	}
}

func (m *Model) GetMetadata(opts ...memory.GetOption) (*traits.Metadata, error) {
	res := m.metadata.Get(opts...)
	return res.(*traits.Metadata), nil
}

func (m *Model) UpdateMetadata(metadata *traits.Metadata, opts ...memory.UpdateOption) (*traits.Metadata, error) {
	res, err := m.metadata.Set(metadata, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*traits.Metadata), nil
}

func (m *Model) UpdateTraitMetadata(traitMetadata *traits.TraitMetadata, opts ...memory.UpdateOption) (*traits.Metadata, error) {
	// update traits and merge equivalently named traits more metadata field.
	opts = append([]memory.UpdateOption{memory.WithUpdatePaths("traits"), memory.InterceptBefore(func(old, new proto.Message) {
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

	// when done is called, then the resource will close recv for us
	recv, done := m.metadata.OnUpdate(ctx)
	go func() {
		defer done()
		for {
			select {
			case <-ctx.Done():
				return
			case change, ok := <-recv:
				if !ok {
					return
				}
				select {
				case <-ctx.Done():
					return
				case send <- metadataChangeToProto(change):
				}
			}
		}
	}()

	return send
}
func metadataChangeToProto(change *memory.ResourceChange) *traits.PullMetadataResponse_Change {
	return &traits.PullMetadataResponse_Change{
		ChangeTime: change.ChangeTime,
		Metadata:   change.Value.(*traits.Metadata),
	}
}
