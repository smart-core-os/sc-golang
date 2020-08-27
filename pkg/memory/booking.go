package memory

import (
	"context"
	"math/rand"
	"sort"
	"sync"

	"git.vanti.co.uk/smartcore/sc-api/go/device/traits"
	"git.vanti.co.uk/smartcore/sc-api/go/types"
	"git.vanti.co.uk/smartcore/sc-api/go/types/time"
	"github.com/iancoleman/strcase"
	fieldMaskUtils "github.com/mennanov/fieldmask-utils"
	"github.com/olebedev/emitter"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	time2 "git.vanti.co.uk/smartcore/sc-golang/pkg/time"
)

type BookingApi struct {
	bookingsById   map[int32]*traits.Booking
	bookingsByIdMu sync.RWMutex
	// emits the "change" event with a single Arg of type *traits.PullBookingsResponse_Change
	bus *emitter.Emitter
	// used for generating ids
	Rng *rand.Rand
}

func NewBookingApi() *BookingApi {
	return &BookingApi{
		bookingsById: make(map[int32]*traits.Booking),
		bus:          &emitter.Emitter{},
		Rng:          rand.New(rand.NewSource(rand.Int63())),
	}
}

func (b *BookingApi) ListBookings(ctx context.Context, request *traits.ListBookingsRequest) (*traits.ListBookingsResponse, error) {
	response := &traits.ListBookingsResponse{}
	b.bookingsByIdMu.RLock()
	for _, booking := range b.bookingsById {
		if bookingMatches(booking, request) {
			response.Bookings = append(response.Bookings, booking)
		}
	}
	b.bookingsByIdMu.RUnlock()
	sort.Slice(response.Bookings, func(i, j int) bool {
		return response.Bookings[i].Id < response.Bookings[j].Id
	})
	return response, nil
}

func (b *BookingApi) CheckInBooking(ctx context.Context, request *traits.CheckInBookingRequest) (*traits.CheckInBookingResponse, error) {
	_, err := b.applyChange(request.BookingId, func(oldBooking, newBooking *traits.Booking) error {
		if newBooking.CheckIn == nil {
			newBooking.CheckIn = &time.Period{}
		}
		newBooking.CheckIn.StartTime = request.Time
		if newBooking.CheckIn.StartTime == nil {
			newBooking.CheckIn.StartTime = timestamppb.Now()
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &traits.CheckInBookingResponse{}, nil
}

func (b *BookingApi) CheckOutBooking(ctx context.Context, request *traits.CheckOutBookingRequest) (*traits.CheckOutBookingResponse, error) {
	_, err := b.applyChange(request.BookingId, func(oldBooking, newBooking *traits.Booking) error {
		if newBooking.CheckIn == nil {
			newBooking.CheckIn = &time.Period{}
		}
		newBooking.CheckIn.EndTime = request.Time
		if newBooking.CheckIn.EndTime == nil {
			newBooking.CheckIn.EndTime = timestamppb.Now()
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &traits.CheckOutBookingResponse{}, nil
}

func (b *BookingApi) CreateBooking(ctx context.Context, request *traits.CreateBookingRequest) (*traits.CreateBookingResponse, error) {
	if request.Booking == nil {
		return nil, status.Error(codes.InvalidArgument, "missing booking")
	}
	id := request.Booking.Id
	idGenerated := id == 0

	if id == 0 {
		// try to generate a unique id
		b.bookingsByIdMu.RLock()
		for i := 0; i < 10; i++ {
			idCandidate := b.Rng.Int31()
			if _, exists := b.bookingsById[idCandidate]; idCandidate != 0 && !exists {
				id = idCandidate
				break
			}
		}
		b.bookingsByIdMu.RUnlock()
	}

	if id == 0 {
		// no id can be generated, return an error
		return nil, status.Error(codes.Aborted, "id generation attempts exhausted")
	}

	// save the new booking
	b.bookingsByIdMu.Lock()
	defer b.bookingsByIdMu.Unlock()

	if _, exists := b.bookingsById[id]; exists {
		if idGenerated {
			return nil, status.Error(codes.Aborted, "generated id concurrently taken")
		} else {
			// user supplied id already exists, error
			return nil, status.Errorf(codes.AlreadyExists, "booking %v", id)
		}
	}

	request.Booking.Id = id
	b.bookingsById[id] = request.Booking
	b.bus.Emit("change", &traits.PullBookingsResponse_Change{
		Type:     types.ChangeType_ADD,
		NewValue: request.Booking,
	})

	return &traits.CreateBookingResponse{BookingId: id}, nil
}

func (b *BookingApi) UpdateBooking(ctx context.Context, request *traits.UpdateBookingRequest) (*traits.UpdateBookingResponse, error) {
	if request.Booking == nil {
		return nil, status.Error(codes.InvalidArgument, "missing booking")
	}

	id := request.Booking.Id
	if id == 0 {
		return nil, status.Error(codes.InvalidArgument, "missing booking.id")
	}

	var mask fieldMaskUtils.Mask
	if request.UpdateMask != nil && len(request.UpdateMask.Paths) > 0 {
		if !request.UpdateMask.IsValid(request.Booking) {
			return nil, status.Error(codes.InvalidArgument, "invalid field_mask")
		}

		var err error
		mask, err = fieldMaskUtils.MaskFromPaths(request.UpdateMask.Paths, strcase.ToCamel)
		if err != nil {
			return nil, err
		}
	}

	change, err := b.applyChange(id, func(oldBooking, newBooking *traits.Booking) error {
		if mask != nil {
			// apply only selected fields
			return fieldMaskUtils.StructToStruct(mask, request.Booking, newBooking)
		} else {
			// replace the booking data

			proto.Reset(newBooking)
			proto.Merge(newBooking, request.Booking)
			return nil
		}
	})
	if err != nil {
		return nil, err
	}
	return &traits.UpdateBookingResponse{
		Booking: change,
	}, nil
}

func (b *BookingApi) PullBookings(request *traits.ListBookingsRequest, server traits.BookingApi_PullBookingsServer) error {
	changes := b.bus.On("change")
	defer b.bus.Off("change", changes)

	// send all the bookings we already know about
	currentBookings, err := b.ListBookings(server.Context(), request)
	if err != nil {
		return err
	}
	initialResponse := &traits.PullBookingsResponse{}
	for _, booking := range currentBookings.Bookings {
		initialResponse.Changes = append(initialResponse.Changes, &traits.PullBookingsResponse_Change{
			Type:     types.ChangeType_ADD,
			NewValue: booking,
		})
	}
	if len(initialResponse.Changes) > 0 {
		if err := server.Send(initialResponse); err != nil {
			return err
		}
	}

	// watch for changes to the bookings list and emit when one matches our query
	for {
		select {
		case <-server.Context().Done():
			return nil
		case event := <-changes:
			change := event.Args[0].(*traits.PullBookingsResponse_Change)
			sentChange := bookingChangeForQuery(request, change)
			// else an update happened but wasn't included in this query
			if sentChange != nil {
				if err := server.Send(&traits.PullBookingsResponse{Changes: []*traits.PullBookingsResponse_Change{
					sentChange,
				}}); err != nil {
					return err
				}
			}
		}
	}
}

func (b *BookingApi) applyChange(id int32, fn func(oldBooking, newBooking *traits.Booking) error) (*traits.Booking, error) {
	b.bookingsByIdMu.RLock()
	booking, exists := b.bookingsById[id]
	b.bookingsByIdMu.RUnlock()
	if !exists {
		return nil, status.Errorf(codes.NotFound, "unknown booking id %v", id)
	}

	newBooking := proto.Clone(booking).(*traits.Booking)
	if err := fn(booking, newBooking); err != nil {
		return nil, err
	}

	b.bookingsByIdMu.Lock()
	defer b.bookingsByIdMu.Unlock()
	bookingAgain := b.bookingsById[id]
	if booking != bookingAgain {
		return nil, status.Errorf(codes.Aborted, "concurrent update detected")
	}

	b.bookingsById[booking.Id] = newBooking
	b.bus.Emit("change", &traits.PullBookingsResponse_Change{
		Type:     types.ChangeType_UPDATE,
		OldValue: booking,
		NewValue: newBooking,
	})
	return newBooking, nil
}

// bookingChangeForQuery converts the given change to be relative to the query.
//
// For example the change might represent an update, but that update changes the inclusion of the booking in the query
// so instead of ChangeType_UPDATE it would be ChangeType_ADD or REMOVE relative to the query being processed.
func bookingChangeForQuery(query *traits.ListBookingsRequest, change *traits.PullBookingsResponse_Change) *traits.PullBookingsResponse_Change {
	var sentChange *traits.PullBookingsResponse_Change

	wasIncluded := bookingMatches(change.OldValue, query)
	isIncluded := bookingMatches(change.NewValue, query)

	if wasIncluded && !isIncluded {
		// removed from query
		sentChange = &traits.PullBookingsResponse_Change{
			Type:     types.ChangeType_REMOVE,
			OldValue: change.OldValue,
		}
	} else if !wasIncluded && isIncluded {
		// added to query
		sentChange = &traits.PullBookingsResponse_Change{
			Type:     types.ChangeType_ADD,
			NewValue: change.NewValue,
		}
	} else if wasIncluded && isIncluded {
		// is an update
		sentChange = change
	}
	return sentChange
}

func bookingMatches(b *traits.Booking, query *traits.ListBookingsRequest) bool {
	if b == nil {
		return false
	}
	if query.BookingIntersects != nil {
		return time2.PeriodsIntersect(b.Booked, query.BookingIntersects)
	}
	return true
}
