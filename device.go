package main

import (
	"fmt"

	"github.com/paulcull/go-webbrick"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/channels"
	"github.com/ninjasphere/go-ninja/model"

	"log"
	"regexp"
	"strings"
)

// OrviboDevice holds info about our socket.
type WebbrickDevice struct {
	driver       ninja.Driver
	info         *model.Device
	sendEvent    func(event string, payload interface{}) error
	onOffChannel *channels.OnOffChannel
	Device       webbrick.Device
}

func NewWebbrickDevice(driver ninja.Driver, id webbrick.Device) *WebbrickDevice {
	name := id.Name

	device := &WebbrickDevice{
		driver: driver,
		Device: id,
		info: &model.Device{
			NaturalID:     fmt.Sprintf("device%s", id.DevID),
			NaturalIDType: "socket",
			Name:          &name,
			Signatures: &map[string]string{
				"ninja:manufacturer": "Webbrick",
				"ninja:productName":  "WebbrickDevice",
				"ninja:productType":  "Socket",
				"ninja:thingType":    "socket",
			},
		},
	}

	device.onOffChannel = channels.NewOnOffChannel(device)
	return device
}

func (d *WebbrickDevice) GetDeviceInfo() *model.Device {
	return d.info
}

func (d *WebbrickDevice) GetDriver() ninja.Driver {
	return d.driver
}

func (d *WebbrickDevice) SetOnOff(state bool) error {
	fmt.Println("Setting state to", state)
	webbrick.SetState(d.Device.DevID, state)
	d.onOffChannel.SendState(state)
	return nil
}

func (d *WebbrickDevice) ToggleOnOff() error {
	fmt.Println("Toggling state")
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

	log.Printf("Setting device name to %s", *name)

	safe := reg.ReplaceAllString(strings.ToLower(*name), "")
	if len(safe) > 16 {
		safe = safe[0:16]
	}

	log.Printf("We can only set 5 lowercase alphanum. Name now: %s", safe)
	d.Device.Name = safe
	d.sendEvent("renamed", safe)

	return &safe, nil
}
