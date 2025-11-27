// Package events provides types that define the actions that clients can take
// on tasks or on other events.
package events

import (
	"errors"

	"github.com/idloquy/trckr/pkg/tasks"
)

var (
	ErrEmptyTaskName  = errors.New("empty name")
	ErrSwitchSameTask = errors.New("switching to the same task")
)

// The Event interface defines the methods that allow for describing and validating
// an event and represents the possible event types.
type Event interface {
	eventMarker()
	Kind() string
	Name() string
	Validate() error
}

type baseEvent struct{}

func (ev baseEvent) eventMarker() {}

// The TaskEvent interface represents the possible task event types.
//
// Task events are events that operate on task states, such as starting or stopping
// a task.
type TaskEvent interface {
	Event
	taskEventMarker()
}

type baseTaskEvent struct {
	baseEvent
}

func (ev baseTaskEvent) taskEventMarker() {}

// Kind returns the kind of the event.
func (ev baseTaskEvent) Kind() string {
	return "task"
}

// A StartEvent represents the starting of a task. The reason leading to the
// stopping of the previous task may be present in the StopReason field.
type StartEvent struct {
	baseTaskEvent
	Task       string `json:"task"`
	StopReason string `json:"stop_reason"`
}

// NewStartEvent returns a StartEvent for the specified task. If non-empty, the
// specified stop reason is also used.
func NewStartEvent(task, stopReason string) (StartEvent, error) {
	ev := StartEvent{
		baseTaskEvent: baseTaskEvent{
			baseEvent: baseEvent{},
		},
		Task:       task,
		StopReason: stopReason,
	}
	err := ev.Validate()
	return ev, err
}

// Name returns the name of the event.
func (ev StartEvent) Name() string {
	return "start"
}

// Validate verifies that the event is valid.
func (ev StartEvent) Validate() error {
	if ev.Task == "" {
		return ErrEmptyTaskName
	}

	return nil
}

// A StopEvent represents the stopping of a task.
type StopEvent struct {
	baseTaskEvent
	Task string `json:"task"`
}

// NewStopEvent returns a StopEvent for the specified task.
func NewStopEvent(task string) (StopEvent, error) {
	ev := StopEvent{
		baseTaskEvent: baseTaskEvent{
			baseEvent: baseEvent{},
		},
		Task: task,
	}
	err := ev.Validate()
	return ev, err
}

// Name returns the name of the event.
func (ev StopEvent) Name() string {
	return "stop"
}

// Validate verifies that the event is valid.
func (ev StopEvent) Validate() error {
	if ev.Task == "" {
		return ErrEmptyTaskName
	}

	return nil
}

// A SwitchEvent represents the switching from one task to another.
type SwitchEvent struct {
	baseTaskEvent
	OldTask string `json:"old_task"`
	NewTask string `json:"new_task"`
}

// NewSwitchEvent returns a SwitchEvent from oldTask to newTask.
func NewSwitchEvent(oldTask, newTask string) (SwitchEvent, error) {
	ev := SwitchEvent{
		baseTaskEvent: baseTaskEvent{
			baseEvent: baseEvent{},
		},
		OldTask: oldTask,
		NewTask: newTask,
	}
	err := ev.Validate()
	return ev, err
}

// Name returns the name of the event.
func (ev SwitchEvent) Name() string {
	return "switch"
}

// Validate verifies that the event is valid.
func (ev SwitchEvent) Validate() error {
	if ev.OldTask == "" || ev.NewTask == "" {
		return ErrEmptyTaskName
	}

	if ev.OldTask == ev.NewTask {
		return ErrSwitchSameTask
	}

	return nil
}

// The OperationEvent interface represents the possible operation event types.
//
// OperationEvents are events that represent operations on task events or operations
// that don't involve task states, such as creating a task or undoing
// an event.
type OperationEvent interface {
	Event
	operationEventMarker()
}

type baseOperationEvent struct {
	baseEvent
}

func (ev baseOperationEvent) operationEventMarker() {}

// Kind returns the kind of the event.
func (ev baseOperationEvent) Kind() string {
	return "operation"
}

// A CreateTaskEvent represents the creation of a task.
type CreateTaskEvent struct {
	baseOperationEvent
	Task tasks.Task `json:"task"`
}

// NewCreateTaskEvent returns a CreateTaskEvent for the specified task.
func NewCreateTaskEvent(task tasks.Task) (CreateTaskEvent, error) {
	ev := CreateTaskEvent{
		Task: task,
	}
	err := ev.Validate()
	return ev, err
}

// Name returns the name of the event.
func (CreateTaskEvent) Name() string {
	return "create task"
}

// Validate verifies that the event is valid.
func (ev CreateTaskEvent) Validate() error {
	if err := ev.Task.Validate(); err != nil {
		return err
	}
	return nil
}

// A RenameTaskEvent represents the renaming of a task.
type RenameTaskEvent struct {
	baseOperationEvent
	Task    string `json:"task"`
	NewName string `json:"new_name"`
}

// NewRenameTaskEvent returns a RenameTaskEvent renaming the specified task to
// newName.
func NewRenameTaskEvent(task string, newName string) (RenameTaskEvent, error) {
	ev := RenameTaskEvent{
		Task:    task,
		NewName: newName,
	}
	err := ev.Validate()
	return ev, err
}

// Name returns the name of the event.
func (ev RenameTaskEvent) Name() string {
	return "rename"
}

// Validate verifies that the event is valid.
func (ev RenameTaskEvent) Validate() error {
	if ev.Task == "" || ev.NewName == "" {
		return ErrEmptyTaskName
	}

	return nil
}

// A DeleteTaskEvent represents the deletion of a task.
type DeleteTaskEvent struct {
	baseOperationEvent
	Task string `json:"task"`
}

// NewDeleteTaskEvent returns a DeleteTaskEvent for the specified task.
func NewDeleteTaskEvent(task string) (DeleteTaskEvent, error) {
	ev := DeleteTaskEvent{
		Task: task,
	}
	err := ev.Validate()
	return ev, err
}

// Name returns the name of the event.
func (ev DeleteTaskEvent) Name() string {
	return "delete task"
}

// Validate verifies that the event is valid.
func (ev DeleteTaskEvent) Validate() error {
	if ev.Task == "" {
		return ErrEmptyTaskName
	}

	return nil
}

// An UndoEvent represents the undoing of a task event.
type UndoEvent struct {
	baseOperationEvent
}

// NewUndoEvent returns an UndoEvent.
func NewUndoEvent() UndoEvent {
	return UndoEvent{
		baseOperationEvent{
			baseEvent: baseEvent{},
		},
	}
}

// Name returns the name of the event.
func (ev UndoEvent) Name() string {
	return "undo"
}

// Validate verifies that the event is valid.
func (ev UndoEvent) Validate() error {
	return nil
}
