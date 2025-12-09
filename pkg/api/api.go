// Package api provides types for facilitating the interaction between the client
// and the server.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/idloquy/trckr/pkg/events"
	"github.com/idloquy/trckr/pkg/tasks"
)

var (
	ErrUnknownResponseStatus    = errors.New("unknown response status")
	ErrUnexpectedResponseStatus = errors.New("unexpected response status")

	ErrUnknownEventType                     = errors.New("unknown event type")
	ErrUnknownEventTypeName                 = errors.New("unknown event type name")
	ErrMissingEventTypeDiscriminator        = errors.New("missing event type discriminator")
	ErrMismatchingEventTypeForEventTypeName = errors.New("event type doesn't match event type name")

	ErrUnknownTaskEventType                         = errors.New("unknown task event type")
	ErrUnknownTaskEventTypeName                     = errors.New("unknown task event type name")
	ErrMissingTaskEventTypeDiscriminator            = errors.New("missing task event type discriminator")
	ErrMismatchingTaskEventTypeForTaskEventTypeName = errors.New("task event type doesn't match task event type name")

	ErrUnknownOperationEventType                              = errors.New("unknown operation event type")
	ErrUnknownOperationEventTypeName                          = errors.New("unknown operation event type name")
	ErrMissingOperationEventTypeDiscriminator                 = errors.New("missing operation event type discriminator")
	ErrMismatchingOperationEventTypeForOperationEventTypeName = errors.New("operation event type doesn't match operation event type name")

	ErrUnexpectedEventType = errors.New("unexpected event type")
)

// ResponseStatus represents response statuses.
type ResponseStatus string

const (
	ResponseStatusSuccess ResponseStatus = "success"
	ResponseStatusError   ResponseStatus = "error"
)

const (
	ServerName = "trckr"
)

// The CreateTaskMsg is to be used by clients when creating tasks.
type CreateTaskMsg struct {
	Name string `json:"name"` // specifies the name of the task to create
}

// The UpdateTaskMsg is to be used by clients when updating tasks.
type UpdateTaskMsg struct {
	Name *string `json:"name"` // specifies the new name of the task
}

// The Response interface represents the possible response types.
type Response interface {
	responseMarker()
}

type ErrorResponseContainerBase struct {
	Error string `json:"error"`
}

// ErrorResponseContainer represents an error response.
type ErrorResponseContainer ErrorResponseContainerBase

func (ErrorResponseContainer) responseMarker() {}

type transientErrorResponseContainer struct {
	ResponseContainerMeta
	ErrorResponseContainerBase
}

func (c ErrorResponseContainer) MarshalJSON() ([]byte, error) {
	transient := transientErrorResponseContainer{
		ResponseContainerMeta:      ResponseContainerMeta{Status: ResponseStatusError},
		ErrorResponseContainerBase: ErrorResponseContainerBase{Error: c.Error},
	}
	return json.Marshal(transient)
}

func (c *ErrorResponseContainer) UnmarshalJSON(data []byte) error {
	var transient transientErrorResponseContainer
	if err := json.Unmarshal(data, &transient); err != nil {
		return err
	}

	if transient.Status != ResponseStatusError {
		return fmt.Errorf("%w: should be %v but is %v", ErrUnexpectedResponseStatus, ResponseStatusError, transient.Status)
	}

	container := ErrorResponseContainer(transient.ErrorResponseContainerBase)
	*c = container

	return nil
}

// The SuccessResponseData interface represents the possible types that can be
// used within success responses.
type SuccessResponseData interface {
	successResponseDataMarker()
}

type baseSuccessResponseData struct{}

func (baseSuccessResponseData) successResponseDataMarker() {}

// InfoResponse represents a response with server information.
type InfoResponse struct {
	baseSuccessResponseData
	Name    string `json:"name"`
	Version int    `json:"version"`
}

// TaskResponse represents a response with task information.
type TaskResponse struct {
	baseSuccessResponseData
	Task tasks.Task `json:"task"`
}

// TaskListResponse represents a response containing a task list.
type TaskListResponse struct {
	baseSuccessResponseData
	Tasks []tasks.Task `json:"tasks"`
}

// TaskEventResponse represents a response with a task event.
type TaskEventResponse struct {
	baseSuccessResponseData
	TaskEvent EventContainer[TaskEvent] `json:"task_event"`
}

// TaskEventListResponse represents a response containing a task event list.
type TaskEventListResponse struct {
	baseSuccessResponseData
	TaskEvents []EventContainer[TaskEvent] `json:"task_events"`
}

// SuccessResponseContainer represents a success response containing data of the
// specified type.
type SuccessResponseContainer[T SuccessResponseData] struct {
	Data T
}

func (SuccessResponseContainer[T]) responseMarker() {}

func (c SuccessResponseContainer[T]) MarshalJSON() ([]byte, error) {
	var translated any

	// This switch is required because we need the actual response data to
	// be at the same level as the metadata, and type parameters can't be embedded,
	// which would be required in order for inlining to be doable otherwise.
	var data any = c.Data
	switch data := data.(type) {
	case InfoResponse:
		translated = struct {
			ResponseContainerMeta
			InfoResponse
		}{
			ResponseContainerMeta: ResponseContainerMeta{Status: ResponseStatusSuccess},
			InfoResponse:          data,
		}
	case TaskResponse:
		translated = struct {
			ResponseContainerMeta
			TaskResponse
		}{
			ResponseContainerMeta: ResponseContainerMeta{Status: ResponseStatusSuccess},
			TaskResponse:          data,
		}
	case TaskListResponse:
		translated = struct {
			ResponseContainerMeta
			TaskListResponse
		}{
			ResponseContainerMeta: ResponseContainerMeta{Status: ResponseStatusSuccess},
			TaskListResponse:      data,
		}
	case TaskEventListResponse:
		translated = struct {
			ResponseContainerMeta
			TaskEventListResponse
		}{
			ResponseContainerMeta: ResponseContainerMeta{Status: ResponseStatusSuccess},
			TaskEventListResponse: data,
		}
	case TaskEventResponse:
		translated = struct {
			ResponseContainerMeta
			TaskEventResponse
		}{
			ResponseContainerMeta: ResponseContainerMeta{Status: ResponseStatusSuccess},
			TaskEventResponse:     data,
		}
	default:
		panic(fmt.Sprintf("handling for type %s not implemented", reflect.TypeOf(data).Name()))
	}

	return json.Marshal(translated)
}

func (c *SuccessResponseContainer[T]) UnmarshalJSON(data []byte) error {
	var transient struct {
		ResponseContainerMeta
	}
	if err := json.Unmarshal(data, &transient); err != nil {
		return err
	}

	if transient.Status != ResponseStatusSuccess {
		return fmt.Errorf("%w: should be %v but is %v", ErrUnexpectedResponseStatus, ResponseStatusSuccess, transient.Status)
	}

	return json.Unmarshal(data, &c.Data)
}

// ErrorResponseJSON is an intermediary representation of an error response that
// can be further unmarshaled into an ErrorResponseContainer.
//
// Attempting to marshal this will result in a panic.
type ErrorResponseJSON struct {
	Data json.RawMessage
}

func (ErrorResponseJSON) responseMarker() {}

func (ErrorResponseJSON) MarshalJSON() ([]byte, error) {
	panic("ErrorResponseJSON can't be marshaled")
}

// SuccessResponseJSON is an intermediary representation of a success response
// that can be further unmarshaled into a SuccessResponseContainer.
//
// Attempting to marshal this will result in a panic.
type SuccessResponseJSON struct {
	Data json.RawMessage
}

func (SuccessResponseJSON) responseMarker() {}

func (SuccessResponseJSON) MarshalJSON() ([]byte, error) {
	panic("SuccessResponseJSON can't be marshaled")
}

// ResponseContainerMeta holds response-agnostic metadata.
type ResponseContainerMeta struct {
	Status ResponseStatus `json:"status"`
}

// ResponseContainer is the base type containing more specific response types.
//
// Clients are expected to use this type when unmarshaling server responses, and
// then further unmarshal the underlying, specific response type.
type ResponseContainer struct {
	Response
}

func (c ResponseContainer) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Response)
}

func (c *ResponseContainer) UnmarshalJSON(data []byte) error {
	var meta ResponseContainerMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return err
	}

	switch meta.Status {
	case ResponseStatusSuccess:
		c.Response = SuccessResponseJSON{data}
	case ResponseStatusError:
		c.Response = ErrorResponseJSON{data}
	default:
		return ErrUnknownResponseStatus
	}

	return nil
}

// NewSuccessResponse returns a ResponseContainer containing a SuccessResponseContainer
// with the specified data.
func NewSuccessResponse[T SuccessResponseData](data T) ResponseContainer {
	return ResponseContainer{
		Response: SuccessResponseContainer[T]{
			Data: data,
		},
	}
}

// NewErrorResponse returns a ResponseContainer containing an ErrorResponseContainer
// with the specified error message.
func NewErrorResponse(msg string) ResponseContainer {
	return ResponseContainer{
		Response: ErrorResponseContainer{
			Error: msg,
		},
	}
}

// EventType represents the possible event types.
type EventType uint

const (
	EventTypeTask EventType = iota
	EventTypeOperation
)

const (
	EventTypeNameTask      = "task"
	EventTypeNameOperation = "operation"
)

type transientEventContainerMeta struct {
	EventType     *EventType `json:"event_type"`
	EventTypeName *string    `json:"event_type_name"`
}

func transientEventContainerMetaFromEvent(ev events.Event) transientEventContainerMeta {
	var evType EventType
	var evTypeName string

	switch ev := ev.(type) {
	case events.TaskEvent:
		evType = EventTypeTask
		evTypeName = EventTypeNameTask
	case events.OperationEvent:
		evType = EventTypeOperation
		evTypeName = EventTypeNameOperation
	default:
		panic(fmt.Sprintf("handling for %s events not implemented", ev.Kind()))
	}

	return transientEventContainerMeta{
		EventType:     &evType,
		EventTypeName: &evTypeName,
	}
}

func (meta transientEventContainerMeta) validate() error {
	if meta.EventType == nil && meta.EventTypeName == nil {
		return ErrMissingEventTypeDiscriminator
	}

	if meta.EventType != nil {
		if *meta.EventType != EventTypeTask &&
			*meta.EventType != EventTypeOperation {
			return ErrUnknownEventType
		}
	}

	if meta.EventTypeName != nil {
		if *meta.EventTypeName != EventTypeNameTask &&
			*meta.EventTypeName != EventTypeNameOperation {
			return ErrUnknownEventTypeName
		}
	}

	if meta.EventType != nil && meta.EventTypeName != nil {
		if (*meta.EventType == EventTypeTask && *meta.EventTypeName != EventTypeNameTask) ||
			(*meta.EventType == EventTypeOperation && *meta.EventTypeName != EventTypeNameOperation) {
			return ErrMismatchingEventTypeForEventTypeName
		}
	}

	return nil
}

// getTypeDiscriminator returns the type discriminator from the EventTypeName field,
// if non-null, or from the EventType field otherwise. It is meant to be called
// after unmarshaling.
//
// NOTE: meta must have been verified to be valid when calling this.
func (meta transientEventContainerMeta) getTypeDiscriminator() string {
	if meta.EventTypeName != nil {
		return *meta.EventTypeName
	}

	switch *meta.EventType {
	case EventTypeTask:
		return EventTypeNameTask
	case EventTypeOperation:
		return EventTypeNameOperation
	default:
		panic("expected valid meta")
	}
}

// The Event interface represents the possible events.
type Event interface {
	eventMarker()
	events.Event
}

// The RequestEvent interface represents the possible events that can be used by
// clients in requests.
type RequestEvent interface {
	requestEventMarker()
	events.Event
}

// EventContainerMeta contains event type-agnostic metadata.
type EventContainerMeta struct {
	At time.Time `json:"at"`
}

type EventContainerBase[T Event] struct {
	EventContainerMeta
	Event T `json:"event"`
}

// EventContainer represents an event of the specified type.
//
// This type implements custom (un)marshaling logic for differentiation between
// task event types. Using this type for (un)marshaling events is the recommended
// approach.
type EventContainer[T Event] EventContainerBase[T]

func (container EventContainer[T]) MarshalJSON() ([]byte, error) {
	transientContainer := struct {
		transientEventContainerMeta
		EventContainerBase[T]
	}{
		transientEventContainerMeta: transientEventContainerMetaFromEvent(container.Event),
		EventContainerBase:          EventContainerBase[T](container),
	}
	return json.Marshal(transientContainer)
}

func (container *EventContainer[T]) UnmarshalJSON(data []byte) error {
	var transientContainer struct {
		transientEventContainerMeta
		EventContainerMeta
		Event json.RawMessage `json:"event"`
	}

	if err := json.Unmarshal(data, &transientContainer); err != nil {
		return err
	}

	if err := transientContainer.transientEventContainerMeta.validate(); err != nil {
		return err
	}

	var err error
	var ev Event = container.Event

	transientEvType := transientContainer.getTypeDiscriminator()
	switch ev := ev.(type) {
	case TaskEvent:
		if transientEvType != EventTypeNameTask {
			return fmt.Errorf("%w: expected %v and got %v", ErrUnexpectedEventType, EventTypeNameTask, transientEvType)
		}
		container.Event, err = unmarshal[T](transientContainer.Event)
	case OperationEvent:
		if transientEvType != EventTypeNameOperation {
			return fmt.Errorf("%w: expected %v and got %v", ErrUnexpectedEventType, EventTypeNameOperation, transientEvType)
		}
		container.Event, err = unmarshal[T](transientContainer.Event)
	case DynamicEvent:
		var unmarshaledEv Event
		switch transientEvType {
		case EventTypeNameTask:
			unmarshaledEv, err = unmarshal[TaskEvent](transientContainer.Event)
		case EventTypeNameOperation:
			unmarshaledEv, err = unmarshal[OperationEvent](transientContainer.Event)
		default:
			return fmt.Errorf("%w: %v", ErrUnknownEventType, transientEvType)
		}
		container.Event = any(DynamicEvent{unmarshaledEv}).(T)
	default:
		panic(fmt.Sprintf("handling for %s events not implemented", ev.Kind()))
	}

	if err != nil {
		return err
	}

	if err := container.Event.Validate(); err != nil {
		return err
	}

	container.EventContainerMeta = transientContainer.EventContainerMeta

	return nil
}

type RequestEventContainerBase[T RequestEvent] struct {
	Event T `json:"event"`
}

// RequestEventContainer represents a request event of the specified type.
//
// Events represented in requests have a different set of fields than those represented
// in responses, so clients are expected to use this when representing an event.
//
// This type implements custom (un)marshaling logic for differentiation between
// task event types. Using this type for (un)marshaling request events is the recommended
// approach.
type RequestEventContainer[T RequestEvent] RequestEventContainerBase[T]

// NewRequestEventContainer returns a RequestEventContainer containing the specified
// event.
func NewRequestEventContainer[T RequestEvent](ev T) (RequestEventContainer[T], error) {
	if err := ev.Validate(); err != nil {
		return RequestEventContainer[T]{}, err
	}

	return RequestEventContainer[T]{
		Event: ev,
	}, nil
}

func (container RequestEventContainer[T]) MarshalJSON() ([]byte, error) {
	transientContainer := struct {
		transientEventContainerMeta
		RequestEventContainerBase[T]
	}{
		transientEventContainerMeta: transientEventContainerMetaFromEvent(container.Event),
		RequestEventContainerBase:   RequestEventContainerBase[T](container),
	}
	return json.Marshal(transientContainer)
}

func (container *RequestEventContainer[T]) UnmarshalJSON(data []byte) error {
	var transientContainer struct {
		transientEventContainerMeta
		Event json.RawMessage `json:"event"`
	}

	if err := json.Unmarshal(data, &transientContainer); err != nil {
		return err
	}

	if err := transientContainer.transientEventContainerMeta.validate(); err != nil {
		return err
	}

	var err error

	var ev RequestEvent = container.Event
	transientEvType := transientContainer.getTypeDiscriminator()

	switch ev := ev.(type) {
	case RequestTaskEvent:
		if transientEvType != EventTypeNameTask {
			return fmt.Errorf("%w: expected %v and got %v", ErrUnexpectedEventType, EventTypeNameTask, transientEvType)
		}
		container.Event, err = unmarshal[T](transientContainer.Event)
	case RequestOperationEvent:
		if transientEvType != EventTypeNameOperation {
			return fmt.Errorf("%w: expected %v and got %v", ErrUnexpectedEventType, EventTypeNameOperation, transientEvType)
		}
		container.Event, err = unmarshal[T](transientContainer.Event)
	case RequestDynamicEvent:
		var unmarshaledEv RequestEvent
		switch transientEvType {
		case EventTypeNameTask:
			unmarshaledEv, err = unmarshal[RequestTaskEvent](transientContainer.Event)
		case EventTypeNameOperation:
			unmarshaledEv, err = unmarshal[RequestOperationEvent](transientContainer.Event)
		default:
			return fmt.Errorf("%w: %v", ErrUnknownEventType, transientEvType)
		}
		container.Event = any(RequestDynamicEvent{unmarshaledEv}).(T)
	default:
		panic(fmt.Sprintf("handling for %s events not implemented", ev.Kind()))
	}

	return err
}

// TaskEventType represents the possible task event types.
type TaskEventType int

const (
	TaskEventTypeStart TaskEventType = iota
	TaskEventTypeStop
	TaskEventTypeSwitch
)

const (
	TaskEventTypeNameStart  = "start"
	TaskEventTypeNameStop   = "stop"
	TaskEventTypeNameSwitch = "switch"
)

type transientTaskEventMeta struct {
	TaskEventType     *TaskEventType `json:"task_event_type"`
	TaskEventTypeName *string        `json:"task_event_type_name"`
}

func transientTaskEventMetaFromTaskEvent(ev events.TaskEvent) transientTaskEventMeta {
	var taskEvType TaskEventType
	var taskEvTypeName string

	switch ev := ev.(type) {
	case events.StartEvent:
		taskEvType = TaskEventTypeStart
		taskEvTypeName = TaskEventTypeNameStart
	case events.StopEvent:
		taskEvType = TaskEventTypeStop
		taskEvTypeName = TaskEventTypeNameStop
	case events.SwitchEvent:
		taskEvType = TaskEventTypeSwitch
		taskEvTypeName = TaskEventTypeNameSwitch
	default:
		panic(fmt.Sprintf("handling for %s events not implemented", ev.Name()))
	}

	return transientTaskEventMeta{
		TaskEventType:     &taskEvType,
		TaskEventTypeName: &taskEvTypeName,
	}
}

func (meta transientTaskEventMeta) validate() error {
	if meta.TaskEventType == nil && meta.TaskEventTypeName == nil {
		return ErrMissingTaskEventTypeDiscriminator
	}

	if meta.TaskEventType != nil {
		if *meta.TaskEventType != TaskEventTypeStart &&
			*meta.TaskEventType != TaskEventTypeStop &&
			*meta.TaskEventType != TaskEventTypeSwitch {
			return ErrUnknownTaskEventType
		}
	}

	if meta.TaskEventTypeName != nil {
		if *meta.TaskEventTypeName != TaskEventTypeNameStart &&
			*meta.TaskEventTypeName != TaskEventTypeNameStop &&
			*meta.TaskEventTypeName != TaskEventTypeNameSwitch {
			return ErrUnknownTaskEventTypeName
		}
	}

	if meta.TaskEventType != nil && meta.TaskEventTypeName != nil {
		if (*meta.TaskEventType == TaskEventTypeStart && *meta.TaskEventTypeName != TaskEventTypeNameStart) ||
			(*meta.TaskEventType == TaskEventTypeStop && *meta.TaskEventTypeName != TaskEventTypeNameStop) ||
			(*meta.TaskEventType == TaskEventTypeSwitch && *meta.TaskEventTypeName != TaskEventTypeNameSwitch) {
			return ErrMismatchingTaskEventTypeForTaskEventTypeName
		}
	}

	return nil
}

// getTypeDiscriminator returns the type discriminator from the EventTypeName field,
// if non-null, or from the EventType field otherwise. It is meant to be called
// after unmarshaling.
//
// NOTE: meta must have been verified to be valid when calling this.
func (meta transientTaskEventMeta) getTypeDiscriminator() string {
	if meta.TaskEventTypeName != nil {
		return *meta.TaskEventTypeName
	}

	switch *meta.TaskEventType {
	case TaskEventTypeStart:
		return TaskEventTypeNameStart
	case TaskEventTypeStop:
		return TaskEventTypeNameStop
	case TaskEventTypeSwitch:
		return TaskEventTypeNameSwitch
	default:
		panic("expected valid meta")
	}
}

// TaskEventMeta represents task event type-agnostic metadata.
type TaskEventMeta struct {
	ID int `json:"id"`
}

type TaskEventBase struct {
	TaskEventMeta
	events.TaskEvent `json:"task_event"`
}

// TaskEvent represents task events.
//
// This type implements custom (un)marshaling logic that transparently handles
// differentiation between task event types. Using this type for (un)marshaling
// task events is the recommended approach.
type TaskEvent TaskEventBase

func (TaskEvent) eventMarker() {}

func (ev TaskEvent) MarshalJSON() ([]byte, error) {
	transient := struct {
		transientTaskEventMeta
		TaskEventBase
	}{
		transientTaskEventMeta: transientTaskEventMetaFromTaskEvent(ev.TaskEvent),
		TaskEventBase:          TaskEventBase(ev),
	}
	return json.Marshal(transient)
}

func unmarshalTaskEvent(evType string, data []byte) (events.TaskEvent, error) {
	switch evType {
	case TaskEventTypeNameStart:
		return unmarshal[events.StartEvent](data)
	case TaskEventTypeNameStop:
		return unmarshal[events.StopEvent](data)
	case TaskEventTypeNameSwitch:
		return unmarshal[events.SwitchEvent](data)
	default:
		return nil, fmt.Errorf("%w: %v", ErrUnknownTaskEventType, evType)
	}
}

func (ev *TaskEvent) UnmarshalJSON(data []byte) error {
	var transientEvent struct {
		transientTaskEventMeta
		TaskEventMeta
		TaskEvent json.RawMessage `json:"task_event"`
	}

	if err := json.Unmarshal(data, &transientEvent); err != nil {
		return err
	}

	if err := transientEvent.transientTaskEventMeta.validate(); err != nil {
		return err
	}

	transientEvType := transientEvent.getTypeDiscriminator()
	baseEv, err := unmarshalTaskEvent(transientEvType, transientEvent.TaskEvent)
	if err != nil {
		return err
	}

	ev.TaskEventMeta = transientEvent.TaskEventMeta
	ev.TaskEvent = baseEv

	return nil
}

type RequestTaskEventBase struct {
	events.TaskEvent `json:"task_event"`
}

// RequestTaskEvent represents request task events.
//
// This type implements custom (un)marshaling logic that transparently handles
// differentiation between request event types. Using this type for (un)marshaling
// request task events is the recommended approach.
type RequestTaskEvent RequestTaskEventBase

func (RequestTaskEvent) requestEventMarker() {}

func (ev RequestTaskEvent) MarshalJSON() ([]byte, error) {
	transient := struct {
		transientTaskEventMeta
		RequestTaskEventBase
	}{
		transientTaskEventMeta: transientTaskEventMetaFromTaskEvent(ev.TaskEvent),
		RequestTaskEventBase:   RequestTaskEventBase(ev),
	}
	return json.Marshal(transient)
}

func (ev *RequestTaskEvent) UnmarshalJSON(data []byte) error {
	var transientEvent struct {
		transientTaskEventMeta
		TaskEvent json.RawMessage `json:"task_event"`
	}
	if err := json.Unmarshal(data, &transientEvent); err != nil {
		return err
	}

	if err := transientEvent.transientTaskEventMeta.validate(); err != nil {
		return err
	}

	transientEvType := transientEvent.getTypeDiscriminator()
	baseEv, err := unmarshalTaskEvent(transientEvType, transientEvent.TaskEvent)
	if err != nil {
		return err
	}

	ev.TaskEvent = baseEv

	return nil
}

// OperationEventType represents the possible operation event types.
type OperationEventType int

const (
	OperationEventTypeCreateTask OperationEventType = iota
	OperationEventTypeRenameTask
	OperationEventTypeDeleteTask
	OperationEventTypeUndo
)

const (
	OperationEventTypeNameCreateTask = "create-task"
	OperationEventTypeNameRenameTask = "rename-task"
	OperationEventTypeNameDeleteTask = "delete-task"
	OperationEventTypeNameUndo       = "undo"
)

type transientOperationEventMeta struct {
	OperationEventType     *OperationEventType `json:"operation_event_type"`
	OperationEventTypeName *string             `json:"operation_event_type_name"`
}

func transientOperationEventMetaFromOperationEvent(ev events.OperationEvent) transientOperationEventMeta {
	var opEvType OperationEventType
	var opEvTypeName string

	switch ev := ev.(type) {
	case events.CreateTaskEvent:
		opEvType = OperationEventTypeCreateTask
		opEvTypeName = OperationEventTypeNameCreateTask
	case events.RenameTaskEvent:
		opEvType = OperationEventTypeRenameTask
		opEvTypeName = OperationEventTypeNameRenameTask
	case events.DeleteTaskEvent:
		opEvType = OperationEventTypeDeleteTask
		opEvTypeName = OperationEventTypeNameDeleteTask
	case events.UndoEvent:
		opEvType = OperationEventTypeUndo
		opEvTypeName = OperationEventTypeNameUndo
	default:
		panic(fmt.Sprintf("handling for %s events not implemented", ev.Name()))
	}

	return transientOperationEventMeta{
		OperationEventType:     &opEvType,
		OperationEventTypeName: &opEvTypeName,
	}
}

func (meta transientOperationEventMeta) validate() error {
	if meta.OperationEventType == nil && meta.OperationEventTypeName == nil {
		return ErrMissingOperationEventTypeDiscriminator
	}

	if meta.OperationEventType != nil {
		if *meta.OperationEventType != OperationEventTypeCreateTask &&
			*meta.OperationEventType != OperationEventTypeRenameTask &&
			*meta.OperationEventType != OperationEventTypeDeleteTask &&
			*meta.OperationEventType != OperationEventTypeUndo {
			return ErrUnknownOperationEventType
		}
	}

	if meta.OperationEventTypeName != nil {
		if *meta.OperationEventTypeName != OperationEventTypeNameCreateTask &&
			*meta.OperationEventTypeName != OperationEventTypeNameRenameTask &&
			*meta.OperationEventTypeName != OperationEventTypeNameDeleteTask &&
			*meta.OperationEventTypeName != OperationEventTypeNameUndo {
			return ErrUnknownOperationEventTypeName
		}
	}

	if meta.OperationEventType != nil && meta.OperationEventTypeName != nil {
		if (*meta.OperationEventType == OperationEventTypeCreateTask && *meta.OperationEventTypeName != OperationEventTypeNameCreateTask) ||
			(*meta.OperationEventType == OperationEventTypeRenameTask && *meta.OperationEventTypeName != OperationEventTypeNameRenameTask) ||
			(*meta.OperationEventType == OperationEventTypeDeleteTask && *meta.OperationEventTypeName != OperationEventTypeNameDeleteTask) ||
			(*meta.OperationEventType == OperationEventTypeUndo && *meta.OperationEventTypeName != OperationEventTypeNameUndo) {
			return ErrMismatchingOperationEventTypeForOperationEventTypeName
		}
	}

	return nil
}

// getTypeDiscriminator returns the type discriminator from the EventTypeName field,
// if non-null, or from the EventType field otherwise. It is meant to be called
// after unmarshaling.
//
// NOTE: meta must have been verified to be valid when calling this.
func (meta transientOperationEventMeta) getTypeDiscriminator() string {
	if meta.OperationEventTypeName != nil {
		return *meta.OperationEventTypeName
	}

	switch *meta.OperationEventType {
	case OperationEventTypeCreateTask:
		return OperationEventTypeNameCreateTask
	case OperationEventTypeRenameTask:
		return OperationEventTypeNameRenameTask
	case OperationEventTypeDeleteTask:
		return OperationEventTypeNameDeleteTask
	case OperationEventTypeUndo:
		return OperationEventTypeNameUndo
	default:
		panic("expected valid meta")
	}
}

type OperationEventBase struct {
	events.OperationEvent `json:"operation_event"`
}

// OperationEvent represents operation events.
//
// This type implements custom (un)marshaling logic that transparently handles
// differentiation between operation event types. Using this type for (un)marshaling
// operation events is the recommended approach.
type OperationEvent OperationEventBase

func (OperationEvent) eventMarker() {}

func (ev OperationEvent) MarshalJSON() ([]byte, error) {
	transient := struct {
		transientOperationEventMeta
		OperationEventBase
	}{
		transientOperationEventMeta: transientOperationEventMetaFromOperationEvent(ev.OperationEvent),
		OperationEventBase:          OperationEventBase(ev),
	}
	return json.Marshal(transient)
}

func (ev *OperationEvent) UnmarshalJSON(data []byte) error {
	var transientEvent struct {
		transientOperationEventMeta
		OperationEvent json.RawMessage `json:"operation_event"`
	}

	if err := json.Unmarshal(data, &transientEvent); err != nil {
		return err
	}

	if err := transientEvent.transientOperationEventMeta.validate(); err != nil {
		return err
	}

	var baseEv events.OperationEvent
	var err error

	transientEvType := transientEvent.getTypeDiscriminator()
	switch transientEvType {
	case OperationEventTypeNameCreateTask:
		baseEv, err = unmarshal[events.CreateTaskEvent](transientEvent.OperationEvent)
	case OperationEventTypeNameRenameTask:
		baseEv, err = unmarshal[events.RenameTaskEvent](transientEvent.OperationEvent)
	case OperationEventTypeNameDeleteTask:
		baseEv, err = unmarshal[events.DeleteTaskEvent](transientEvent.OperationEvent)
	case OperationEventTypeNameUndo:
		baseEv, err = unmarshal[events.UndoEvent](transientEvent.OperationEvent)
	default:
		return fmt.Errorf("%w: %v", ErrUnknownOperationEventType, transientEvType)
	}

	if err != nil {
		return err
	}

	ev.OperationEvent = baseEv

	return nil
}

// RequestOperationEvent represents request operation events.
//
// This type implements custom (un)marshaling logic that transparently handles
// differentiation between operation event types. Using this type for (un)marshaling
// events of an unknown type is the recommended approach.
type RequestOperationEvent OperationEvent

func (RequestOperationEvent) requestEventMarker() {}

// DynamicEvent represents an event that may be either an operation event or a
// task event.
//
// This type implements custom (un)marshaling logic that transparently handles
// differentiation between event types. Using this type for (un)marshaling events
// of an unknown type is the recommended approach.
type DynamicEvent struct {
	Event
}

func (DynamicEvent) eventMarker() {}

// RequestDynamicEvent represents a request event that may be either an operation
// event or a task event.
//
// This type implements custom (un)marshaling logic that transparently handles
// differentiation between event types. Using this type for (un)marshaling request
// events of an unknown type is the recommended approach.
type RequestDynamicEvent struct {
	RequestEvent
}

func (RequestDynamicEvent) requestEventMarker() {}
