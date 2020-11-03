package memory

import (
	"context"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	goTime "time"

	"git.vanti.co.uk/smartcore/sc-api/go/traits"
	"git.vanti.co.uk/smartcore/sc-api/go/types"
	"git.vanti.co.uk/smartcore/sc-api/go/types/time"
	"github.com/iancoleman/strcase"
	fieldMaskUtils "github.com/mennanov/fieldmask-utils"
	"github.com/olebedev/emitter"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	time2 "git.vanti.co.uk/smartcore/sc-golang/pkg/time"
)

const listBookingsOnPull = false

type BookingApi struct {
	traits.UnimplementedBookingApiServer
	bookingsById   map[string]*traits.Booking
	bookingsByIdMu sync.RWMutex
	// emits the "change" event with a single Arg of type *traits.PullBookingsResponse_Change
	bus *emitter.Emitter
	// used for generating ids
	Rng *rand.Rand
}

// compile time check that we implement the interface we need
var _ traits.BookingApiServer = &BookingApi{}

func NewBookingApi() *BookingApi {
	return &BookingApi{
		bookingsById: make(map[string]*traits.Booking),
		bus:          &emitter.Emitter{},
		Rng:          rand.New(rand.NewSource(rand.Int63())),
	}
}

func (b *BookingApi) ListBookings(_ context.Context, request *traits.ListBookingsRequest) (*traits.ListBookingsResponse, error) {
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
	log.Printf("ListBookings %v (%d returned)", request, len(response.Bookings))
	return response, nil
}

func (b *BookingApi) CheckInBooking(ctx context.Context, request *traits.CheckInBookingRequest) (*traits.CheckInBookingResponse, error) {
	if request.Time == nil {
		request.Time = serverTimestamp()
	}
	_, err := b.UpdateBooking(ctx, &traits.UpdateBookingRequest{
		Name: request.Name,
		Booking: &traits.Booking{
			Id: request.BookingId,
			CheckIn: &time.Period{
				StartTime: request.Time,
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"check_in.start_time"},
		},
	})
	if err != nil {
		return nil, err
	}
	return &traits.CheckInBookingResponse{}, nil
}

func (b *BookingApi) CheckOutBooking(ctx context.Context, request *traits.CheckOutBookingRequest) (*traits.CheckOutBookingResponse, error) {
	if request.Time == nil {
		request.Time = serverTimestamp()
	}
	_, err := b.UpdateBooking(ctx, &traits.UpdateBookingRequest{
		Name: request.Name,
		Booking: &traits.Booking{
			Id: request.BookingId,
			CheckIn: &time.Period{
				EndTime: request.Time,
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"check_in.end_time"},
		},
	})
	if err != nil {
		return nil, err
	}
	return &traits.CheckOutBookingResponse{}, nil
}

func (b *BookingApi) CreateBooking(_ context.Context, request *traits.CreateBookingRequest) (*traits.CreateBookingResponse, error) {
	if request.Booking == nil {
		return nil, status.Error(codes.InvalidArgument, "missing booking")
	}
	id := request.Booking.Id
	idGenerated := id == ""

	if id == "" {
		// try to generate a unique id
		b.bookingsByIdMu.RLock()
		for i := 0; i < 10; i++ {
			idCandidate := strconv.Itoa(b.Rng.Int())
			if _, exists := b.bookingsById[idCandidate]; idCandidate != "" && !exists {
				id = idCandidate
				break
			}
		}
		b.bookingsByIdMu.RUnlock()
	}

	if id == "" {
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
	log.Printf("CreateBooking %v %v", request.Name, request.Booking)
	b.bus.Emit("change", &traits.PullBookingsResponse_Change{
		Type:     types.ChangeType_ADD,
		NewValue: request.Booking,
	})

	return &traits.CreateBookingResponse{BookingId: id}, nil
}

func (b *BookingApi) UpdateBooking(_ context.Context, request *traits.UpdateBookingRequest) (*traits.UpdateBookingResponse, error) {
	if request.Booking == nil {
		return nil, status.Error(codes.InvalidArgument, "missing booking")
	}

	id := request.Booking.Id
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "missing booking.id")
	}

	var mask fieldMaskUtils.Mask
	if len(request.UpdateMask.GetPaths()) > 0 {
		if !request.UpdateMask.IsValid(request.Booking) {
			return nil, status.Error(codes.InvalidArgument, "invalid field_mask")
		}

		var err error
		mask, err = fieldMaskUtils.MaskFromPaths(request.UpdateMask.Paths, strcase.ToCamel)
		if err != nil {
			return nil, err
		}
	}

	change, err := b.applyChange(request.Name, id, func(newBooking *traits.Booking) error {
		if mask != nil {
			// apply only selected fields
			err := fieldMaskUtils.StructToStruct(mask, request.Booking, newBooking)
			if err != nil {
				return err
			}
		} else {
			// replace the booking data

			proto.Reset(newBooking)
			proto.Merge(newBooking, request.Booking)

		}
		log.Printf("UpdateBooking %v %v (mask:%v)", request.Name, newBooking, mask)
		return nil
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
	id := rand.Int()
	t0 := goTime.Now()
	sentItems := 0
	defer func() {
		log.Printf("[%x] PullBookings done in %v: sent %v", id, goTime.Now().Sub(t0), sentItems)
	}()
	log.Printf("[%x] PullBookings start %v", id, request)

	if listBookingsOnPull {
		// send all the bookings we already know about
		currentBookings, err := b.ListBookings(server.Context(), request)
		if err != nil {
			return err
		}
		initialResponse := &traits.PullBookingsResponse{}
		for _, booking := range currentBookings.Bookings {
			initialResponse.Changes = append(initialResponse.Changes, &traits.PullBookingsResponse_Change{
				Name:     request.Name,
				Type:     types.ChangeType_ADD,
				NewValue: booking,
			})
		}
		if len(initialResponse.Changes) > 0 {
			if err := server.Send(initialResponse); err != nil {
				return err
			}
			sentItems += len(initialResponse.Changes)
		}
	}

	// watch for changes to the bookings list and emit when one matches our query
	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
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
				sentItems++
			}
		}
	}
}

func (b *BookingApi) applyChange(name string, id string, fn func(newBooking *traits.Booking) error) (*traits.Booking, error) {
	oldValue, newValue, err := applyChange(
		&b.bookingsByIdMu,
		func() (proto.Message, bool) {
			val, exists := b.bookingsById[id]
			return val, exists
		},
		func(message proto.Message) error {
			return fn(message.(*traits.Booking))
		},
		func(message proto.Message) {
			b.bookingsById[id] = message.(*traits.Booking)
		})

	if err != nil {
		if s, ok := status.FromError(err); ok {
			return nil, status.Errorf(s.Code(), "%v %v", s.Message(), id)
		}
		return nil, err
	}
	b.bus.Emit("change", &traits.PullBookingsResponse_Change{
		Name:     name,
		Type:     types.ChangeType_UPDATE,
		OldValue: oldValue.(*traits.Booking),
		NewValue: newValue.(*traits.Booking),
	})
	return newValue.(*traits.Booking), nil
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
			Name:     change.Name,
			Type:     types.ChangeType_REMOVE,
			OldValue: change.OldValue,
		}
	} else if !wasIncluded && isIncluded {
		// added to query
		sentChange = &traits.PullBookingsResponse_Change{
			Name:     change.Name,
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
