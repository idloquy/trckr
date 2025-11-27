// Package tasks defines types for managing tasks.
package tasks

import (
	"errors"
)

var (
	ErrEmptyTaskName = errors.New("empty name")
	ErrInvalidState  = errors.New("invalid state")
)

// State represents task states.
type State string

// Possible task states.
const (
	StoppedState       = "stopped"
	RunningState State = "running"
)

// ParseState parses a string representation of a State.
func ParseState(state string) (State, error) {
	switch state {
	case string(StoppedState):
		return StoppedState, nil
	case string(RunningState):
		return RunningState, nil
	default:
		return "", ErrInvalidState
	}
}

// Validate verifies that a given State is valid.
func (state State) Validate() bool {
	return state == StoppedState || state == RunningState
}

// A task represents a unit of work.
type Task struct {
	Name  string `json:"name"`
	State State  `json:"state"`
}

// New returns a task with the given name and state.
func New(name string, state State) (Task, error) {
	var task Task
	if name == "" {
		return task, ErrEmptyTaskName
	}

	if !state.Validate() {
		return task, ErrInvalidState
	}

	return Task{
		Name:  name,
		State: state,
	}, nil
}

// Validate verifies that a task is valid.
//
// A valid task has a non-empty name and a valid state.
func (t Task) Validate() error {
	if t.Name == "" {
		return ErrEmptyTaskName
	}

	if !t.State.Validate() {
		return ErrInvalidState
	}

	return nil
}
