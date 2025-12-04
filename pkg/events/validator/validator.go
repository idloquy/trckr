// Package validator provides functionality for validating event sequences.
package validator

import (
	"errors"
	"fmt"

	"github.com/idloquy/trckr/pkg/events"
)

var (
	ErrUnexpectedEvent      = errors.New("unexpected event")
	ErrTaskNotRunning       = errors.New("task not running")
	ErrUnexpectedStopReason = errors.New("unexpected stop reason")
	ErrUnexpectedStopTags   = errors.New("unexpected stop tags")
)

// The Validator provides functionality for validating event sequences.
type Validator struct {
	partialSequence bool
}

// New returns a standard Validator with no special behavior.
func New() *Validator {
	return &Validator{
		partialSequence: false,
	}
}

// ForPartialSequence returns a Validator that can be used for validating partial
// event sequences.
func ForPartialSequence() *Validator {
	return &Validator{
		partialSequence: true,
	}
}

// DefaultValidator is a validator with no special behavior.
var DefaultValidator = &Validator{
	partialSequence: false,
}

// ValidateSequence validates that ev can be added to the sequence in evs without
// breaking any rules.
//
// A start event can be added if it is the first event, or if the previous event
// is a stop event.
//
// A stop event can be added if it is the first event and this is a partial sequence
// validator, or if the previous event is a start or a switch event and the task
// it refers to is running.
//
// A switch event can be added if it is the first event and this is a partial sequence
// validator, or if the previous event is a start event or a switch event. In both
// cases, the new task must be different from the old task.
func (v *Validator) ValidateSequence(evs []events.TaskEvent, ev events.TaskEvent) error {
	if err := ev.Validate(); err != nil {
		return err
	}

	isFirst := len(evs) == 0
	if isFirst {
		switch ev := ev.(type) {
		case events.StartEvent:
			if ev.StopReason != "" && !v.partialSequence {
				return ErrUnexpectedStopReason
			}

			if len(ev.StopTags) > 0 && !v.partialSequence {
				return ErrUnexpectedStopTags
			}
		case events.StopEvent:
			if !v.partialSequence {
				return ErrUnexpectedEvent
			}
		case events.SwitchEvent:
			if !v.partialSequence {
				return ErrUnexpectedEvent
			}
		default:
			panic(fmt.Sprintf("handling for %s events not implemented", ev.Name()))
		}
	} else {
		prevEv := evs[len(evs)-1]
		switch ev := ev.(type) {
		case events.StartEvent:
			switch prevEv.(type) {
			case events.StopEvent:
			case events.StartEvent:
				return ErrUnexpectedEvent
			case events.SwitchEvent:
				return ErrUnexpectedEvent
			default:
				panic(fmt.Sprintf("handling for %s events not implemented", ev.Name()))
			}
		case events.StopEvent:
			var prevTask string

			switch prevEv := prevEv.(type) {
			case events.StartEvent:
				prevTask = prevEv.Task
			case events.SwitchEvent:
				prevTask = prevEv.NewTask
			case events.StopEvent:
				return ErrUnexpectedEvent
			default:
				panic(fmt.Sprintf("handling for %s events not implemented", prevEv.Name()))
			}

			if ev.Task != prevTask {
				return ErrTaskNotRunning
			}
		case events.SwitchEvent:
			var prevTask string

			switch prevEv := prevEv.(type) {
			case events.StartEvent:
				prevTask = prevEv.Task
			case events.SwitchEvent:
				prevTask = prevEv.NewTask
			case events.StopEvent:
				return ErrUnexpectedEvent
			default:
				panic(fmt.Sprintf("handling for %s events not implemented", prevEv.Name()))
			}

			if prevTask != ev.OldTask {
				return ErrTaskNotRunning
			}
		default:
			panic(fmt.Sprintf("handling for %s events not implemented", ev.Name()))
		}
	}
	return nil
}
