package trait

import (
	"strings"
)

type Name string

// Local returns the last part of a fully qualified trait name.
func (n Name) Local() string {
	traitName := string(n)
	lastDot := strings.LastIndex(traitName, ".")
	if lastDot == -1 {
		return traitName
	}
	return traitName[lastDot+1:]
}

func (n Name) String() string {
	return string(n)
}

const (
	AirQualitySensor Name = "smartcore.traits.AirQualitySensor"
	AirTemperature   Name = "smartcore.traits.AirTemperature"
	Booking          Name = "smartcore.traits.Booking"
	BrightnessSensor Name = "smartcore.traits.BrightnessSensor"
	Channel          Name = "smartcore.traits.Channel"
	Count            Name = "smartcore.traits.Count"
	Electric         Name = "smartcore.traits.Electric"
	Emergency        Name = "smartcore.traits.Emergency"
	EnergyStorage    Name = "smartcore.traits.EnergyStorage"
	ExtendRetract    Name = "smartcore.traits.ExtendRetract"
	FanSpeed         Name = "smartcore.traits.FanSpeed"
	InputSelect      Name = "smartcore.traits.InputSelect"
	Light            Name = "smartcore.traits.Light"
	Metadata         Name = "smartcore.traits.Metadata"
	Microphone       Name = "smartcore.traits.Microphone"
	MotionSensor     Name = "smartcore.traits.MotionSensor"
	OccupancySensor  Name = "smartcore.traits.OccupancySensor"
	OnOff            Name = "smartcore.traits.OnOff"
	OpenClose        Name = "smartcore.traits.OpenClose"
	Parent           Name = "smartcore.traits.Parent"
	PowerSupply      Name = "smartcore.traits.PowerSupply"
	Ptz              Name = "smartcore.traits.Ptz"
	Speaker          Name = "smartcore.traits.Speaker"
)
