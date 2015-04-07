package main

import (
	"fmt" // For outputting stuff to the screen

	"github.com/paulcull/go-webbrick" // The magic part that lets us control devices

	"github.com/davecgh/go-spew/spew" // For neatly outputting stuff

	"github.com/ninjasphere/go-ninja/api" // Ninja Sphere API
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/support"

	//"log"  // Similar thing, I suppose?
	"time" // Used as part of "setInterval" and for pausing code to allow for data to come back
)

// package.json is required, otherwise the app just exits and doesn't show any output
var info = ninja.LoadModuleInfo("./package.json")
var serial string

var log = logger.GetLogger(info.Name)

// Are we ready to rock?
var ready = false
var started = false // Stops us from running theloop twice
var device = make(map[int]*WebbrickDevice)

// WebbrickDriver holds info about our driver, including our configuration
type WebbrickDriver struct {
	support.DriverSupport
	config *webbrick.WebbrickDriverConfig
	conn   *ninja.Connection
}

// WebbrickDriverConfig holds config info. I don't think it's extensively used in this driver?
// type WebbrickDriverConfig struct {
// 	Name            string
// 	logger          logger.Logger
// 	Initialised     bool
// 	NumberOfDevices int
// 	PollingMinutes  int
// 	PollingActive   bool
// }

// No config provided? Set up some defaults
func defaultConfig() *webbrick.WebbrickDriverConfig {

	//log = logger.GetLogger(info.Name)

	return &webbrick.WebbrickDriverConfig{
		Name:        "PKHome",
		Initialised: false,
		//		NinjaLogControl: log,
		NumberOfDevices: 0,
		PollingMinutes:  5,
		PollingActive:   false,
	}
}

// NewDriver does what it says on the tin: makes a new driver for us to run.
func NewWebBrickDriver() (*WebbrickDriver, error) {

	// Make a new WebbrickDriver. Ampersand means to make a new copy, not reference the parent one (so A = new B instead of A = new B, C = A)
	driver := &WebbrickDriver{}

	// Initialize our driver. Throw back an error if necessary. Remember, := is basically a short way of saying "var blah string = 'abcd'"
	err := driver.Init(info)
	if err != nil {
		log.Fatalf("Failed to initialize Webbrick driver: %s", err)
		log.Fatalf("Failed to initialize driver: %s", err)
	}

	// Now we export the driver so the Sphere can find it (?)
	err = driver.Export(driver)
	if err != nil {
		log.Fatalf("Failed to export Webbrick driver: %s", err)
	}

	// NewDriver returns two things, WebbrickDriver, and an error if present
	return driver, nil
}

// Start is where the fun and magic happens! The driver is fired up and starts finding sockets
func (d *WebbrickDriver) Start(config *webbrick.WebbrickDriverConfig) error {
	log.Infof("Driver Starting with config %v", config)

	d.config = config
	if !d.config.Initialised {
		d.config = defaultConfig()
	}

	if started == false {
		theloop(d, config)
	}

	return d.SendEvent("config", config)
}

func theloop(d *WebbrickDriver, config *webbrick.WebbrickDriverConfig) error {
	go func() {
		started = true
		log.Infof("Calling theloop")

		ready, err := webbrick.Prepare(config) // You ready?
		if err != nil {
			log.Errorf("Error calling prepare", err)
		}
		if ready == true { // Yep! Let's do this!
			// Because we'll never reach the end of the for loop (in theory),
			// we run SendEvent here.

			for { // Loop forever
				select { // This lets us do non-blocking channel reads. If we have a message, process it. If not, check for UDP data and loop

				// Lets process all these events separately so that we don't miss anything, but can probcess them carefully
				case msg := <-webbrick.Events:
					log.Infof(" **** Event for " + msg.Name + " received...") // Tell the world what we've got
					switch msg.Name {
					// case "existingtempfound", "existingtriggerfound", "existingpirfound", "existingoutputfound", "existingbuttonfound", "existinglightchannelfound":
					// 	fmt.Println("  **** "+msg.Name+" Webbrick device updated! DEV ID is", msg.DeviceInfo.DevID)

					case "existingwebbrickupdated": // Have got a webbrick that we've already seen
						//fmt.Println("******* HERE *******")
						log.Infof("  **** "+msg.Name+" Webbrick seen again! DEV ID is", msg.DeviceInfo.DevID)
						//fallthrough

					case "newwebbrickfound": // Have got a new webbrick, lets go see what it can do for us
						log.Infof("  **** "+msg.Name+" Webbrick found! DEV ID is", msg.DeviceInfo.DevID)
						// Start the poller for the webbrick
						webbrick.PollWBStatus(msg.DeviceInfo.DevID)
						//fallthrough

					case "newtempfound", "newtriggerfound", "newpirfound", "newoutputfound", "newbuttonfound", "newlightchannelfound":
						// These are all the devices that we care about - so lets look after them now
						log.Infof("  **** "+msg.Name+" Webbrick device found! DEV ID is", msg.DeviceInfo.DevID)
						str := spew.Sdump(msg.DeviceInfo)
						log.Debugf(str)

						if device[msg.DeviceInfo.ID] == nil { // Do I already know about this on the sphere ???
							log.Infof("  **** NEW DEVICE NEEDED -  Webbrick device found! DEV ID is", msg.DeviceInfo.DevID)

							// Lets create a new sphere device driver for this webbrick device
							device[msg.DeviceInfo.ID] = NewWebbrickDevice(d, msg.DeviceInfo)

							// Now we've got the device back from the creator let's tell the world
							_ = d.Conn.ExportDevice(device[msg.DeviceInfo.ID])

							// Set the rest of the device up
							device[msg.DeviceInfo.ID].Device.Name = msg.Name
							device[msg.DeviceInfo.ID].Device.State = msg.DeviceInfo.State
							webbrick.Devices[msg.DeviceInfo.DevID].Queried = true

							// Create any special channels for specfic devices and knock it out

							// State output is really just an on-off that we want to support
							if msg.Name == "newoutputfound" {
								_ = d.Conn.ExportChannel(device[msg.DeviceInfo.ID], device[msg.DeviceInfo.ID].onOffChannel, "on-off")
								device[msg.DeviceInfo.ID].onOffChannel.SendState(msg.DeviceInfo.State)
							}

							// pir output is really just motion that we want to support
							if msg.Name == "newpirfound" {
								_ = d.Conn.ExportChannel(device[msg.DeviceInfo.ID], device[msg.DeviceInfo.ID].motionChannel, "motion")
							}

							// Light output is really just an on-off and brightnessthat we want to support. Don't need colour, but
							// don't know if NS really care about the non-colour lights
							if msg.Name == "newlightchannelfound" {
								_ = d.Conn.ExportChannel(device[msg.DeviceInfo.ID], device[msg.DeviceInfo.ID].onOffChannel, "on-off")
								_ = d.Conn.ExportChannel(device[msg.DeviceInfo.ID], device[msg.DeviceInfo.ID].brightnessChannel, "brightness")
								device[msg.DeviceInfo.ID].onOffChannel.SendState(msg.DeviceInfo.State)
								device[msg.DeviceInfo.ID].brightnessChannel.Set(msg.DeviceInfo.Level)
							}

						} else {
							log.Infof("  **** EXISTING DEVICE FOUND -  Webbrick device found! DEV ID is", msg.DeviceInfo.DevID)

							// we have this device already on the sphere, lets just check our labels and levels

							device[msg.DeviceInfo.ID].Device.Name = msg.DeviceInfo.Name

							// State output is really just an on-off that we want to support
							if msg.Name == "newoutputfound" {
								device[msg.DeviceInfo.ID].onOffChannel.SendState(msg.DeviceInfo.State)
							}

							// Light output is really just an on-off and brightnessthat we want to support. Don't need colour, but
							// don't know if NS really care about the non-colour lights
							if msg.Name == "newlightchannelfound" {
								device[msg.DeviceInfo.ID].onOffChannel.SendState(msg.DeviceInfo.State)
								device[msg.DeviceInfo.ID].brightnessChannel.Set(msg.DeviceInfo.Level)
							}

						}

					}
				default:
					webbrick.CheckForMessages()
				}

			}

		} else {
			fmt.Println("Error:", err)

		}

	}()
	return nil
}

func (d *WebbrickDriver) Stop() error {
	return fmt.Errorf("This driver does not support being stopped. YOU HAVE NO POWER HERE.")

}

func setInterval(what func(), delay time.Duration) chan bool {
	stop := make(chan bool)

	go func() {
		for {
			what()
			select {
			case <-time.After(delay):
			case <-stop:
				return
			}
		}
	}()

	return stop
}
