package led

import (
	"image/color"
	"log"
	"math"
	"sync"
	"time"

	"github.com/morphar/remotepi/pkg/nec"
	"github.com/stianeikeland/go-rpio/v4"
)

const (
	commandSpacing = 100 * time.Millisecond
)

type State struct {
	On         bool
	Brightness int
	Color      color.RGBA
	Effect     string
}

type Command func()

type Controller struct {
	pin         rpio.Pin
	state       State
	mu          sync.Mutex
	commandChan chan Command
	stopChan    chan struct{}
}

type PresetColor struct {
	Name  string
	Color color.RGBA
	Code  uint32
}

var presetColors = []PresetColor{
	{Name: "red", Color: color.RGBA{R: 255, G: 0, B: 0, A: 255}, Code: nec.Red},
	{Name: "green", Color: color.RGBA{R: 0, G: 255, B: 0, A: 255}, Code: nec.Green},
	{Name: "blue", Color: color.RGBA{R: 0, G: 0, B: 255, A: 255}, Code: nec.Blue},
	{Name: "white", Color: color.RGBA{R: 255, G: 255, B: 255, A: 255}, Code: nec.White},
	{Name: "color1", Color: color.RGBA{R: 255, G: 51, B: 51, A: 255}, Code: nec.Color1},
	{Name: "color2", Color: color.RGBA{R: 51, G: 255, B: 51, A: 255}, Code: nec.Color2},
	{Name: "color3", Color: color.RGBA{R: 51, G: 51, B: 255, A: 255}, Code: nec.Color3},
	{Name: "color4", Color: color.RGBA{R: 200, G: 200, B: 255, A: 255}, Code: nec.Color4},
	{Name: "color5", Color: color.RGBA{R: 255, G: 0, B: 128, A: 255}, Code: nec.Color5},
	{Name: "color6", Color: color.RGBA{R: 0, G: 255, B: 128, A: 255}, Code: nec.Color6},
	{Name: "color7", Color: color.RGBA{R: 128, G: 0, B: 255, A: 255}, Code: nec.Color7},
	{Name: "color8", Color: color.RGBA{R: 255, G: 128, B: 0, A: 255}, Code: nec.Color8},
	{Name: "color9", Color: color.RGBA{R: 255, G: 128, B: 128, A: 255}, Code: nec.Color9},
	{Name: "color0", Color: color.RGBA{R: 128, G: 255, B: 128, A: 255}, Code: nec.Color0},
}

var Effects = map[string]uint32{
	"none":   0,
	"flash":  nec.EffectFlash,
	"fade":   nec.EffectFade,
	"smooth": nec.EffectSmooth,
	"strobe": nec.EffectStrobe,
}

func NewController(pin rpio.Pin) *Controller {
	c := &Controller{
		pin: pin,
		state: State{
			On:         false,
			Brightness: 255,
			Color:      color.RGBA{R: 255, G: 255, B: 255, A: 255},
			Effect:     "none",
		},
		commandChan: make(chan Command, 100),
		stopChan:    make(chan struct{}),
	}
	go c.processCommands()
	return c
}

func (c *Controller) Stop() {
	close(c.stopChan)
}

func (c *Controller) processCommands() {
	for {
		select {
		case <-c.stopChan:
			return
		case cmd := <-c.commandChan:
			cmd()
			time.Sleep(commandSpacing)
		}
	}
}

func (c *Controller) queueCommand(cmd Command) {
	select {
	case c.commandChan <- cmd:
	default:
		log.Println("LED command queue full, dropping command")
	}
}

func (c *Controller) GetState() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

func (c *Controller) SetOn(on bool) {
	c.mu.Lock()
	if on == c.state.On {
		c.mu.Unlock()
		return
	}
	c.state.On = on
	c.mu.Unlock()

	c.queueCommand(func() {
		nec.Send(c.pin, nec.Power)
		if on {
			preset := c.findNearestPreset()
			nec.Send(c.pin, preset.Code)
		}
	})
}

func (c *Controller) SetBrightness(brightness int, transition time.Duration) {
	if brightness < 0 {
		brightness = 0
	}
	if brightness > 255 {
		brightness = 255
	}

	c.mu.Lock()
	currentBrightness := c.state.Brightness
	c.state.Brightness = brightness
	isOn := c.state.On
	c.mu.Unlock()

	if !isOn {
		return
	}

	steps := brightness - currentBrightness
	if steps == 0 {
		return
	}

	if transition > 0 && abs(steps) > 20 {
		c.transitionBrightness(currentBrightness, brightness, transition)
	} else {
		c.setBrightnessImmediate(steps)
	}
}

func (c *Controller) setBrightnessImmediate(steps int) {
	command := nec.BrightnessUp
	if steps < 0 {
		command = nec.BrightnessDown
		steps = -steps
	}

	for i := 0; i < steps/20; i++ {
		c.queueCommand(func() {
			nec.Send(c.pin, command)
		})
	}
}

func (c *Controller) transitionBrightness(from, to int, duration time.Duration) {
	steps := abs(to - from)
	if steps == 0 {
		return
	}

	numCommands := steps / 20
	if numCommands == 0 {
		numCommands = 1
	}

	interval := duration / time.Duration(numCommands)

	go func() {
		direction := 1
		command := nec.BrightnessUp
		if to < from {
			direction = -1
			command = nec.BrightnessDown
		}

		for i := 0; i < numCommands; i++ {
			c.queueCommand(func() {
				nec.Send(c.pin, command)
			})
			time.Sleep(interval)
		}
	}()
}

func (c *Controller) SetColor(r, g, b uint8) {
	c.mu.Lock()
	c.state.Color = color.RGBA{R: r, G: g, B: b, A: 255}
	isOn := c.state.On
	c.mu.Unlock()

	if !isOn {
		return
	}

	preset := c.findNearestPreset()
	c.queueCommand(func() {
		nec.Send(c.pin, preset.Code)
	})
}

func (c *Controller) SetEffect(effect string) {
	code, ok := Effects[effect]
	if !ok {
		return
	}

	c.mu.Lock()
	c.state.Effect = effect
	isOn := c.state.On
	c.mu.Unlock()

	if !isOn || code == 0 {
		return
	}

	c.queueCommand(func() {
		nec.Send(c.pin, code)
	})
}

func (c *Controller) SetWhite() {
	c.mu.Lock()
	c.state.Color = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	c.state.Effect = "none"
	isOn := c.state.On
	c.mu.Unlock()

	if !isOn {
		return
	}

	c.queueCommand(func() {
		nec.Send(c.pin, nec.White)
	})
}

func (c *Controller) BrightnessStep(step int) {
	c.mu.Lock()
	isOn := c.state.On
	currentBrightness := c.state.Brightness
	c.mu.Unlock()

	if !isOn {
		return
	}

	newBrightness := currentBrightness + step
	if newBrightness < 0 {
		newBrightness = 0
	}
	if newBrightness > 255 {
		newBrightness = 255
	}

	c.mu.Lock()
	c.state.Brightness = newBrightness
	c.mu.Unlock()

	numPresses := abs(step) / 20
	if numPresses < 1 {
		numPresses = 1
	}
	if numPresses > 5 {
		numPresses = 5
	}

	command := nec.BrightnessUp
	if step < 0 {
		command = nec.BrightnessDown
	}

	for i := 0; i < numPresses; i++ {
		c.queueCommand(func() {
			nec.Send(c.pin, command)
		})
	}
}

func (c *Controller) findNearestPreset() PresetColor {
	c.mu.Lock()
	defer c.mu.Unlock()

	minDist := math.MaxFloat64
	var nearest PresetColor = presetColors[0]

	for _, preset := range presetColors {
		dist := colorDistance(c.state.Color, preset.Color)
		if dist < minDist {
			minDist = dist
			nearest = preset
		}
	}

	return nearest
}

func colorDistance(c1, c2 color.RGBA) float64 {
	dr := float64(c1.R) - float64(c2.R)
	dg := float64(c1.G) - float64(c2.G)
	db := float64(c1.B) - float64(c2.B)
	return math.Sqrt(dr*dr + dg*dg + db*db)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}