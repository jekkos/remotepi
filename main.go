package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"
	"time"

	"github.com/morphar/remotepi/pkg/config"
	"github.com/morphar/remotepi/pkg/led"
	"github.com/morphar/remotepi/pkg/mqtt"
	"github.com/morphar/remotepi/pkg/rc5"
	"github.com/stianeikeland/go-rpio/v4"
)

func init() {
	log.SetFlags(log.Ltime | log.Lshortfile)
}

var reRunning = regexp.MustCompile("state: RUNNING")

func main() {
	cfg := config.Load()

	err := rpio.Open()
	exitOnErr(err)
	defer rpio.Close()

	marantzPin := rpio.Pin(cfg.MarantzPin)
	defer marantzPin.Low()

	ledPin := rpio.Pin(cfg.LEDPin)
	defer ledPin.Low()

	turnOn := rc5.CommandX(16, 12, 1, 0)
	turnOff := rc5.CommandX(16, 12, 2, 0)

	offDelay := cfg.OffDelay

	statusFiles, err := filepath.Glob("/proc/asound/" + cfg.AudioCard + "/pcm*/sub*/status")
	exitOnErr(err)

	var lastOn time.Time
	var stateOn bool

	ledController := led.NewController(ledPin)

	mqttClient := mqtt.NewClient(cfg, ledController)
	err = mqttClient.Connect()
	exitOnErr(err)
	defer mqttClient.Disconnect()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			log.Println("Shutting down...")
			return

		case <-ticker.C:
			curStateOn := false
			for _, statusFile := range statusFiles {
				src, err := os.ReadFile(statusFile)
				exitOnErr(err)
				if reRunning.Match(src) {
					curStateOn = true
					break
				}
			}

			if curStateOn {
				lastOn = time.Now()
			}

			if curStateOn && !stateOn {
				stateOn = true
				rc5.Send(marantzPin, turnOn, true)
				time.Sleep(time.Second)
				marantzPin.Input()
				marantzPin.PullOff()
				continue
			}

			if !curStateOn && stateOn && time.Since(lastOn) > offDelay {
				stateOn = false
				rc5.Send(marantzPin, turnOff, true)
				marantzPin.Input()
				marantzPin.PullOff()
			}
		}
	}
}

func exitOnErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}