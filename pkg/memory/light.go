package memory

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/tanema/gween"
	"github.com/tanema/gween/ease"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// LightApi implements the LightApiServer interface for a single device by storing state in memory.
type LightApi struct {
	traits.UnimplementedLightApiServer
	brightness     *Resource
	brightnessTick time.Duration // duration between updates when tweening brightness

	// todo: support presets
}

func NewLightApi() *LightApi {
	return &LightApi{
		brightnessTick: time.Second / 15,
		brightness: NewResource(
			WithInitialValue(InitialBrightness()),
			WithWritablePaths(&traits.Brightness{},
				"level_percent",
				"brightness_tween.total_duration",
			),
		),
	}
}

func InitialBrightness() *traits.Brightness {
	return &traits.Brightness{}
}

func (s *LightApi) GetBrightness(_ context.Context, _ *traits.GetBrightnessRequest) (*traits.Brightness, error) {
	return s.brightness.Get().(*traits.Brightness), nil
}

func (s *LightApi) UpdateBrightness(ctx context.Context, request *traits.UpdateBrightnessRequest) (*traits.Brightness, error) {
	if err := ValidateTweenOnUpdate("brightness", request.GetBrightness().GetBrightnessTween()); err != nil {
		return nil, err
	}

	duration := request.Brightness.GetBrightnessTween().GetTotalDuration().AsDuration()
	if duration > 0 {
		startTime := time.Now()
		lastObj, err := s.brightness.Set(request.Brightness,
			WithUpdatePaths("level_percent", "brightness_tween", "target_level_percent"),
			WithMoreWritablePaths("brightness_tween", "target_level_percent"),
			InterceptBefore(func(old, new proto.Message) {
				current := old.(*traits.Brightness)
				next := new.(*traits.Brightness)
				if request.Delta {
					next.LevelPercent += current.LevelPercent
				}
				// move properties into their tween equivalents
				next.TargetLevelPercent = next.LevelPercent
				next.LevelPercent = current.LevelPercent
				next.BrightnessTween.Progress = 0
			}),
		)
		if err != nil {
			return nil, err
		}

		startVal := lastObj.(*traits.Brightness)
		tween := gween.New(startVal.LevelPercent, startVal.TargetLevelPercent, float32(duration.Milliseconds()), ease.Linear)

		go func() {
			ticker := time.NewTicker(s.brightnessTick)
			defer ticker.Stop()
			for {
				now := <-ticker.C
				playTime := now.Sub(startTime)
				newValue, finished := tween.Set(float32(playTime.Milliseconds()))
				if finished {
					// the tween has completed, reset the tween data
					_, err := s.brightness.Set(&traits.Brightness{LevelPercent: newValue},
						WithUpdatePaths("level_percent"),
						WithResetPaths("target_level_percent", "brightness_tween"),
						WithExpectedValue(lastObj),
					)
					if err != nil && err != ExpectedValuePreconditionFailed {
						panic(err) // programmer error
					}
					return
				}

				// calculate using time, not value, which leave room for easing (and is mentioned in the tween spec)
				progress := 100 * float32(playTime.Milliseconds()) / float32(duration.Milliseconds())
				lastObj, err = s.brightness.Set(&traits.Brightness{LevelPercent: newValue, BrightnessTween: &types.Tween{Progress: progress}},
					WithUpdatePaths("level_percent", "brightness_tween.progress"),
					WithMoreWritablePaths("brightness_tween.progress"),
					WithExpectedValue(lastObj),
				)
				switch {
				case err == ExpectedValuePreconditionFailed:
					// somebody else changed the value, tweening is done
					return
				case err != nil:
					panic(err) // programmer error
				}
			}
		}()

		return startVal, nil
	}

	res, err := s.brightness.Set(
		request.Brightness,
		// if there's a tween in progress, clear the tween props
		WithResetPaths("target_level_percent", "brightness_tween"),
		InterceptBefore(func(old, change proto.Message) {
			if request.Delta {
				val := old.(*traits.Brightness)
				delta := change.(*traits.Brightness)
				delta.LevelPercent += val.LevelPercent
			}
		}))
	if err != nil {
		return nil, err
	}
	return res.(*traits.Brightness), nil
}

func (s *LightApi) PullBrightness(request *traits.PullBrightnessRequest, server traits.LightApi_PullBrightnessServer) error {
	changes, done := s.brightness.OnUpdate(server.Context())
	defer done()

	for {
		select {
		case <-server.Context().Done():
			return status.FromContextError(server.Context().Err()).Err()
		case event := <-changes:
			brightness := event.Value.(*traits.Brightness)
			// don't emit progress if the caller doesn't want it
			if request.ExcludeRamping {
				progress := brightness.GetBrightnessTween().GetProgress()
				if progress != 0 && progress != 100 {
					continue
				}
			}

			change := &traits.PullBrightnessResponse_Change{
				Name:       request.Name,
				Brightness: brightness,
				ChangeTime: event.ChangeTime,
			}
			err := server.Send(&traits.PullBrightnessResponse{
				Changes: []*traits.PullBrightnessResponse_Change{change},
			})
			if err != nil {
				return err
			}
		}
	}
}
