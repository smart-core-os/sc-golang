package booking

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// Model models the Booking trait.
type Model struct {
	bookings *resource.Collection // Of *traits.Booking
}

// NewModel creates a new Model without any bookings.
func NewModel() *Model {
	return &Model{
		bookings: resource.NewCollection(),
	}
}

func (m *Model) ListBookings(opts ...resource.ReadOption) []*traits.Booking {
	msgs := m.bookings.List(opts...)
	res := make([]*traits.Booking, len(msgs))
	for i, msg := range msgs {
		res[i] = msg.(*traits.Booking)
	}
	return res
}

func (m *Model) CreateBooking(booking *traits.Booking) (*traits.Booking, error) {
	var msg proto.Message
	var err error
	if booking.Id == "" {
		msg, err = m.bookings.AddFn(func(id string) proto.Message {
			booking.Id = id
			return booking
		})
	} else {
		msg, err = m.bookings.Add(booking.Id, booking)
	}
	if msg == nil {
		return nil, err
	}
	return msg.(*traits.Booking), err
}

func (m *Model) UpdateBooking(booking *traits.Booking, opts ...resource.WriteOption) (*traits.Booking, error) {
	if booking.Id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "missing booking.id")
	}

	newVal, err := m.bookings.Update(booking.Id, booking, opts...)
	if newVal == nil {
		return nil, err
	}
	return newVal.(*traits.Booking), err
}

//goland:noinspection GoNameStartsWithPackageName
type BookingChange struct {
	ChangeTime time.Time
	ChangeType types.ChangeType

	OldValue, NewValue *traits.Booking
}

func (m *Model) PullBookings(ctx context.Context, opts ...resource.ReadOption) <-chan BookingChange {
	send := make(chan BookingChange)

	go func() {
		defer close(send)
		for change := range m.bookings.Pull(ctx, opts...) {
			event := BookingChange{
				ChangeTime: change.ChangeTime,
				ChangeType: change.ChangeType,
			}
			if change.OldValue != nil {
				event.OldValue = change.OldValue.(*traits.Booking)
			}
			if change.NewValue != nil {
				event.NewValue = change.NewValue.(*traits.Booking)
			}

			select {
			case <-ctx.Done():
				return
			case send <- event:
			}
		}
	}()

	return send
}
