package main

import (
	"fmt"

	"github.com/paulcull/go-webbrick"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/channels"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"

	//	"log"
	"regexp"
	"strings"
)

// webBrickDevice holds info about our socket.
type WebbrickDevice struct {
	driver            ninja.Driver
	info              *model.Device
	sendEvent         func(event string, payload interface{}) error
	onOffChannel      *channels.OnOffChannel
	brightnessChannel *channels.BrightnessChannel
	motionChannel     *channels.MotionChannel
	temperature       *channels.TemperatureChannel
	Device            webbrick.Device
	log               *logger.Logger
}

func NewWebbrickDevice(driver ninja.Driver, id webbrick.Device) *WebbrickDevice {
	name := id.Name

	device := &WebbrickDevice{
		driver: driver,
		Device: id,
		info: &model.Device{
			NaturalID:     fmt.Sprintf("device%s", id.DevID),
			NaturalIDType: "light",
			Name:          &name,
			Signatures: &map[string]string{
				"ninja:manufacturer": "Webbrick",
				"ninja:productName":  "WebbrickLightDevice",
				"ninja:productType":  "Light",
				"ninja:thingType":    "light",
			},
		},
		log: logger.GetLogger("Light Device - " + id.Name),
	}
	// var log = logger.GetLogger(info.Name)
	// device.log = log

	device.onOffChannel = channels.NewOnOffChannel(device)
	device.brightnessChannel = channels.NewBrightnessChannel(device)
	device.temperature = channels.NewTemperatureChannel(device)
	device.motionChannel = channels.NewMotionChannel()
	return device
}

func (d *WebbrickDevice) GetDeviceInfo() *model.Device {
	return d.info
}

func (d *WebbrickDevice) GetDriver() ninja.Driver {
	return d.driver
}

func (d *WebbrickDevice) SetBrightness(level float64) error {
	log.Infof("Setting state to", level)
	webbrick.SetLevel(d.Device.DevID, level)
	d.brightnessChannel.Set(level)
	return nil
}

func (d *WebbrickDevice) SetOnOff(state bool) error {
	log.Infof("Setting state to", state)
	webbrick.SetState(d.Device.DevID, state)
	d.onOffChannel.SendState(state)
	return nil
}

func (d *WebbrickDevice) ToggleOnOff() error {
	log.Infof("Toggling state")
	webbrick.ToggleState(d.Device.DevID)
	d.onOffChannel.SendState(d.Device.State)
	return nil
}

func (d *WebbrickDevice) SetEventHandler(sendEvent func(event string, payload interface{}) error) {
	d.sendEvent = sendEvent
}

var reg, _ = regexp.Compile("[^a-z0-9]")

// Exported by service/device schema
func (d *WebbrickDevice) SetName(name *string) (*string, error) {

	log.Infof("Setting device name to %s", *name)

	safe := reg.ReplaceAllString(strings.ToLower(*name), "")
	if len(safe) > 16 {
		safe = safe[0:16]
	}

	log.Warningf("We can only set 5 lowercase alphanum. Name now: %s", safe)
	d.Device.Name = safe
	d.sendEvent("renamed", safe)

	return &safe, nil
}
