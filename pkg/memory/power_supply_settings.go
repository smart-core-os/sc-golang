package memory

//go:generate protoc -I. --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. power_supply_settings.proto

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"google.golang.org/protobuf/proto"
)

func (s *PowerSupplyApi) readSettings() *MemoryPowerSupplySettings {
	return s.settings.Get().(*MemoryPowerSupplySettings)
}

func (s *PowerSupplyApi) GetSettings(_ context.Context, req *GetMemoryPowerSupplySettingsReq) (*MemoryPowerSupplySettings, error) {
	return s.settings.Get(WithGetMask(req.Fields)).(*MemoryPowerSupplySettings), nil
}

func (s *PowerSupplyApi) UpdateSettings(_ context.Context, req *UpdateMemoryPowerSupplySettingsReq) (*MemoryPowerSupplySettings, error) {
	var oldSettings *MemoryPowerSupplySettings
	newVal, err := s.settings.Set(
		req.Settings,
		InterceptAfter(func(old, new proto.Message) {
			oldSettings = old.(*MemoryPowerSupplySettings)
		}),
	)
	if err != nil {
		return nil, err
	}
	newSettings := newVal.(*MemoryPowerSupplySettings)
	if err := s.updateCapacityForSettingChange(oldSettings, newSettings); err != nil {
		return nil, err
	}

	return newSettings, nil
}

func (s *PowerSupplyApi) PullSettings(req *PullMemoryPowerSupplySettingsReq, server MemoryPowerSupplySettingsApi_PullSettingsServer) error {
	events, done := s.settings.OnUpdate(server.Context())
	defer done()

	var lastSentMsg *MemoryPowerSupplySettings
	filter := masks.NewResponseFilter(masks.WithFieldMask(req.Fields))
	for event := range events {
		settings := filter.FilterClone(event.Value).(*MemoryPowerSupplySettings)
		if lastSentMsg != nil && proto.Equal(lastSentMsg, settings) {
			// nothing has changed, nothing to send
			continue
		}
		res := &PullMemoryPowerSupplySettingsRes{
			Changes: []*PullMemoryPowerSupplySettingsRes_Change{
				{
					Name:       req.Name,
					Settings:   settings,
					ChangeTime: event.ChangeTime,
				},
			},
		}
		if err := server.Send(res); err != nil {
			return err
		}
	}
	return nil
}

func (s *PowerSupplyApi) updateCapacityForSettingChange(oldSettings, newSettings *MemoryPowerSupplySettings) error {
	var updateCapacity bool
	var capacityUpdateFields []string
	if oldSettings.Rating != newSettings.Rating {
		updateCapacity = true
		capacityUpdateFields = append(capacityUpdateFields, "rating")
	}
	if oldSettings.Voltage != newSettings.Voltage {
		updateCapacity = true
		capacityUpdateFields = append(capacityUpdateFields, "voltage")
	}
	if oldSettings.Load != newSettings.Load {
		updateCapacity = true
		capacityUpdateFields = append(capacityUpdateFields, "load")
	}
	if oldSettings.Reserved != newSettings.Reserved {
		updateCapacity = true
	}

	if updateCapacity {
		capacity := traits.PowerCapacity{
			Rating:  newSettings.Rating,
			Voltage: newSettings.Voltage,
			Load:    &newSettings.Load,
		}
		_, err := s.powerCapacity.Set(&capacity,
			WithUpdatePaths(capacityUpdateFields...),
			WithMoreWritablePaths(capacityUpdateFields...),
			InterceptAfter(func(old, new proto.Message) {
				newCapacity := new.(*traits.PowerCapacity)
				adjustPowerCapacityForLoad(newCapacity, newSettings.Reserved)
			}))
		if err != nil {
			return err
		}
	}
	return nil
}
