package resource

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
)

func TestWithUpdatesOnly(t *testing.T) {
	t.Parallel()

	t.Run("Value (default)", func(t *testing.T) {
		v := NewValue(WithInitialValue(&traits.OnOff{State: traits.OnOff_ON}))
		ctx, done := context.WithCancel(context.Background())
		t.Cleanup(done)
		c := v.Pull(ctx)
		var events []*ValueChange
		complete := make(chan struct{})
		go func() {
			defer close(complete)
			for change := range c {
				events = append(events, change)
			}
		}()

		_, err := v.Set(&traits.OnOff{State: traits.OnOff_OFF})
		if err != nil {
			t.Fatal(err)
		}

		time.AfterFunc(10*time.Millisecond, done)
		<-complete // wait for the inner go routine to complete

		got := make([]proto.Message, len(events))
		for i, event := range events {
			got[i] = event.Value
		}
		want := []proto.Message{
			&traits.OnOff{State: traits.OnOff_ON},  // initial value
			&traits.OnOff{State: traits.OnOff_OFF}, // update value
		}

		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("Incorrect events (-want, +got)\n%v", diff)
		}
	})
	t.Run("Value (updates only)", func(t *testing.T) {
		v := NewValue(WithInitialValue(&traits.OnOff{State: traits.OnOff_ON}))
		ctx, done := context.WithCancel(context.Background())
		t.Cleanup(done)
		c := v.Pull(ctx, WithUpdatesOnly(true))
		var events []*ValueChange
		complete := make(chan struct{})
		go func() {
			defer close(complete)
			for change := range c {
				events = append(events, change)
			}
		}()

		_, err := v.Set(&traits.OnOff{State: traits.OnOff_OFF})
		if err != nil {
			t.Fatal(err)
		}

		time.AfterFunc(10*time.Millisecond, done)
		<-complete // wait for the inner go routine to complete

		got := make([]proto.Message, len(events))
		for i, event := range events {
			got[i] = event.Value
		}
		want := []proto.Message{
			// no initial value
			&traits.OnOff{State: traits.OnOff_OFF}, // update value
		}

		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("Incorrect events (-want, +got)\n%v", diff)
		}
	})

	t.Run("Collection (default)", func(t *testing.T) {
		v := NewCollection()
		add(t, v, "A", &traits.OnOff{State: traits.OnOff_ON})

		ctx, done := context.WithCancel(context.Background())
		t.Cleanup(done)
		c := v.Pull(ctx)
		var events []*CollectionChange
		complete := make(chan struct{})
		go func() {
			defer close(complete)
			for change := range c {
				events = append(events, change)
			}
		}()

		_, err := v.Update("A", &traits.OnOff{State: traits.OnOff_OFF})
		if err != nil {
			t.Fatal(err)
		}

		time.AfterFunc(10*time.Millisecond, done)
		<-complete // wait for the inner go routine to complete

		got := make([]collectionChange, len(events))
		for i, event := range events {
			got[i] = collectionChange{Id: event.Id, OldValue: event.OldValue, NewValue: event.NewValue, ChangeType: event.ChangeType}
		}
		want := []collectionChange{
			{Id: "A", OldValue: nil, NewValue: &traits.OnOff{State: traits.OnOff_ON}, ChangeType: types.ChangeType_ADD},
			{Id: "A", OldValue: &traits.OnOff{State: traits.OnOff_ON}, NewValue: &traits.OnOff{State: traits.OnOff_OFF}, ChangeType: types.ChangeType_UPDATE},
		}

		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("Incorrect events (-want, +got)\n%v", diff)
		}
	})
	t.Run("Collection (updates only)", func(t *testing.T) {
		v := NewCollection()
		add(t, v, "A", &traits.OnOff{State: traits.OnOff_ON})

		ctx, done := context.WithCancel(context.Background())
		t.Cleanup(done)
		c := v.Pull(ctx, WithUpdatesOnly(true))
		var events []*CollectionChange
		complete := make(chan struct{})
		go func() {
			defer close(complete)
			for change := range c {
				events = append(events, change)
			}
		}()

		_, err := v.Update("A", &traits.OnOff{State: traits.OnOff_OFF})
		if err != nil {
			t.Fatal(err)
		}

		time.AfterFunc(10*time.Millisecond, done)
		<-complete // wait for the inner go routine to complete

		got := make([]collectionChange, len(events))
		for i, event := range events {
			got[i] = collectionChange{Id: event.Id, OldValue: event.OldValue, NewValue: event.NewValue, ChangeType: event.ChangeType}
		}
		want := []collectionChange{
			{Id: "A", OldValue: &traits.OnOff{State: traits.OnOff_ON}, NewValue: &traits.OnOff{State: traits.OnOff_OFF}, ChangeType: types.ChangeType_UPDATE},
		}

		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("Incorrect events (-want, +got)\n%v", diff)
		}
	})
}

func TestWithInclude(t *testing.T) {
	t.Run("List", func(t *testing.T) {
		c := NewCollection()
		add(t, c, "A", &traits.OnOff{State: traits.OnOff_ON})
		add(t, c, "B", &traits.OnOff{State: traits.OnOff_OFF})
		add(t, c, "C", &traits.OnOff{State: traits.OnOff_STATE_UNSPECIFIED})

		t.Run("id filter", func(t *testing.T) {
			got := c.List(WithInclude(func(id string, item proto.Message) bool {
				return id == "B" || id == "C"
			}))
			want := []proto.Message{
				&traits.OnOff{State: traits.OnOff_OFF},
				&traits.OnOff{State: traits.OnOff_STATE_UNSPECIFIED},
			}
			if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
				t.Fatalf("(-want,+got)\n%v", diff)
			}
		})

		t.Run("body filter", func(t *testing.T) {
			got := c.List(WithInclude(func(id string, item proto.Message) bool {
				itemVal := item.(*traits.OnOff)
				return itemVal.State != traits.OnOff_STATE_UNSPECIFIED
			}))
			want := []proto.Message{
				&traits.OnOff{State: traits.OnOff_ON},
				&traits.OnOff{State: traits.OnOff_OFF},
			}
			if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
				t.Fatalf("(-want,+got)\n%v", diff)
			}
		})
	})

	t.Run("Pull", func(t *testing.T) {
		v := NewCollection()
		add(t, v, "A", &traits.OnOff{State: traits.OnOff_ON})
		add(t, v, "B", &traits.OnOff{State: traits.OnOff_OFF})
		add(t, v, "C", &traits.OnOff{State: traits.OnOff_STATE_UNSPECIFIED})

		ctx, done := context.WithCancel(context.Background())
		t.Cleanup(done)

		// pull only items that are off
		c := v.Pull(ctx, WithInclude(func(_ string, item proto.Message) bool {
			itemVal := item.(*traits.OnOff)
			return itemVal.State == traits.OnOff_OFF
		}))
		var events []*CollectionChange
		complete := make(chan struct{})
		go func() {
			defer close(complete)
			for change := range c {
				events = append(events, change)
			}
		}()

		_, err := v.Update("A", &traits.OnOff{State: traits.OnOff_OFF})
		if err != nil {
			t.Fatal(err)
		}
		_, err = v.Update("B", &traits.OnOff{State: traits.OnOff_ON})
		if err != nil {
			t.Fatal(err)
		}

		time.AfterFunc(10*time.Millisecond, done)
		<-complete // wait for the inner go routine to complete

		got := make([]collectionChange, len(events))
		for i, event := range events {
			got[i] = collectionChange{Id: event.Id, OldValue: event.OldValue, NewValue: event.NewValue, ChangeType: event.ChangeType}
		}
		want := []collectionChange{
			{Id: "B", NewValue: &traits.OnOff{State: traits.OnOff_OFF}, ChangeType: types.ChangeType_ADD},
			{Id: "A", NewValue: &traits.OnOff{State: traits.OnOff_OFF}, ChangeType: types.ChangeType_ADD},
			{Id: "B", OldValue: &traits.OnOff{State: traits.OnOff_OFF}, ChangeType: types.ChangeType_REMOVE},
		}

		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("Incorrect events (-want, +got)\n%v", diff)
		}
	})
}

func TestWithBackpressure_False(t *testing.T) {
	val := NewValue(WithInitialValue(&traits.OnOff{}))

	t.Run("false", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// with backpressure disabled, we can open a Pull, fail to receive, and it doesn't block
		_ = val.Pull(ctx, WithBackpressure(false))
		success := make(chan struct{})
		go func() {
			defer close(success)

			// do a set call, which shouldn't block or error
			_, err := val.Set(&traits.OnOff{State: traits.OnOff_OFF})
			if err != nil {
				t.Error(err)
			}
		}()

		select {
		case <-success:
		case <-time.After(100 * time.Millisecond):
			t.Error("calls blocked")
		}
	})

	t.Run("true", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// with backpressure enabled, we can open a Pull, fail to receive, and it will block calls to Set
		_ = val.Pull(ctx, WithBackpressure(true))
		completed := make(chan struct{})
		go func() {
			defer close(completed)

			// do a set call, which should block
			_, err := val.Set(&traits.OnOff{State: traits.OnOff_OFF})
			if err != nil {
				t.Error(err)
			}
		}()

		select {
		case <-completed:
			t.Error("expected call to Set to block")
		case <-time.After(100 * time.Millisecond):
		}
	})
}

func TestWithIDInterceptor(t *testing.T) {
	c := NewCollection(WithIDInterceptor(strings.ToLower))
	add(t, c, "A", &traits.OnOff{State: traits.OnOff_ON})
	expect := &traits.OnOff{State: traits.OnOff_ON}
	actual, ok := c.Get("a")
	if !ok {
		t.Error("expected to find item with id 'a'")
	}
	if diff := cmp.Diff(expect, actual, protocmp.Transform()); diff != "" {
		t.Errorf("Get('a') returned wrong value (-want,+got)\n%v", diff)
	}
}

// List CollectionChange but without the timestamp
type collectionChange struct {
	Id                 string
	OldValue, NewValue proto.Message
	ChangeType         types.ChangeType
}

func add(t *testing.T, c *Collection, id string, msg proto.Message) {
	t.Helper()
	_, err := c.Add(id, msg)
	if err != nil {
		t.Fatal(err)
	}
}
