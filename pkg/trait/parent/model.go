package parent

import (
	"context"
	"fmt"
	"sort"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/time/clock"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Model models a collection of children.
type Model struct {
	children *resource.Collection
}

func NewModel() *Model {
	return &Model{
		children: resource.NewCollection(resource.WithClock(clock.Real())),
	}
}

// AddChild inserts a child into this model.
// If a child with the given Child.Name already exists, no changes will be made.
// Panics if child.Traits are not sorted in ascending order, required by AddChildTrait.
func (m *Model) AddChild(child *traits.Child) {
	// force traits to be sorted
	if !sort.SliceIsSorted(child.Traits, func(i, j int) bool {
		return child.Traits[i].Name < child.Traits[j].Name
	}) {
		panic(fmt.Errorf("%v Traits are not sorted: %v", child.Name, child.Traits))
	}
	_, _ = m.children.Add(child.Name, child)
}

// RemoveChildByName removes a child from this model matching the given name.
// If no such child exists then nil, false will be returned, else the removed child and true.
func (m *Model) RemoveChildByName(name string) (child *traits.Child, existed bool) {
	old := m.children.Delete(name)
	if old == nil {
		return nil, false
	}
	return old.(*traits.Child), true
}

// AddChildTrait ensures that a child with the given name and list of trait names exists in this model.
// If no child with the given name is already know, one will be created.
// If a child is already known with the given name, its traits will be unioned with the given trait names.
func (m *Model) AddChildTrait(name string, traitName ...trait.Name) (child *traits.Child, created bool) {
	msg, err := m.children.Update(name, &traits.Child{Name: name},
		resource.WithCreateIfAbsent(),
		resource.WithCreatedCallback(func() {
			created = true
		}),
		resource.InterceptBefore(func(old, value proto.Message) {
			oldChild := old.(*traits.Child)
			newChild := value.(*traits.Child)
			newChild.Traits = traitUnion(oldChild.Traits, traitName...)
		}))
	if err != nil {
		panic(err) // shouldn't happen
	}
	return msg.(*traits.Child), created
}

// ListChildren returns a slice of all known Child instances.
func (m *Model) ListChildren() []*traits.Child {
	msgs := m.children.List()
	children := make([]*traits.Child, len(msgs))
	for i, msg := range msgs {
		children[i] = msg.(*traits.Child)
	}
	return children
}

// PullChildren returns a chat that will emit when changes are made to the known children of this model.
func (m *Model) PullChildren(ctx context.Context) <-chan *traits.PullChildrenResponse_Change {
	out := make(chan *traits.PullChildrenResponse_Change)
	changes := m.children.Pull(ctx)

	go func() {
		for change := range changes {
			out <- childrenChangeToProto(change)
		}
	}()

	return out
}

func childrenChangeToProto(change *resource.CollectionChange) *traits.PullChildrenResponse_Change {
	pChange := traits.PullChildrenResponse_Change{
		Type:       change.ChangeType,
		ChangeTime: timestamppb.New(change.ChangeTime),
	}
	if change.OldValue != nil {
		pChange.OldValue = change.OldValue.(*traits.Child)
	}
	if change.NewValue != nil {
		pChange.NewValue = change.NewValue.(*traits.Child)
	}
	return &pChange
}

// traitUnion returns a slice containing the union of has and more (converted to *traits.Trait).
// The has slice should be sorted in ascending order by Trait.Name.
// The returned slice will be sorted in ascending order by Trait.Name.
func traitUnion(has []*traits.Trait, more ...trait.Name) []*traits.Trait {
	// has should be sorted by Trait.Name
	for _, t := range more {
		ts := string(t)
		insertIndex := sort.Search(len(has), func(i int) bool {
			return has[i].Name >= ts
		})
		switch {
		case insertIndex == len(has): // append to end
			has = append(has, &traits.Trait{Name: ts})
		case has[insertIndex].Name == ts: // already exists, do nothing
		default: // insert into slice
			has = append(has[:insertIndex+1], has[insertIndex:]...)
			has[insertIndex] = &traits.Trait{Name: ts}
		}
	}
	return has
}
