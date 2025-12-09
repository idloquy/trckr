// Package client provides functionality for interacting with trckr servers.
package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"time"

	"github.com/idloquy/trckr/pkg/api"
	"github.com/idloquy/trckr/pkg/events"
	"github.com/idloquy/trckr/pkg/events/validator"
	"github.com/idloquy/trckr/pkg/tasks"
)

const APIVersion = 1

// An InvalidParamsError indicates that a parameter passed in by the user is invalid.
// The underlying error describes the particular reason why the parameter is invalid.
type InvalidParamsError struct {
	Err error
}

func (e *InvalidParamsError) Error() string {
	return fmt.Sprintf("invalid params: %v", e.Err)
}

func (e *InvalidParamsError) Unwrap() error {
	return e.Err
}

// An ApplicationError represents an error returned by the server.
type ApplicationError struct {
	StatusCode int
	Message    string
}

func (e *ApplicationError) Error() string {
	return e.Message
}

// A GeneralBugError represents an error that means there is a bug either in this
// library or in the server.
type GeneralBugError struct {
	Err error
}

func (e *GeneralBugError) Error() string {
	return fmt.Sprintf("client or server bug encountered: %v", e.Err)
}

func (e *GeneralBugError) Unwrap() error {
	return e.Err
}

// An InvalidResponseError indicates that the response returned by the server is
// invalid for some reason. The underlying error describes the particular reason
// why the response is invalid.
type InvalidResponseError struct {
	Response []byte
	Reason   error
}

func (e *InvalidResponseError) Error() string {
	return fmt.Sprintf("invalid response: %s", e.Reason)
}

var (
	ErrInvalidServer  = errors.New("server is invalid")
	ErrOutdatedServer = errors.New("server is outdated")
	ErrOutdatedClient = errors.New("client is outdated")
	ErrEmptyResponse  = errors.New("empty response")
	ErrNoContent      = errors.New("no content")
)

// The Client is responsible for all interaction with the server, from creating
// and starting tasks to listening to events.
type Client struct {
	httpClient *http.Client
	server     *url.URL
}

// A ClientOption represents a configuration option that can be passed in to the
// Client constructor.
type ClientOption func(*Client)

// WithCustomHTTPClient returns a ClientOption that sets the Client's underlying
// HTTP client to httpClient.
func WithCustomHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// New returns a Client set up to interact with the server at the specified URL.
// The URL is automatically validated to refer to a valid server.
func New(server *url.URL, opts ...ClientOption) (*Client, error) {
	c := &Client{
		httpClient: http.DefaultClient,
		server:     server,
	}

	for _, f := range opts {
		f(c)
	}

	if err := c.Validate(); err != nil {
		switch {
		case errors.Is(err, ErrOutdatedServer):
			fallthrough
		case errors.Is(err, ErrInvalidServer):
			return nil, err
		default:
			return nil, fmt.Errorf("failed to get server info: %v", err)
		}
	}

	return c, nil
}

func doRequest(httpClient *http.Client, uri *url.URL, method string, msg any) (*http.Response, error) {
	var req *http.Request
	if method != http.MethodGet && method != http.MethodDelete {
		var err error

		encodedMsg, err := json.Marshal(&msg)
		if err != nil {
			return nil, err
		}

		body := bytes.NewReader(encodedMsg)
		req, err = http.NewRequest(method, uri.String(), body)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %v", err)
		}
		if method == http.MethodPatch {
			req.Header.Add("Content-Type", "application/merge-patch+json")
		} else {
			req.Header.Add("Content-Type", "application/json")
		}
	} else {
		var err error
		req, err = http.NewRequest(method, uri.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %v", err)
		}
	}

	return httpClient.Do(req)
}

func makeRequest(httpClient *http.Client, uri *url.URL, method string, msg any) error {
	res, err := doRequest(httpClient, uri, method, msg)
	if err != nil {
		return err
	}

	if res.ContentLength == 0 {
		return nil
	}

	if res.StatusCode == http.StatusNoContent {
		return nil
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	var responseContainer api.ResponseContainer
	if err := json.Unmarshal(resBody, &responseContainer); err != nil {
		switch {
		case errors.Is(err, api.ErrUnknownResponseStatus):
			return &GeneralBugError{
				Err: err,
			}
		default:
			return &InvalidResponseError{
				Response: resBody,
				Reason:   err,
			}
		}
	}

	switch response := responseContainer.Response.(type) {
	case api.SuccessResponseJSON:
		return nil
	case api.ErrorResponseJSON:
		var container api.ErrorResponseContainer
		if err := json.Unmarshal(response.Data, &container); err != nil {
			return &InvalidResponseError{
				Response: resBody,
				Reason:   err,
			}
		}
		e := &ApplicationError{StatusCode: res.StatusCode, Message: container.Error}
		return e
	default:
		panic(fmt.Sprintf("handling for response type %s not implemented", reflect.TypeOf(response).Name()))
	}
}

func makeRequestWithResponse[T api.SuccessResponseData](httpClient *http.Client, uri *url.URL, method string, msg any) (T, error) {
	var data T

	res, err := doRequest(httpClient, uri, method, msg)
	if err != nil {
		return data, err
	}

	if res.ContentLength == 0 {
		return data, ErrEmptyResponse
	}

	if res.StatusCode == http.StatusNoContent {
		return data, ErrNoContent
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return data, fmt.Errorf("failed to read response body: %v", err)
	}

	var responseContainer api.ResponseContainer
	if err := json.Unmarshal(resBody, &responseContainer); err != nil {
		switch {
		case errors.Is(err, api.ErrUnknownResponseStatus):
			return data, &GeneralBugError{
				Err: err,
			}
		default:
			return data, &InvalidResponseError{
				Response: resBody,
				Reason:   err,
			}
		}
	}

	switch response := responseContainer.Response.(type) {
	case api.SuccessResponseJSON:
		var container api.SuccessResponseContainer[T]
		if err := json.Unmarshal(response.Data, &container); err != nil {
			switch {
			case errors.Is(err, api.ErrUnknownEventType):
				fallthrough
			case errors.Is(err, api.ErrUnknownEventTypeName):
				fallthrough
			case errors.Is(err, api.ErrUnknownTaskEventType):
				fallthrough
			case errors.Is(err, api.ErrUnknownTaskEventTypeName):
				fallthrough
			case errors.Is(err, api.ErrUnknownOperationEventType):
				fallthrough
			case errors.Is(err, api.ErrUnknownOperationEventTypeName):
				return data, &GeneralBugError{Err: err}
			default:
				return data, &InvalidResponseError{
					Response: resBody,
					Reason:   err,
				}
			}
		}
		return container.Data, nil
	case api.ErrorResponseJSON:
		var container api.ErrorResponseContainer
		if err := json.Unmarshal(response.Data, &container); err != nil {
			return data, &InvalidResponseError{
				Response: resBody,
				Reason:   err,
			}
		}
		e := &ApplicationError{StatusCode: res.StatusCode, Message: container.Error}
		return data, e
	default:
		panic(fmt.Sprintf("handling for response type %s not implemented", reflect.TypeOf(response).Name()))
	}
}

// Validate verifies that the server that the Client is configured to use is a
// valid trckr server.
func (c *Client) Validate() error {
	uri := c.server.JoinPath("/info")
	res, err := makeRequestWithResponse[api.InfoResponse](c.httpClient, uri, http.MethodGet, nil)
	if err != nil {
		var invalidResponseErr *InvalidResponseError
		if errors.As(err, &invalidResponseErr) {
			return ErrInvalidServer
		}

		var applicationErr *ApplicationError
		if errors.As(err, &applicationErr) && applicationErr.StatusCode == http.StatusNotFound {
			return ErrInvalidServer
		}

		return err
	}
	if res.Name != api.ServerName {
		return ErrInvalidServer
	}

	if res.Version < APIVersion {
		return ErrOutdatedServer
	}

	if res.Version > APIVersion {
		return ErrOutdatedClient
	}

	return nil
}

// CreateTask creates a task named taskName.
func (c *Client) CreateTask(taskName string) error {
	uri := c.server.JoinPath("/v1/tasks")
	msg := api.CreateTaskMsg{Name: taskName}
	_, err := makeRequestWithResponse[api.TaskResponse](c.httpClient, uri, http.MethodPost, msg)
	if errors.Is(err, ErrNoContent) || errors.Is(err, ErrEmptyResponse) {
		return nil
	}
	return err
}

func (c *Client) addEvent(ev events.TaskEvent) error {
	uri := c.server.JoinPath("/v1/events")

	msg, err := api.NewRequestEventContainer[api.RequestTaskEvent](api.RequestTaskEvent{TaskEvent: ev})
	if err != nil {
		return &InvalidParamsError{Err: err}
	}

	_, err = makeRequestWithResponse[api.TaskEventResponse](c.httpClient, uri, http.MethodPost, msg)
	return err
}

// StartTask starts the specified task, using stopReason as the stop reason if
// non-empty.
func (c *Client) StartTask(taskName string, stopReason string) error {
	ev, err := events.NewStartEvent(taskName, stopReason)
	if err != nil {
		return &InvalidParamsError{Err: err}
	}

	return c.addEvent(ev)
}

// StopTask stops the specified task.
func (c *Client) StopTask(taskName string) error {
	ev, err := events.NewStopEvent(taskName)
	if err != nil {
		return &InvalidParamsError{Err: err}
	}

	return c.addEvent(ev)
}

// SwitchTask switches from oldTask to newTask.
func (c *Client) SwitchTask(oldTask string, newTask string) error {
	ev, err := events.NewSwitchEvent(oldTask, newTask)
	if err != nil {
		return &InvalidParamsError{Err: err}
	}

	return c.addEvent(ev)
}

// ListTasks lists the existing tasks.
func (c *Client) ListTasks() ([]tasks.Task, error) {
	uri := c.server.JoinPath("/v1/tasks")

	res, err := makeRequestWithResponse[api.TaskListResponse](c.httpClient, uri, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}

	return res.Tasks, nil
}

// RenameTask renames the specified task.
func (c *Client) RenameTask(task string, newName string) error {
	uri := c.server.JoinPath(fmt.Sprintf("/v1/tasks/%s", url.PathEscape(task)))
	msg := api.UpdateTaskMsg{Name: &newName}
	_, err := makeRequestWithResponse[api.TaskResponse](c.httpClient, uri, http.MethodPatch, msg)
	return err
}

// DeleteTask deletes the specified task.
func (c *Client) DeleteTask(task string) error {
	uri := c.server.JoinPath(fmt.Sprintf("/v1/tasks/%s", url.PathEscape(task)))
	err := makeRequest(c.httpClient, uri, http.MethodDelete, nil)
	return err
}

// GetEvent returns the event corresponding to the specified id.
func (c *Client) GetEvent(id int) (api.EventContainer[api.TaskEvent], error) {
	uri := c.server.JoinPath(fmt.Sprintf("/v1/events/%d", id))
	res, err := makeRequestWithResponse[api.TaskEventResponse](c.httpClient, uri, http.MethodGet, nil)
	if err != nil {
		return api.EventContainer[api.TaskEvent]{}, err
	}
	return res.TaskEvent, nil
}

// GetLatestEvent returns the latest event.
func (c *Client) GetLatestEvent() (api.EventContainer[api.TaskEvent], error) {
	uri := c.server.JoinPath("/v1/events/latest")
	res, err := makeRequestWithResponse[api.TaskEventResponse](c.httpClient, uri, http.MethodGet, nil)
	if err != nil {
		return api.EventContainer[api.TaskEvent]{}, err
	}
	return res.TaskEvent, nil
}

// UpdateEvent updates the event corresponding to the specified id to updatedEvent.
func (c *Client) UpdateEvent(id int, updatedEvent events.TaskEvent) error {
	uri := c.server.JoinPath(fmt.Sprintf("/v1/events/%d", id))
	msg := api.TaskEvent{TaskEvent: updatedEvent}
	err := makeRequest(c.httpClient, uri, http.MethodPatch, msg)
	return err
}

// UndoEvent undoes the event corresponding to the specified id.
func (c *Client) UndoEvent(id int) error {
	uri := c.server.JoinPath(fmt.Sprintf("/v1/events/%d", id))
	err := makeRequest(c.httpClient, uri, http.MethodDelete, nil)
	return err
}

func validateHistoryEvents(apiEvs []api.EventContainer[api.TaskEvent]) error {
	if len(apiEvs) == 0 {
		return nil
	}

	var v *validator.Validator

	firstEv := apiEvs[0]
	if firstEv.Event.ID > 1 {
		v = validator.ForPartialSequence()
	} else {
		v = validator.New()
	}

	var evs []events.TaskEvent
	for _, ev := range apiEvs {
		if err := v.ValidateSequence(evs, ev.Event.TaskEvent); err != nil {
			return err
		}
		evs = append(evs, ev.Event.TaskEvent)
	}

	return nil
}

// GetHistoryEvents fetches the history of all events that have occurred.
func (c *Client) GetHistoryEvents() ([]api.EventContainer[api.TaskEvent], error) {
	uri := c.server.JoinPath("/v1/events")
	res, err := makeRequestWithResponse[api.TaskEventListResponse](c.httpClient, uri, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}

	if err := validateHistoryEvents(res.TaskEvents); err != nil {
		return nil, err
	}

	return res.TaskEvents, nil
}

// GetHistoryDateEvents fetches the history of the events that have occurred from
// fromDate to toDate.
func (c *Client) GetHistoryDateEvents(fromDate, toDate time.Time) ([]api.EventContainer[api.TaskEvent], error) {
	uri := c.server.JoinPath("/v1/events")
	query := uri.Query()
	query.Add("from_date", strconv.FormatInt(fromDate.Unix(), 10))
	query.Add("to_date", strconv.FormatInt(toDate.Unix(), 10))
	uri.RawQuery = query.Encode()

	res, err := makeRequestWithResponse[api.TaskEventListResponse](c.httpClient, uri, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}

	if err := validateHistoryEvents(res.TaskEvents); err != nil {
		return nil, err

	}
	return res.TaskEvents, nil
}

// GetHistoryDaysEvents fetches the history of the events that have occurred within
// the specified number of days, where a day is delimited by the task specified
// in dayDivider.
func (c *Client) GetHistoryDaysEvents(days int, dayDivider string) ([]api.EventContainer[api.TaskEvent], error) {
	uri := c.server.JoinPath("/v1/events")
	query := uri.Query()
	query.Add("days", strconv.Itoa(days))
	query.Add("day_divider", dayDivider)
	uri.RawQuery = query.Encode()

	res, err := makeRequestWithResponse[api.TaskEventListResponse](c.httpClient, uri, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}

	if err := validateHistoryEvents(res.TaskEvents); err != nil {
		return nil, err
	}

	return res.TaskEvents, nil
}

// A PartialProductivePeriod represents an ongoing period during which some task
// is running.
type PartialProductivePeriod struct {
	Task      string
	StartTime time.Time
}

// A ProductivePeriod represents a length of time during which some task is running.
type ProductivePeriod struct {
	Task      string
	StartTime time.Time
	StopTime  time.Time
}

// A NonProductivePeriod represents a length of time during which no task is running.
type NonProductivePeriod struct {
	Reason    string
	StartTime time.Time
	StopTime  time.Time
}

// A Period represents a length of time during which a task may either be running
// or not.
type Period interface {
	periodMarker()
	IsProductive() bool
	StartedAt() time.Time
}

// A FullyBoundedPeriod is a Period which has both a start and an end.
type FullyBoundedPeriod interface {
	Period
	StoppedAt() time.Time
}

func (PartialProductivePeriod) periodMarker() {}

// IsProductive returns whether some task is running during the period.
func (PartialProductivePeriod) IsProductive() bool {
	return true
}

// StartedAt returns the time at which the period started.
func (p PartialProductivePeriod) StartedAt() time.Time {
	return p.StartTime
}

func (ProductivePeriod) periodMarker() {}

// IsProductive returns whether some task is running during the period.
func (ProductivePeriod) IsProductive() bool {
	return true
}

// StartedAt returns the time at which the period started.
func (p ProductivePeriod) StartedAt() time.Time {
	return p.StartTime
}

// StoppedAt returns the time at which the period stopped.
func (p ProductivePeriod) StoppedAt() time.Time {
	return p.StopTime
}

// Duration returns the duration of the period.
func (p ProductivePeriod) Duration() time.Duration {
	return p.StopTime.Sub(p.StartTime)
}

func (NonProductivePeriod) periodMarker() {}

// IsProductive returns whether some task is running during the period.
func (NonProductivePeriod) IsProductive() bool {
	return false
}

// StartedAt returns the time at which the period started.
func (p NonProductivePeriod) StartedAt() time.Time {
	return p.StartTime
}

// StoppedAt returns the time at which the period stopped.
func (p NonProductivePeriod) StoppedAt() time.Time {
	return p.StopTime
}

// Duration returns the duration of the period.
func (p NonProductivePeriod) Duration() time.Duration {
	return p.StopTime.Sub(p.StartTime)
}

// TaskStats holds statistics for a given task.
type TaskStats struct {
	Task     string
	Duration time.Duration
}

// TaskHistory represents the history of operations for all tasks, in chronological
// order.
type TaskHistory struct {
	events  []api.EventContainer[api.TaskEvent]
	periods []Period
}

// NOTE: evs must have been validated to be a proper event sequence when calling
// this.
func historyFromTaskEvents(evs []api.EventContainer[api.TaskEvent]) (*TaskHistory, error) {
	var periods []Period
	for i := 0; i < len(evs); {
		container := evs[i]
		ev := container.Event.TaskEvent

		startTime := container.At
		if startEv, ok := ev.(events.StartEvent); ok {
			if i > 0 {
				// Note that the previous event is necessarily
				// a stop event, because evs is assumed to be a
				// proper sequence.
				prevStopTime := evs[i-1].At
				period := NonProductivePeriod{
					Reason:    startEv.StopReason,
					StartTime: prevStopTime,
					StopTime:  startTime,
				}
				periods = append(periods, period)
			}
		}

		var startedTask string
		// Note that the current event is assumed not to be a stop event,
		// because we're assuming evs to be a proper sequence and are manually
		// skipping events relying on that assumption.
		switch ev := ev.(type) {
		case events.StartEvent:
			startedTask = ev.Task
		case events.SwitchEvent:
			startedTask = ev.NewTask
		default:
			panic(fmt.Sprintf("handling for non-stop %s events not implemented", ev.Name()))
		}

		var period Period
		if len(evs) > i+1 {
			nextContainer := evs[i+1]
			nextEv := nextContainer.Event.TaskEvent
			stopTime := nextContainer.At
			period = ProductivePeriod{
				Task:      startedTask,
				StartTime: startTime,
				StopTime:  stopTime,
			}

			// Skip non-initiator events.
			switch nextEv.(type) {
			case events.StartEvent:
			case events.SwitchEvent:
			case events.StopEvent:
				i += 1
			default:
				panic(fmt.Sprintf("handling for %s events not implemented", ev.Name()))
			}
		} else {
			period = PartialProductivePeriod{
				Task:      startedTask,
				StartTime: startTime,
			}
		}
		periods = append(periods, period)

		i += 1
	}
	return &TaskHistory{events: evs, periods: periods}, nil
}

// IsEmpty returns whether there are no periods in the history.
func (h *TaskHistory) IsEmpty() bool {
	return len(h.periods) == 0
}

// Periods returns the periods in the history.
func (h *TaskHistory) Periods() []Period {
	periods := make([]Period, len(h.periods))
	copy(periods, h.periods)
	return periods
}

// Events returns the events in the history.
func (h *TaskHistory) Events() []api.EventContainer[api.TaskEvent] {
	events := make([]api.EventContainer[api.TaskEvent], len(h.events))
	copy(events, h.events)
	return events
}

// SplitByDay splits the history by day.
func (h *TaskHistory) SplitByDay() []*TaskHistory {
	if len(h.periods) == 0 {
		return nil
	}

	var splittedHistory []*TaskHistory

	var firstPerStart time.Time
	switch per := h.periods[0].(type) {
	case ProductivePeriod:
		firstPerStart = per.StartTime
	case PartialProductivePeriod:
		firstPerStart = per.StartTime
	default:
		panic(fmt.Sprintf("handling for period type %s not implemented", reflect.TypeOf(per).Name()))
	}
	firstDayStart := time.Date(firstPerStart.Year(), firstPerStart.Month(), firstPerStart.Day(), 0, 0, 0, 0, time.Local)

	currentDayEnd := firstDayStart.Add(time.Hour * 24)
	var currentDay []Period

	for _, period := range h.periods {
		var startTime time.Time
		switch per := period.(type) {
		case ProductivePeriod:
			startTime = per.StartTime
		case PartialProductivePeriod:
			startTime = per.StartTime
		case NonProductivePeriod:
			startTime = per.StartTime
		default:
			panic(fmt.Sprintf("handling for period type %s not implemented", reflect.TypeOf(per).Name()))
		}

		if startTime.Compare(currentDayEnd) >= 0 {
			nonProductivePer, isNonProductivePer := period.(NonProductivePeriod)
			if isNonProductivePer {
				currentDay = append(currentDay, nonProductivePer)
			}

			dayHistory := &TaskHistory{periods: currentDay}
			splittedHistory = append(splittedHistory, dayHistory)

			currentDay = nil
			currentDayStart := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, time.Local)
			currentDayEnd = currentDayStart.Add(time.Hour * 24)
			if isNonProductivePer {
				continue
			}
		}

		currentDay = append(currentDay, period)
	}

	if len(currentDay) > 0 {
		leftoverDayHistory := &TaskHistory{periods: currentDay}
		splittedHistory = append(splittedHistory, leftoverDayHistory)
	}

	return splittedHistory
}

// SplitByTask splits the history using the specified task as the separator.
func (h *TaskHistory) SplitByTask(task string) []*TaskHistory {
	if len(h.periods) == 0 {
		return nil
	}

	var splittedHistory []*TaskHistory

	var currentDay []Period
	waitingForNonProductive := false
	for i, period := range h.periods {
		currentDay = append(currentDay, period)

		var currentTask string
		switch per := period.(type) {
		case ProductivePeriod:
			currentTask = per.Task
		case PartialProductivePeriod:
			currentTask = per.Task
		case NonProductivePeriod:
		default:
			panic(fmt.Sprintf("handling for period type %s not implemented", reflect.TypeOf(per).Name()))
		}

		if !waitingForNonProductive && currentTask != task {
			continue
		}

		if currentTask == task {
			if i < len(h.periods)-1 {
				nextPer := h.periods[i+1]
				switch nextPer.(type) {
				case ProductivePeriod:
				case PartialProductivePeriod:
				case NonProductivePeriod:
					waitingForNonProductive = true
					continue
				default:
					panic(fmt.Sprintf("handling for period type %s not implemented", reflect.TypeOf(period).Name()))
				}
			}
		}

		dayHistory := &TaskHistory{periods: currentDay}
		splittedHistory = append(splittedHistory, dayHistory)
		currentDay = nil

		waitingForNonProductive = false

	}

	if len(currentDay) != 0 {
		leftoverDayHistory := &TaskHistory{periods: currentDay}
		splittedHistory = append(splittedHistory, leftoverDayHistory)
	}

	return splittedHistory
}

// TaskStats returns the TaskStats for each task present in the history.
func (h *TaskHistory) TaskStats() []TaskStats {
	stats := make(map[string]time.Duration)
	for _, period := range h.periods {
		if productivePeriod, ok := period.(ProductivePeriod); ok {
			stats[productivePeriod.Task] += productivePeriod.Duration()
		}
	}

	var sortedStats []TaskStats
	for task, duration := range stats {
		stats := TaskStats{Task: task, Duration: duration}
		sortedStats = append(sortedStats, stats)
	}
	slices.SortFunc(sortedStats, func(stats1 TaskStats, stats2 TaskStats) int {
		if stats1.Duration < stats2.Duration {
			return -1
		} else if stats1.Duration > stats2.Duration {
			return 1
		} else {
			return 0
		}
	})

	return sortedStats
}

// GetHistory returns the TaskHistory corresponding to all events that have occurred.
func (c *Client) GetHistory() (*TaskHistory, error) {
	evs, err := c.GetHistoryEvents()
	if err != nil {
		return nil, err
	}

	return historyFromTaskEvents(evs)
}

// GetHistoryDate returns the TaskHistory corresponding to the events that have
// occurred from fromDate to toDate.
func (c *Client) GetHistoryDate(fromDate time.Time, toDate time.Time) (*TaskHistory, error) {
	evs, err := c.GetHistoryDateEvents(fromDate, toDate)
	if err != nil {
		return nil, err
	}

	return historyFromTaskEvents(evs)
}

// GetHistoryDays returns the TaskHistory corresponding to the events that have
// occurred within the specified number of days, where a day is delimited by the
// task specified in dayDivider.
func (c *Client) GetHistoryDays(days int, dayDivider string) (*TaskHistory, error) {
	evs, err := c.GetHistoryDaysEvents(days, dayDivider)
	if err != nil {
		return nil, err
	}

	return historyFromTaskEvents(evs)
}

// GetEventReceiver returns an EventReceiver configured to use the same server
// as the Client.
func (c *Client) GetEventReceiver(eventReceiverOpts ...EventReceiverOption) (*EventReceiver, error) {
	wsServer := c.server
	wsServer.Scheme = "ws"
	receiver, err := newEventReceiver(c.server, eventReceiverOpts...)
	return receiver, err

}

// An EventReceiverOption represents a configuration option that can be passed
// in to the EventReceiver constructor.
type EventReceiverOption = func(*EventReceiver)

// EventReceiverWithCustomDialer returns an EventReceiverOption that sets the EventReceiver's
// underlying websocket dialer to the specified dialer.
func EventReceiverWithCustomDialer(dialer *websocket.Dialer) EventReceiverOption {
	return func(receiver *EventReceiver) {
		receiver.dialer = dialer
	}
}

// An EventReceiver provides an interface for event reception.
type EventReceiver struct {
	dialer *websocket.Dialer
	conn   *websocket.Conn
}

func newEventReceiver(server *url.URL, opts ...EventReceiverOption) (*EventReceiver, error) {
	receiver := &EventReceiver{
		dialer: websocket.DefaultDialer,
	}

	for _, f := range opts {
		f(receiver)
	}

	uri := server.JoinPath("/ws/events")
	conn, _, err := receiver.dialer.Dial(uri.String(), nil)
	if err != nil {
		return nil, err
	}

	receiver.conn = conn
	return receiver, nil
}

// Next waits for the next event to be transmitted by the server and returns it.
func (receiver *EventReceiver) Next() (api.EventContainer[api.DynamicEvent], error) {
	var container api.EventContainer[api.DynamicEvent]
	err := receiver.conn.ReadJSON(&container)
	if err != nil {
		switch {
		case errors.Is(err, api.ErrUnknownEventType):
			fallthrough
		case errors.Is(err, api.ErrUnknownEventTypeName):
			fallthrough
		case errors.Is(err, api.ErrUnknownTaskEventType):
			fallthrough
		case errors.Is(err, api.ErrUnknownTaskEventTypeName):
			fallthrough
		case errors.Is(err, api.ErrUnknownOperationEventType):
			fallthrough
		case errors.Is(err, api.ErrUnknownOperationEventTypeName):
			return container, &GeneralBugError{
				Err: err,
			}
		}
	}
	return container, err
}
