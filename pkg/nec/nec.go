package nec

import (
	"time"

	"github.com/morphar/powernap"
	"github.com/stianeikeland/go-rpio/v4"
)

const (
	frequency                 = 38000
	dutyCycle                 = 0.25
	dutyPulseFrequency        = frequency / dutyCycle
	halfBitPulseRepetitions   = 32

	headerMarkDuration   = 9000 * time.Microsecond
	headerSpaceDuration  = 4500 * time.Microsecond
	bitMarkDuration      = 562 * time.Microsecond
	oneSpaceDuration     = 1687 * time.Microsecond
	zeroSpaceDuration    = 562 * time.Microsecond
	repeatSpaceDuration  = 2250 * time.Microsecond

	dutyCycleDuration = time.Second / frequency
	dutyPulseDuration  = time.Second / dutyPulseFrequency
	idleDuration       = 108000 * time.Microsecond
)

func addMark(plan *powernap.Plan, start time.Duration, pin rpio.Pin) {
	plan.Schedule(start, pin.Low)
	start += bitMarkDuration

	pulses := int(bitMarkDuration / dutyCycleDuration)
	for i := 0; i < pulses && i < halfBitPulseRepetitions; i++ {
		plan.Schedule(start, pin.High)
		plan.Schedule(start+dutyPulseDuration, pin.Low)
		start += dutyCycleDuration
	}
}

func addSpace(plan *powernap.Plan, start time.Duration, pin rpio.Pin, duration time.Duration) {
	plan.Schedule(start, pin.Low)
	plan.Schedule(start+duration, pin.Low)
}

func addHeader(plan *powernap.Plan, start time.Duration, pin rpio.Pin) {
	for i := 0; i < int(headerMarkDuration/dutyCycleDuration); i++ {
		plan.Schedule(start, pin.High)
		plan.Schedule(start+dutyPulseDuration, pin.Low)
		start += dutyCycleDuration
	}
	addSpace(plan, start, pin, headerSpaceDuration)
}

func Send(pin rpio.Pin, code uint32) {
	pin.Output()

	plan := powernap.NewPlan()
	n := time.Duration(0)

	addHeader(plan, n, pin)
	n += headerMarkDuration + headerSpaceDuration

	for i := 31; i >= 0; i-- {
		addMark(plan, n, pin)
		n += bitMarkDuration

		if code&(1<<i) != 0 {
			addSpace(plan, n, pin, oneSpaceDuration-bitMarkDuration)
			n += oneSpaceDuration
		} else {
			addSpace(plan, n, pin, zeroSpaceDuration-bitMarkDuration)
			n += zeroSpaceDuration
		}
	}

	addMark(plan, n, pin)
	n += bitMarkDuration

	plan.Schedule(n, pin.Low)
	plan.Schedule(n+idleDuration, func() {})

	plan.StartTightBlocking()
}

func SendRepeat(pin rpio.Pin) {
	pin.Output()

	plan := powernap.NewPlan()
	n := time.Duration(0)

	addHeader(plan, n, pin)
	n += headerMarkDuration + headerSpaceDuration

	addSpace(plan, n, pin, repeatSpaceDuration)
	n += repeatSpaceDuration

	addMark(plan, n, pin)
	n += bitMarkDuration

	plan.Schedule(n, pin.Low)
	plan.Schedule(n+idleDuration, func() {})

	plan.StartTightBlocking()
}