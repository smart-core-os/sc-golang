package booking

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/go-cmp/cmp"
	"github.com/smart-core-os/sc-api/go/traits"
	scTime "github.com/smart-core-os/sc-api/go/types/time"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

var randIds = []string{
	"Uv38ByGCZU8WP18PmmIdcg==",
}

func TestBookingApi_CreateBooking(t *testing.T) {
	api := NewMemoryDevice()
	api.Rng.Seed(1)
	ctx := context.Background()
	res, err := api.CreateBooking(ctx, &traits.CreateBookingRequest{
		Name: "room1",
		Booking: &traits.Booking{
			Bookable: "room1",
			Title:    "My Bookings",
			Booked: &scTime.Period{
				StartTime: &timestamp.Timestamp{},
				EndTime:   &timestamp.Timestamp{},
			},
		},
	})

	if err != nil {
		t.Errorf("error %s", err)
	}

	if res.BookingId != randIds[0] {
		t.Errorf("expected new id == %v; got %s", randIds[0], res.BookingId)
	}
}

func TestBookingApi_CheckInBooking(t *testing.T) {
	api := NewMemoryDevice()
	api.Rng.Seed(1)
	ctx := context.Background()
	createRes, err := api.CreateBooking(ctx, &traits.CreateBookingRequest{
		Name: "room1",
		Booking: &traits.Booking{
			Bookable: "room1",
			Title:    "My Bookings",
			Booked: &scTime.Period{
				StartTime: &timestamp.Timestamp{},
				EndTime:   &timestamp.Timestamp{},
			},
		},
	})

	if err != nil {
		t.Errorf("error %s", err)
	}

	_, err = api.CheckInBooking(ctx, &traits.CheckInBookingRequest{
		Name:      "room1",
		BookingId: createRes.BookingId,
		Time:      &timestamp.Timestamp{Seconds: 5},
	})

	if err != nil {
		t.Errorf("error %s", err)
	}
}

func TestBookingApi_CheckOutBooking(t *testing.T) {
	api := NewMemoryDevice()
	api.Rng.Seed(1)
	ctx := context.Background()
	createRes, err := api.CreateBooking(ctx, &traits.CreateBookingRequest{
		Name: "room1",
		Booking: &traits.Booking{
			Bookable: "room1",
			Title:    "My Bookings",
			Booked: &scTime.Period{
				StartTime: &timestamp.Timestamp{},
				EndTime:   &timestamp.Timestamp{},
			},
		},
	})

	if err != nil {
		t.Errorf("error %s", err)
	}

	_, err = api.CheckOutBooking(ctx, &traits.CheckOutBookingRequest{
		Name:      "room1",
		BookingId: createRes.BookingId,
		Time:      &timestamp.Timestamp{Seconds: 5},
	})

	if err != nil {
		t.Errorf("error %s", err)
	}
}

func TestBookingApi_UpdateBooking(t *testing.T) {
	api := NewMemoryDevice()
	api.Rng.Seed(1)
	ctx := context.Background()
	booking := &traits.Booking{
		Bookable: "room1",
		Title:    "My Bookings",
		Booked: &scTime.Period{
			StartTime: &timestamp.Timestamp{},
			EndTime:   &timestamp.Timestamp{},
		},
	}
	createRes, err := api.CreateBooking(ctx, &traits.CreateBookingRequest{
		Name:    "room1",
		Booking: booking,
	})

	if err != nil {
		t.Errorf("error %s", err)
	}

	res, err := api.UpdateBooking(ctx, &traits.UpdateBookingRequest{
		Name: "room1",
		Booking: &traits.Booking{
			Id: createRes.BookingId,
			Booked: &scTime.Period{
				StartTime: &timestamp.Timestamp{Seconds: 5},
				EndTime:   &timestamp.Timestamp{Seconds: 10},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{
			"booked.start_time",
			"booked.end_time",
		}},
	})

	if err != nil {
		t.Errorf("error %s", err)
	}
	if res.Booking == booking {
		t.Errorf("expected res.Booking != booking")
	}
	expected := &traits.Booking{
		Bookable: "room1",
		Title:    "My Bookings",
		Id:       createRes.BookingId,
		Booked: &scTime.Period{
			StartTime: &timestamp.Timestamp{Seconds: 5},
			EndTime:   &timestamp.Timestamp{Seconds: 10},
		},
	}

	if diff := cmp.Diff(expected, res.Booking, protocmp.Transform()); diff != "" {
		t.Errorf("response mismatch (-want, +got)\n%v", diff)
	}
}

func TestBookingApi_ListBookings(t *testing.T) {
	api := NewMemoryDevice()
	api.Rng.Seed(1)
	ctx := context.Background()
	b1 := &traits.Booking{
		Id:       "1",
		Bookable: "room1",
		Title:    "My Bookings",
		Booked: &scTime.Period{
			StartTime: &timestamp.Timestamp{},
			EndTime:   &timestamp.Timestamp{},
		},
	}
	b2 := &traits.Booking{
		Id:       "2",
		Bookable: "room1",
		Title:    "My Bookings",
		Booked: &scTime.Period{
			StartTime: &timestamp.Timestamp{Seconds: 5},
			EndTime:   &timestamp.Timestamp{Seconds: 10},
		},
	}
	b3 := &traits.Booking{
		Id:       "3",
		Bookable: "room1",
		Title:    "My Bookings",
		Booked: &scTime.Period{
			StartTime: &timestamp.Timestamp{Seconds: 6},
			EndTime:   &timestamp.Timestamp{Seconds: 15},
		},
	}
	_, err := api.CreateBooking(ctx, &traits.CreateBookingRequest{
		Name:    "room1",
		Booking: b1,
	})
	_, err = api.CreateBooking(ctx, &traits.CreateBookingRequest{
		Name:    "room1",
		Booking: b2,
	})
	_, err = api.CreateBooking(ctx, &traits.CreateBookingRequest{
		Name:    "room1",
		Booking: b3,
	})

	if err != nil {
		t.Errorf("error %s", err)
	}

	res, err := api.ListBookings(ctx, &traits.ListBookingsRequest{
		Name: "room1",
		BookingIntersects: &scTime.Period{
			StartTime: &timestamp.Timestamp{Seconds: 2},
			EndTime:   &timestamp.Timestamp{Seconds: 6},
		},
	})
	expected := []*traits.Booking{
		b2,
	}

	if err != nil {
		t.Errorf("error %s", err)
	}
	if !reflect.DeepEqual(res.Bookings, expected) {
		t.Errorf("expected %v == %v", res.Bookings, expected)
	}
}
