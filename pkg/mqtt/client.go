package mqtt

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/morphar/remotepi/pkg/config"
	"github.com/morphar/remotepi/pkg/led"
)

const (
	discoveryTopic = "homeassistant/light/remotepi_ledstrip/config"
	commandTopic   = "homeassistant/light/remotepi_ledstrip/set"
	stateTopic     = "homeassistant/light/remotepi_ledstrip/state"
)

type Command struct {
	State         string          `json:"state"`
	Brightness    int             `json:"brightness,omitempty"`
	BrightnessStep int             `json:"brightness_step,omitempty"`
	Color         *Color          `json:"color,omitempty"`
	WhiteValue    int             `json:"white_value,omitempty"`
	Effect        string          `json:"effect,omitempty"`
	Transition    json.RawMessage `json:"transition,omitempty"`
}

type Color struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

type State struct {
	State      string  `json:"state"`
	Brightness int     `json:"brightness"`
	Color      *Color  `json:"color"`
	Effect     string  `json:"effect"`
	Transition int     `json:"transition,omitempty"`
}

type Client struct {
	client     mqtt.Client
	controller *led.Controller
	mu         sync.Mutex
}

func NewClient(cfg *config.Config, controller *led.Controller) *Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.MQTTBroker)
	opts.SetClientID(cfg.MQTTClientID)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectTimeout(30 * time.Second)

	if cfg.MQTTUsername != "" {
		opts.SetUsername(cfg.MQTTUsername)
		opts.SetPassword(cfg.MQTTPassword)
	}

	c := &Client{
		controller: controller,
	}

	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Printf("MQTT connected to %s", cfg.MQTTBroker)
		c.publishDiscovery()
		c.subscribe()
		c.publishState()
	})

	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Printf("MQTT connection lost: %v", err)
	})

	c.client = mqtt.NewClient(opts)
	return c
}

func (c *Client) Connect() error {
	token := c.client.Connect()
	token.Wait()
	return token.Error()
}

func (c *Client) Disconnect() {
	c.controller.Stop()
	c.client.Disconnect(250)
}

func (c *Client) publishDiscovery() {
	discovery := map[string]interface{}{
		"name":             "LED Strip",
		"unique_id":        "remotepi_ledstrip",
		"command_topic":    commandTopic,
		"state_topic":      stateTopic,
		"schema":           "json",
		"rgb":              true,
		"brightness":        true,
		"effect":           true,
		"effect_list":      []string{"none", "flash", "fade", "smooth", "strobe"},
		"brightness_scale": 255,
		"payload_on":       "ON",
		"payload_off":      "OFF",
	}

	payload, err := json.Marshal(discovery)
	if err != nil {
		log.Printf("Failed to marshal discovery: %v", err)
		return
	}

	token := c.client.Publish(discoveryTopic, 1, true, payload)
	token.Wait()
	if token.Error() != nil {
		log.Printf("Failed to publish discovery: %v", token.Error())
	} else {
		log.Printf("Published MQTT discovery")
	}
}

func (c *Client) subscribe() {
	token := c.client.Subscribe(commandTopic, 1, c.handleCommand)
	token.Wait()
	if token.Error() != nil {
		log.Printf("Failed to subscribe to %s: %v", commandTopic, token.Error())
	} else {
		log.Printf("Subscribed to %s", commandTopic)
	}
}

func (c *Client) handleCommand(client mqtt.Client, msg mqtt.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var cmd Command
	if err := json.Unmarshal(msg.Payload(), &cmd); err != nil {
		log.Printf("Failed to parse command: %v", err)
		return
	}

	transition := parseTransition(cmd.Transition)

	log.Printf("Received command: state=%s, brightness=%d, brightness_step=%d, effect=%s, transition=%v", 
		cmd.State, cmd.Brightness, cmd.BrightnessStep, cmd.Effect, transition)

	if cmd.State == "OFF" {
		c.controller.SetOn(false)
	} else if cmd.State == "ON" {
		c.controller.SetOn(true)

		if cmd.BrightnessStep != 0 {
			c.controller.BrightnessStep(cmd.BrightnessStep)
		} else if cmd.Brightness > 0 {
			c.controller.SetBrightness(cmd.Brightness, transition)
		}

		if cmd.Color != nil {
			c.controller.SetColor(cmd.Color.R, cmd.Color.G, cmd.Color.B)
		}

		if cmd.WhiteValue > 0 {
			c.controller.SetWhite()
		}

		if cmd.Effect != "" {
			c.controller.SetEffect(cmd.Effect)
		}
	}

	c.publishState()
}

func parseTransition(raw json.RawMessage) time.Duration {
	if len(raw) == 0 {
		return 0
	}

	var transition int
	if err := json.Unmarshal(raw, &transition); err != nil {
		var transitionFloat float64
		if err := json.Unmarshal(raw, &transitionFloat); err != nil {
			return 0
		}
		transition = int(transitionFloat)
	}

	if transition <= 0 {
		return 0
	}

	return time.Duration(transition) * time.Second
}

func (c *Client) publishState() {
	state := c.controller.GetState()

	effect := state.Effect
	if effect == "" {
		effect = "none"
	}

	var colorVal *Color
	if state.On {
		colorVal = &Color{
			R: state.Color.R,
			G: state.Color.G,
			B: state.Color.B,
		}
	}

	payload := State{
		State:      "OFF",
		Brightness: state.Brightness,
		Color:      colorVal,
		Effect:     effect,
	}

	if state.On {
		payload.State = "ON"
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal state: %v", err)
		return
	}

	token := c.client.Publish(stateTopic, 1, true, data)
	token.Wait()
	if token.Error() != nil {
		log.Printf("Failed to publish state: %v", token.Error())
	}
}

func (c *Client) Announce() {
	c.publishDiscovery()
	c.publishState()
}