package memory

import (
	"fmt"
	"testing"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
	"golang.org/x/net/context"
	"google.golang.org/protobuf/types/known/durationpb"
)

var ctx = context.Background()

func powerSupplyClient() traits.PowerSupplyApiClient {
	var device = NewPowerSupplyApi()
	device.SetLoad(20)
	var client = wrap.PowerSupplyApiServer(device)
	return client
}

func powerCapacity(client traits.PowerSupplyApiClient) *traits.PowerCapacity {
	capacity, _ := client.GetPowerCapacity(ctx, &traits.GetPowerCapacityRequest{})
	return capacity
}

func ExamplePowerSupplyApi_GetPowerCapacity() {
	client := powerSupplyClient()
	capacity, err := client.GetPowerCapacity(ctx, &traits.GetPowerCapacityRequest{
		Name: "WallSocket",
	})
	if err != nil {
		fmt.Printf("failed to get capacity: %v", err)
		return
	}

	fmt.Printf("current free capacity: %vA", capacity.Free)
	// Output: current free capacity: 40A
}

func ExamplePowerSupplyApi_CreateDrawNotification_noMinDraw() {
	// this client only has 40A free
	client := powerSupplyClient()
	deviceName := "WallSocket"
	actual, _ := client.CreateDrawNotification(ctx, &traits.CreateDrawNotificationRequest{
		Name: deviceName,
		DrawNotification: &traits.DrawNotification{
			MaxDraw: 100, // amps
			// no MinDraw, we accept 100 or nothing
		},
	})

	fmt.Printf("reserved capacity: %vA", actual.MaxDraw)
	// Output: reserved capacity: 0A
}

func ExamplePowerSupplyApi_CreateDrawNotification_minDraw() {
	// this client has 40A free
	client := powerSupplyClient()
	deviceName := "WallSocket"
	actual, _ := client.CreateDrawNotification(ctx, &traits.CreateDrawNotificationRequest{
		Name: deviceName,
		DrawNotification: &traits.DrawNotification{
			MaxDraw: 100, // amps
			MinDraw: 10,  // amps
		},
	})

	fmt.Printf("reserved capacity: %vA", actual.MaxDraw)
	// Output: reserved capacity: 40A
}

func TestPowerSupplyApi_CreateDrawNotification(t *testing.T) {
	t.Run("expires", func(t *testing.T) {
		client := powerSupplyClient()
		ramp := 100 * time.Millisecond
		actual, _ := client.CreateDrawNotification(ctx, &traits.CreateDrawNotificationRequest{
			DrawNotification: &traits.DrawNotification{
				MaxDraw:      10,
				RampDuration: durationpb.New(ramp),
			},
		})
		if actual.RampDuration.AsDuration() != ramp {
			t.Errorf("ramp duration: want %v got %v", ramp, actual.RampDuration.AsDuration())
		}

		capacity := powerCapacity(client)
		if capacity.Notified != 10 {
			t.Fatalf("notified: want %v got %v", 10, capacity.Notified)
		}

		// wait for the notification to expire (plus some overhead)
		<-time.After(ramp + 100*time.Millisecond)

		capacity = powerCapacity(client)
		if capacity.Notified != 0 {
			t.Fatalf("notified: want %v got %v", 0, capacity.Notified)
		}
	})
}
