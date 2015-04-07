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

const (
	UNKNOWN = -1 + iota // UNKNOWN is obviously a device that isn't implemented or is unknown. iota means add 1 to the next const, so SOCKET = 0, ALLONE = 1 etc.

	LIGHT     // LIGHT - is possibly a dimmer
	PIR       // PIR - Trigger
	BUTTON    // Pushbutton - Trigger
	TEMP      // Temp sensor
	STATE     // State
	HEARTBEAT // Heartbeat

)

// webBrickDevice holds info about our socket.
type WebbrickDevice struct {
	driver            ninja.Driver
	info              *model.Device
	sendEvent         func(event string, payload interface{}) error
	onOffChannel      *channels.OnOffChannel
	brightnessChannel *channels.BrightnessChannel
	motionChannel     *channels.MotionChannel
	tempChannel       *channels.TemperatureChannel
	Device            webbrick.Device
	log               *logger.Logger
}

func NewWebbrickDevice(driver ninja.Driver, id webbrick.Device) *WebbrickDevice {
	//name := id.Name

	log.Infof("In creating NewWebbrickDevie", id.Name)

	var devProductType, devThingType string

	switch id.Type {
	case LIGHT:
		devProductType = "light"
		devThingType = "Light"
	case PIR:
		devProductType = "motion"
		devThingType = "Motion"
	case TEMP:
		devProductType = "temperature"
		devThingType = "Temperature"
	default:
		devProductType = "light"
		devThingType = "Light"
	}

	// LIGHT     // LIGHT - is possibly a dimmer
	// PIR       // PIR - Trigger
	// BUTTON    // Pushbutton - Trigger
	// TEMP      // Temp sensor
	// STATE     // State
	// HEARTBEAT // Heartbeat

	// switch Devices[devID].Type {

	// // Its a light
	// case LIGHT:
	// 	// update the record for new levels
	// 	Devices[devID].Level = int(level)

	// select { // This lets us do non-blocking channel reads. If we have a message, process it. If not, check for UDP data and loop
	// case id.Type:
	// }
	// devType := id.Type

	log.Infof("Creating a new Device, type: %s. Name now: %s", devThingType, id.Name)

	device := &WebbrickDevice{
		driver: driver,
		Device: id,
		info: &model.Device{
			NaturalID:     fmt.Sprintf("device%s", id.DevID),
			NaturalIDType: devProductType,
			Name:          &id.Name,
			Signatures: &map[string]string{
				"ninja:manufacturer": "Webbrick",
				"ninja:productName":  "Webbrick" + devThingType + "Device",
				"ninja:productType":  devThingType,
				"ninja:thingType":    devProductType,
			},
		},
		log: logger.GetLogger("Light Device - " + id.Name),
	}

	if devProductType == "light" {
		device.onOffChannel = channels.NewOnOffChannel(device)
		device.brightnessChannel = channels.NewBrightnessChannel(device)
	}
	if devProductType == "state" {
		device.onOffChannel = channels.NewOnOffChannel(device)
	}
	if devProductType == "temp" {
		device.onOffChannel = channels.NewTemperatureChannel(device)
	}
	if devProductType == "pir" {
		device.motionChannel = channels.NewMotionChannel()
	}
	return device
}

func (d *WebbrickDevice) GetDeviceInfo() *model.Device {
	return d.info
}

func (d *WebbrickDevice) GetDriver() ninja.Driver {
	return d.driver
}

func (d *WebbrickDevice) SetBrightness(level float64) error {
	log.Infof("Setting Brightness level to", level)
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
