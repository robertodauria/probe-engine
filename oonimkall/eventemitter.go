package oonimkall

import (
	"time"

	"github.com/ooni/probe-engine/atomicx"
)

// eventEmitter emits event on a channel
type eventEmitter struct {
	disabled map[string]bool
	out      chan<- *eventRecord
	timeouts *atomicx.Int64
}

// newEventEmitter creates a new Emitter
func newEventEmitter(disabledEvents []string, out chan<- *eventRecord) *eventEmitter {
	ee := &eventEmitter{out: out}
	ee.disabled = make(map[string]bool)
	for _, eventname := range disabledEvents {
		ee.disabled[eventname] = true
	}
	ee.timeouts = atomicx.NewInt64()
	return ee
}

// EmitFailureStartup emits the failureStartup event
func (ee *eventEmitter) EmitFailureStartup(failure string) {
	ee.EmitFailureGeneric(failureStartup, failure)
}

// EmitFailure emits a failure event
func (ee *eventEmitter) EmitFailureGeneric(name, failure string) {
	ee.Emit(name, eventFailureGeneric{Failure: failure})
}

// EmitStatusProgress emits the status.Progress event
func (ee *eventEmitter) EmitStatusProgress(percentage float64, message string) {
	ee.Emit(statusProgress, eventStatusProgress{Message: message, Percentage: percentage})
}

// Emit emits the specified event
func (ee *eventEmitter) Emit(key string, value interface{}) {
	if ee.disabled[key] == true {
		return
	}
	select {
	case <-time.After(250 * time.Millisecond):
		ee.timeouts.Add(1)
	case ee.out <- &eventRecord{Key: key, Value: value}:
	}
}
