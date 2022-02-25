package powersupply

//go:generate protomod protoc -- -I../../../ --go_out=paths=source_relative:../../../ --go-grpc_out=paths=source_relative:../../../ pkg/trait/powersupply/memory_settings.proto

import (
	"context"

	"github.com/smart-core-os/sc-golang/pkg/resource"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"google.golang.org/protobuf/proto"
)

func (s *MemoryDevice) readSettings() *MemorySettings {
	return s.settings.Get().(*MemorySettings)
}

func (s *MemoryDevice) GetSettings(_ context.Context, req *GetMemorySettingsReq) (*MemorySettings, error) {
	return s.settings.Get(resource.WithGetMask(req.Fields)).(*MemorySettings), nil
}

func (s *MemoryDevice) UpdateSettings(_ context.Context, req *UpdateMemorySettingsReq) (*MemorySettings, error) {
	var oldSettings *MemorySettings
	newVal, err := s.settings.Set(
		req.Settings,
		resource.WithUpdateMask(req.UpdateMask),
		resource.InterceptAfter(func(old, new proto.Message) {
			oldSettings = old.(*MemorySettings)
		}),
	)
	if err != nil {
		return nil, err
	}
	newSettings := newVal.(*MemorySettings)
	if err := s.updateCapacityForSettingChange(oldSettings, newSettings); err != nil {
		return nil, err
	}

	return newSettings, nil
}

func (s *MemoryDevice) PullSettings(req *PullMemorySettingsReq, server MemorySettingsApi_PullSettingsServer) error {
	events := s.settings.Pull(server.Context())

	var lastSentMsg *MemorySettings
	filter := masks.NewResponseFilter(masks.WithFieldMask(req.Fields))
	for event := range events {
		settings := filter.FilterClone(event.Value).(*MemorySettings)
		if lastSentMsg != nil && proto.Equal(lastSentMsg, settings) {
			// nothing has changed, nothing to send
			continue
		}
		res := &PullMemorySettingsRes{
			Changes: []*PullMemorySettingsRes_Change{
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

func (s *MemoryDevice) updateCapacityForSettingChange(oldSettings, newSettings *MemorySettings) error {
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
			resource.WithUpdatePaths(capacityUpdateFields...),
			resource.WithMoreWritablePaths(capacityUpdateFields...),
			resource.InterceptAfter(func(old, new proto.Message) {
				newCapacity := new.(*traits.PowerCapacity)
				adjustPowerCapacityForLoad(newCapacity, newSettings.Reserved)
			}))
		if err != nil {
			return err
		}
	}
	return nil
}
