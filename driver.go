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

// OrviboDriver holds info about our driver, including our configuration
type WebbrickDriver struct {
	support.DriverSupport
	config *WebbrickDriverConfig
	conn   *ninja.Connection
}

// OrviboDriverConfig holds config info. I don't think it's extensively used in this driver?
type WebbrickDriverConfig struct {
	Initialised     bool
	NumberOfDevices int
	PollingMinutes  int
	PollingActive   bool
}

// No config provided? Set up some defaults
func defaultConfig() *WebbrickDriverConfig {
	return &WebbrickDriverConfig{
		Initialised: false,
	}
}

// NewDriver does what it says on the tin: makes a new driver for us to run.
func NewWebbrickDriver() (*WebbrickDriver, error) {

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
func (d *WebbrickDriver) Start(config *WebbrickDriverConfig) error {
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

func theloop(d *WebbrickDriver, config *WebbrickDriverConfig) error {
	go func() {
		started = true
		log.Infof("Calling theloop")

		ready, err := webbrick.Prepare() // You ready?
		if ready == true {               // Yep! Let's do this!
			// Because we'll never reach the end of the for loop (in theory),
			// we run SendEvent here.

			for { // Loop forever
				select { // This lets us do non-blocking channel reads. If we have a message, process it. If not, check for UDP data and loop
				case msg := <-webbrick.Events:
					log.Infof(" **** Event for ", msg.Name, "received...")
					switch msg.Name {
					// case "existingtempfound", "existingtriggerfound", "existingpirfound", "existingoutputfound", "existingbuttonfound", "existinglightchannelfound":
					// 	fmt.Println("  **** "+msg.Name+" Webbrick device updated! DEV ID is", msg.DeviceInfo.DevID)

					case "newwebbrickfound":
						log.Infof("  **** "+msg.Name+" Webbrick found! DEV ID is", msg.DeviceInfo.DevID)
						// Start the poller for the webbrick
						webbrick.PollWBStatus(msg.DeviceInfo.DevID)

					case "newtempfound", "newtriggerfound", "newpirfound", "newoutputfound", "newbuttonfound", "newlightchannelfound":
						log.Infof("  **** "+msg.Name+" Webbrick device found! DEV ID is", msg.DeviceInfo.DevID)
						str := spew.Sdump(msg.DeviceInfo)
						log.Infof(str)

						if msg.DeviceInfo.Queried == false {
							device[msg.DeviceInfo.ID] = NewWebbrickDevice(d, msg.DeviceInfo)

							// _ = d.Conn.ExportDevice(device[msg.DeviceInfo.ID])
							// _ = d.Conn.ExportChannel(device[msg.DeviceInfo.ID], device[msg.DeviceInfo.ID].onOffChannel, "on-off")
							// device[msg.DeviceInfo.ID].Device.Name = msg.Name
							// device[msg.DeviceInfo.ID].Device.State = msg.DeviceInfo.State
							// orvibo.Devices[msg.DeviceInfo.MACAddress].Queried = true
							// device[msg.DeviceInfo.ID].onOffChannel.SendState(msg.DeviceInfo.State)

						}

					}
				default:
					webbrick.CheckForMessages()
				}

				// for { // Loop forever
				// 	select { // This lets us do non-blocking channel reads. If we have a message, process it. If not, check for UDP data and loop
				// 	case msg := <-webbrick.Events:
				// 		switch msg.Name {
				// 		case "existingsocketfound":
				// 			fallthrough
				// 		case "socketfound":
				// 			fmt.Println("Socket found! MAC address is", msg.DeviceInfo.MACAddress)
				// 			orvibo.Subscribe() // Subscribe to any unsubscribed sockets
				// 			orvibo.Query()     // And query any unqueried sockets
				// 		case "subscribed":
				// 			if msg.DeviceInfo.Subscribed == false {

				// 				fmt.Println("Subscription successful!")
				// 				orvibo.Devices[msg.DeviceInfo.MACAddress].Subscribed = true
				// 				orvibo.Query()
				// 				fmt.Println("Query called")

				// 			}
				// 			orvibo.Query()
				// 		case "queried":

				// 			if msg.DeviceInfo.Queried == false {
				// 				device[msg.DeviceInfo.ID] = NewOrviboDevice(d, msg.DeviceInfo)

				// 				_ = d.Conn.ExportDevice(device[msg.DeviceInfo.ID])
				// 				_ = d.Conn.ExportChannel(device[msg.DeviceInfo.ID], device[msg.DeviceInfo.ID].onOffChannel, "on-off")
				// 				device[msg.DeviceInfo.ID].Device.Name = msg.Name
				// 				device[msg.DeviceInfo.ID].Device.State = msg.DeviceInfo.State
				// 				orvibo.Devices[msg.DeviceInfo.MACAddress].Queried = true
				// 				device[msg.DeviceInfo.ID].onOffChannel.SendState(msg.DeviceInfo.State)

				// 			}

				// 		case "statechanged":
				// 			fmt.Println("State changed to:", msg.DeviceInfo.State)
				// 			if msg.DeviceInfo.Queried == true {
				// 				device[msg.DeviceInfo.ID].Device.State = msg.DeviceInfo.State
				// 				device[msg.DeviceInfo.ID].onOffChannel.SendState(msg.DeviceInfo.State)
				// 			}
				// 		}
				// 	default:
				// 		web.CheckForMessages()
				// 	}

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
