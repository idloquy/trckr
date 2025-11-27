package database

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	gosqlite "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	"github.com/idloquy/trckr/pkg/events"
	"github.com/idloquy/trckr/pkg/events/validator"
	"github.com/idloquy/trckr/pkg/tasks"
)

var (
	ErrTaskAlreadyExists    = errors.New("task already exists")
	ErrTaskNameNotAvailable = errors.New("task name not available")
	ErrInvalidEventSequence = errors.New("invalid event sequence")
	ErrNoSuchEvent          = errors.New("no such event")
	ErrNoSuchTask           = errors.New("no such task")
	ErrNoEvents             = errors.New("no events")
	ErrNotLatestEvent       = errors.New("specified event is not the latest event")
	ErrMismatchingEventType = errors.New("mismatching event type")
	ErrRenameTaskSameName       = errors.New("attempted to rename a task to its own name")
	ErrDeleteTaskRunning    = errors.New("task running")
	ErrNonUpdatableEvent    = errors.New("non-updatable event")
	ErrIllegalUpdate        = errors.New("illegal update")
)

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

// A DatabaseGeneralError represents a general database-related error condition.
// The underlying error specifies the particular error condition.
type DatabaseGeneralError struct {
	Err error
}

func (e *DatabaseGeneralError) Error() string {
	return fmt.Sprintf("general database error: %v", e.Err)
}

func (e *DatabaseGeneralError) Unwrap() error {
	return e.Err
}

// The Database interface specifies the functionality that a database implementation
// should provide.
type Database interface {
	Subscribe() chan Event
	Unsubscribe(chan Event)
	AddEvent(event events.TaskEvent) (TaskEvent, error)
	GetEvents() ([]TaskEvent, error)
	GetEvent(id int) (TaskEvent, error)
	GetLatestEvent() (TaskEvent, error)
	UpdateEvent(id int, newEvent events.TaskEvent) (TaskEvent, error)
	UndoEvent(id int) error

	CreateTask(name string) (tasks.Task, error)
	GetTask(name string) (tasks.Task, error)
	GetTasks() ([]tasks.Task, error)
	RenameTask(task string, newName string) (tasks.Task, error)
	DeleteTask(name string) error
}

type eventSubscriber struct {
	channel chan Event
	// cancel signals the closure of the channel internally. It is used by
	// the unsubscribe method, which is called by EventPubSub's method of the
	// same name when subscribers don't want to listen any longer.
	cancel chan struct{}
}

func newEventSubscriber() eventSubscriber {
	return eventSubscriber{
		channel: make(chan Event),
		cancel:  make(chan struct{}),
	}
}

func (sub *eventSubscriber) publish(ev Event) {
	select {
	case sub.channel <- ev:
	case <-sub.cancel:
	}
}

func (sub *eventSubscriber) unsubscribe() {
	close(sub.cancel)
}

// An EventPubSub allows for events to be published and listened to in such a manner
// that ordering is guaranteed to be consistent across subscribers.
type EventPubSub struct {
	// NOTE: This being an RWMutex (as opposed to a mere Mutex) is absolutely
	// required for proper functioning, but its value may not be immediately
	// apparent. Using an RWMutex allows us to avoid subtle deadlocks when an
	// event is published while one of the subscribers is attempting to unsubscribe.
	// With a normal Mutex, a deadlock would ensue because both publish and
	// Unsubscribe would need the lock, but Unsubscribe would be left waiting
	// for publish to complete. Simultaneously, publish would be waiting for
	// the send to complete, which would never occur because the subscriber
	// would not be waiting for events (because it's calling Unsubscribe),
	// and Unsubscribe is kept hanging, and is therefore unable to send a cancellation
	// signal.
	m             sync.RWMutex
	subscriptions []eventSubscriber
	publishCh     chan Event
}

// NewEventPubSub returns a new EventPubSub initialized with no subscribers.
func NewEventPubSub() *EventPubSub {
	var subscriptions []eventSubscriber
	ps := &EventPubSub{
		subscriptions: subscriptions,
		publishCh:     make(chan Event),
	}
	go ps.handlePublishing()
	return ps
}

// Subscribe returns a channel from which events can be received.
//
// Unsubscribe must be called when the channel is no longer needed.
func (ps *EventPubSub) Subscribe() chan Event {
	subscription := newEventSubscriber()

	ps.m.Lock()
	defer ps.m.Unlock()
	ps.subscriptions = append(ps.subscriptions, subscription)
	return subscription.channel
}

// Unsubscribe performs the removal of a subscriber from the list of subscribers.
// It must be called as soon as possible once receiving events ceases to be desired.
func (ps *EventPubSub) Unsubscribe(subscription chan Event) {
	ps.m.RLock()
	idx := slices.IndexFunc(ps.subscriptions, func(sub eventSubscriber) bool {
		return sub.channel == subscription
	})
	if idx == -1 {
		ps.m.RUnlock()
		return
	}

	ps.subscriptions[idx].unsubscribe()
	ps.m.RUnlock()

	// Note that, even if a publish call makes it before this lock and after
	// the unsubscribe call above, proper functioning is still retained, as
	// unsubscribe guarantees that any further calls to publish return immediately.
	ps.m.Lock()
	ps.subscriptions = slices.Delete(ps.subscriptions, idx, idx+1)
	ps.m.Unlock()
}

func (ps *EventPubSub) publish(ev Event) {
	ps.publishCh <- ev
}

// handlePublishing ensures that multiple publish operations can't run concurrently,
// thereby retaining event ordering across subscribers. It achieves that by being
// a central broker, which runs in a goroutine and listens to events in order to
// publish them.
// NOTE: The reason why this is required has much to do with the requirement of
// using an RWMutex. Specifically, the issue this solves is only possible at all
// because locking the RWMutex for reading doesn't guarantee exclusive acquisition
// (as would be the case for Mutex's Lock method), which opens the possibility of
// multiple publish calls running concurrently.
func (ps *EventPubSub) handlePublishing() {
	for {
		ev := <-ps.publishCh
		// Note that, even though we're only acquiring a read lock, eventSubscriber
		// is thread-safe, so this is still safe.
		ps.m.RLock()
		for _, sub := range ps.subscriptions {
			sub.publish(ev)
		}
		ps.m.RUnlock()
	}
}

// The Event interface represents the possible event types and defines functionality
// common to all events.
type Event interface {
	eventMarker()
	At() time.Time
}

// A TaskEvent is a task event with ID and time metadata.
type TaskEvent struct {
	ID   int
	Time time.Time
	events.TaskEvent
}

func (ev TaskEvent) eventMarker() {}

// At returns the time at which the event occurred.
func (ev TaskEvent) At() time.Time {
	return ev.Time
}

// An OperationEvent is an operation event with time metadata.
type OperationEvent struct {
	Time time.Time
	events.OperationEvent
}

func (ev OperationEvent) eventMarker() {}

// At returns the time at which the event occurred.
func (ev OperationEvent) At() time.Time {
	return ev.Time
}

type sqlLockedRetrier interface {
	retry(func() error) error
}

type sqlDatabase struct {
	*EventPubSub
	lockedRetrier sqlLockedRetrier
	db            *gorm.DB
}

// newSQLDatabase returns a sqlDatabase using the specified db and lockedRetrier.
//
// The database must be configured for foreign key support and error translation, and logging should be disabled. Concurrent transactions should also be disabled.
func newSQLDatabase(db *gorm.DB, lockedRetrier sqlLockedRetrier) (*sqlDatabase, error) {
	if err := db.AutoMigrate(&dbStartEvent{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&dbStopEvent{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&dbSwitchEvent{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&proxyEvent{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&dbTask{}); err != nil {
		return nil, err
	}

	return &sqlDatabase{
		EventPubSub:   NewEventPubSub(),
		lockedRetrier: lockedRetrier,
		db:            db,
	}, nil
}

// CreateTask creates a task identified by name.
func (db *sqlDatabase) CreateTask(name string) (tasks.Task, error) {
	task, err := newDBTask(name)
	if err != nil {
		return tasks.Task{}, &InvalidParamsError{Err: err}
	}

	ctx := context.Background()
	err = db.lockedRetrier.retry(func() error {
		return db.db.Transaction(func(tx *gorm.DB) error {
			if err := gorm.G[dbTask](tx).Create(ctx, &task); err != nil {
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					// Soft-deleted tasks may also lead to
					// duplicated key errors. We use the RowsAffected
					// field of the response to determine whether
					// the task has been soft deleted, and
					// only return an error if not.
					res := tx.Unscoped().Model(&task).Where("name = ?", name).Where("deleted_at IS NOT NULL").Update("deleted_at", nil)
					if res.Error != nil {
						return &DatabaseGeneralError{Err: res.Error}
					}

					if res.RowsAffected == 0 {
						return ErrTaskAlreadyExists
					}
				} else {
					return &DatabaseGeneralError{Err: err}
				}
			}

			ev, _ := events.NewCreateTaskEvent(task.task())
			db.publish(OperationEvent{
				Time:           time.Now(),
				OperationEvent: ev,
			})

			return nil
		})
	})
	if err != nil {
		return tasks.Task{}, err
	}
	return task.task(), nil
}

// NOTE: This must be called from within a transaction.
func addEvent[T polymorphicEventModel](db *gorm.DB, ev T) (TaskEvent, error) {
	var resEv TaskEvent

	evs, err := getBaseTaskEvents(db)
	if err != nil {
		return resEv, err
	}

	if err := validator.DefaultValidator.ValidateSequence(evs, ev.event()); err != nil {
		return resEv, fmt.Errorf("%w: %w", ErrInvalidEventSequence, err)
	}

	res := db.Model(&ev).Clauses(clause.Returning{}).Create(&ev)
	if res.Error != nil {
		return resEv, &DatabaseGeneralError{Err: res.Error}
	}

	// The event ID is not populated on insertions with RETURNING clauses due
	// to being a pointer, so we query and set it manually in the returned
	// struct.
	latestEv, err := getLatestProxyEvent(db)
	if err != nil {
		return resEv, err
	}

	resEv = TaskEvent{
		ID:        *latestEv.ID,
		Time:      ev.at(),
		TaskEvent: ev.event(),
	}
	return resEv, nil
}

func taskExists(db *gorm.DB, task string) (bool, error) {
	var count int64
	res := db.Model(&dbTask{}).Where("name = ?", task).Count(&count)
	if res.Error != nil {
		return false, &DatabaseGeneralError{Err: res.Error}
	}

	return count != 0, nil
}

// AddEvent adds the event to the event log and updates affected tasks as necessary.
func (db *sqlDatabase) AddEvent(ev events.TaskEvent) (TaskEvent, error) {
	var resEv TaskEvent

	err := db.lockedRetrier.retry(func() error {
		return db.db.Transaction(func(tx *gorm.DB) error {
			var taskEv TaskEvent
			var res *gorm.DB

			switch ev := ev.(type) {
			case events.StartEvent:
				taskName := ev.Task

				task, err := newDBTask(taskName)
				if err != nil {
					return &InvalidParamsError{Err: err}
				}

				exists, err := taskExists(tx, taskName)
				if err != nil {
					return err
				}
				if !exists {
					return ErrNoSuchTask
				}

				dbEv, err := newDBStartEvent(ev)
				if err != nil {
					return &InvalidParamsError{Err: err}
				}

				taskEv, err = addEvent(tx, dbEv)
				if err != nil {
					return err
				}

				// Note that, because the event was successfully
				// added, there's no risk of conflicts or invalid
				// states resulting from this.
				res = tx.Model(&task).Where("name = ?", taskName).Update("state", tasks.RunningState)
			case events.StopEvent:
				taskName := ev.Task

				task, err := newDBTask(taskName)
				if err != nil {
					return &InvalidParamsError{Err: err}
				}

				exists, err := taskExists(tx, taskName)
				if err != nil {
					return err
				}
				if !exists {
					return ErrNoSuchTask
				}

				dbEv, err := newDBStopEvent(ev)
				if err != nil {
					return &InvalidParamsError{Err: err}
				}

				taskEv, err = addEvent(tx, dbEv)
				if err != nil {
					return err
				}

				res = tx.Model(&task).Where("name = ?", taskName).Update("state", tasks.StoppedState)
			case events.SwitchEvent:
				oldTaskName := ev.OldTask
				oldTask, err := newDBTask(oldTaskName)
				if err != nil {
					return &InvalidParamsError{Err: err}
				}
				exists, err := taskExists(tx, oldTaskName)
				if err != nil {
					return err
				}
				if !exists {
					return ErrNoSuchTask
				}

				newTaskName := ev.NewTask
				newTask, err := newDBTask(newTaskName)
				if err != nil {
					return &InvalidParamsError{Err: err}
				}
				exists, err = taskExists(tx, newTaskName)
				if err != nil {
					return err
				}
				if !exists {
					return ErrNoSuchTask
				}

				dbEv, err := newDBSwitchEvent(ev)
				if err != nil {
					return &InvalidParamsError{Err: err}
				}

				taskEv, err = addEvent(tx, dbEv)
				if err != nil {
					return err
				}

				res = tx.Model(&oldTask).Where("name = ?", oldTaskName).Update("state", tasks.StoppedState)
				if res.Error != nil {
					return &DatabaseGeneralError{Err: res.Error}
				}

				res = tx.Model(&newTask).Where("name = ?", newTaskName).Update("state", tasks.RunningState)
			default:
				panic(fmt.Sprintf("handling for %s events not implemented", ev.Name()))
			}

			if res.Error != nil {
				return &DatabaseGeneralError{Err: res.Error}
			}

			db.publish(taskEv)
			resEv = taskEv

			return nil
		})
	})

	return resEv, err
}

// NOTE: This must be called from within a transaction.
func getDBEvents(db *gorm.DB) ([]polymorphicEventModel, error) {
	ctx := context.Background()

	var evs []polymorphicEventModel
	startEvs, err := gorm.G[dbStartEvent](db).Preload("ProxyEvent", nil).Find(ctx)
	if err != nil {
		return nil, &DatabaseGeneralError{Err: err}
	}
	for _, ev := range startEvs {
		evs = append(evs, ev)
	}

	stopEvs, err := gorm.G[dbStopEvent](db).Preload("ProxyEvent", nil).Find(ctx)
	if err != nil {
		return nil, &DatabaseGeneralError{Err: err}
	}
	for _, ev := range stopEvs {
		evs = append(evs, ev)
	}

	switchEvs, err := gorm.G[dbSwitchEvent](db).Preload("ProxyEvent", nil).Find(ctx)
	if err != nil {
		return nil, &DatabaseGeneralError{Err: err}
	}
	for _, ev := range switchEvs {
		evs = append(evs, ev)
	}

	slices.SortFunc(evs, func(ev1, ev2 polymorphicEventModel) int {
		if ev1.absoluteID() > ev2.absoluteID() {
			return 1
		} else if ev1.absoluteID() < ev2.absoluteID() {
			return -1
		} else {
			return 0
		}
	})

	return evs, nil
}

// NOTE: This must be called from within a transaction.
func getBaseTaskEvents(db *gorm.DB) ([]events.TaskEvent, error) {
	dbEvs, err := getDBEvents(db)
	if err != nil {
		return nil, err
	}

	var evs []events.TaskEvent
	for _, dbEv := range dbEvs {
		evs = append(evs, dbEv.event())
	}

	return evs, nil
}

// NOTE: This must be called from within a transaction.
func getTaskEvents(db *gorm.DB) ([]TaskEvent, error) {
	dbEvs, err := getDBEvents(db)
	if err != nil {
		return nil, err
	}

	var evs []TaskEvent
	for _, dbEv := range dbEvs {
		ev := TaskEvent{
			ID:        dbEv.absoluteID(),
			Time:      dbEv.at(),
			TaskEvent: dbEv.event(),
		}
		evs = append(evs, ev)
	}

	return evs, nil
}

// GetEvents returns the full event log.
func (db *sqlDatabase) GetEvents() ([]TaskEvent, error) {
	var evs []TaskEvent

	err := db.lockedRetrier.retry(func() error {
		return db.db.Transaction(func(tx *gorm.DB) error {
			var err error
			evs, err = getTaskEvents(tx)
			return err
		})
	})

	return evs, err
}

// GetEvent returns the event corresponding to the specified id.
func (db *sqlDatabase) GetEvent(id int) (TaskEvent, error) {
	var resEv TaskEvent

	var dbEvs []polymorphicEventModel
	err := db.lockedRetrier.retry(func() error {
		return db.db.Transaction(func(tx *gorm.DB) error {
			var err error
			dbEvs, err = getDBEvents(tx)
			return err
		})
	})
	if err != nil {
		return resEv, err
	}

	for _, dbEv := range dbEvs {
		if dbEv.absoluteID() != id {
			continue
		}

		resEv = TaskEvent{
			ID:        dbEv.absoluteID(),
			Time:      dbEv.at(),
			TaskEvent: dbEv.event(),
		}
		return resEv, nil
	}

	return resEv, ErrNoSuchEvent
}

func getLatestProxyEvent(db *gorm.DB) (proxyEvent, error) {
	var resEv proxyEvent

	res := db.Model(&proxyEvent{}).Last(&resEv)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return resEv, ErrNoEvents
		}
		return resEv, &DatabaseGeneralError{Err: res.Error}
	}

	return resEv, nil
}

// NOTE: This must be called from within a transaction.
func getLatestDBEvent(db *gorm.DB) (polymorphicEventModel, error) {
	var resEv polymorphicEventModel

	latestProxyEv, err := getLatestProxyEvent(db)
	if err != nil {
		return resEv, err
	}

	var res *gorm.DB
	switch latestProxyEv.EventType {
	case startEventsTable:
		var startEv dbStartEvent
		res = db.Model(&dbStartEvent{}).Where("id = ?", latestProxyEv.EventID).Preload("ProxyEvent").First(&startEv)
		resEv = startEv
	case stopEventsTable:
		var stopEv dbStopEvent
		res = db.Model(&dbStopEvent{}).Where("id = ?", latestProxyEv.EventID).Preload("ProxyEvent").First(&stopEv)
		resEv = stopEv
	case switchEventsTable:
		var switchEv dbSwitchEvent
		res = db.Model(&switchEv).Where("id = ?", latestProxyEv.EventID).Preload("ProxyEvent").First(&switchEv)
		resEv = switchEv
	}

	if res.Error != nil {
		return resEv, &DatabaseGeneralError{Err: res.Error}
	}

	return resEv, nil
}

// NOTE: This must be called from within a transaction.
func getLatestTaskEvent(db *gorm.DB) (TaskEvent, error) {
	ev, err := getLatestDBEvent(db)
	if err != nil {
		return TaskEvent{}, err
	}

	return TaskEvent{
		ID:        ev.absoluteID(),
		Time:      ev.at(),
		TaskEvent: ev.event(),
	}, nil
}

// GetLatestEvent returns the latest event.
func (db *sqlDatabase) GetLatestEvent() (TaskEvent, error) {
	var ev TaskEvent
	err := db.lockedRetrier.retry(func() error {
		return db.db.Transaction(func(tx *gorm.DB) error {
			var err error
			ev, err = getLatestTaskEvent(tx)
			return err
		})
	})
	return ev, err
}

// UpdateEvent updates the event corresponding to the specified id to match newEvent,
// performing modifications to tasks as required.
//
// For all event types, task names may be updated.
//
// For start events, both the task name and the stop reason may be updated.
//
// Note that the time cannot be updated.
func (db *sqlDatabase) UpdateEvent(id int, newEvent events.TaskEvent) (TaskEvent, error) {
	var resEv TaskEvent

	err := db.lockedRetrier.retry(func() error {
		return db.db.Transaction(func(tx *gorm.DB) error {
			dbEvs, err := getDBEvents(tx)
			if err != nil {
				return err
			}

			if len(dbEvs) == 0 {
				return ErrNoEvents
			}

			latestEv := dbEvs[len(dbEvs)-1]
			if latestEv.absoluteID() != id {
				return ErrNotLatestEvent
			}

			var evs []events.TaskEvent
			for _, dbEv := range dbEvs {
				evs = append(evs, dbEv.event())
			}

			var res *gorm.DB
			var resDBEv polymorphicEventModel
			switch newEvent := newEvent.(type) {
			case events.StartEvent:
				startEv, isStartEv := latestEv.event().(events.StartEvent)
				if !isStartEv {
					return ErrMismatchingEventType
				}

				if newEvent.Task != startEv.Task {
					exists, err := taskExists(tx, newEvent.Task)
					if err != nil {
						return err
					}
					if !exists {
						return ErrNoSuchTask
					}
				}

				if err := validator.DefaultValidator.ValidateSequence(evs[:len(evs)-1], newEvent); err != nil {
					return fmt.Errorf("%w: %w", ErrInvalidEventSequence, err)
				}

				// Explicitly set a proxy event so that GORM doesn't
				// think that it's meant to refer to a new proxy
				// event and inserts another row into the proxy
				// table.
				dbEv, err := newDBStartEventWithProxy(newEvent, latestEv.proxyEvent())
				if err != nil {
					return &InvalidParamsError{Err: err}
				}

				// Note that, because we're using hooks to validate
				// events, we're unable to use the generics API.
				// Otherwise, GORM would pass zero-valued events
				// to the hooks, which would lead to improper errors
				// being returned, even though the event passed
				// in isn't being used directly in the operation.
				// Also note that event-specific updates need to
				// be done using the relative ID, whereas the ID
				// passed in to us as an argument is the absolute
				// ID.
				res = tx.Model(&dbEv).Clauses(clause.Returning{}).Where("id = ?", latestEv.relativeID()).Updates(map[string]any{"Task": newEvent.Task, "StopReason": newEvent.StopReason})
				if res.Error != nil {
					break
				}

				if newEvent.Task != startEv.Task {
					// The task is necessarily valid because it exists,
					// so no errors can be returned here.
					task, _ := newDBTask(newEvent.Task)
					res = tx.Model(&task).Where("name = ?", newEvent.Task).Update("state", tasks.RunningState)
					if res.Error != nil {
						break
					}

					res = tx.Model(&task).Where("name = ?", startEv.Task).Update("state", tasks.StoppedState)
					if res.Error != nil {
						break
					}
				}

				resDBEv = dbEv
			case events.StopEvent:
				return fmt.Errorf("%w: stop events can't be updated without becoming invalid", ErrNonUpdatableEvent)
			case events.SwitchEvent:
				switchEv, isSwitchEv := latestEv.event().(events.SwitchEvent)
				if !isSwitchEv {
					return ErrMismatchingEventType
				}

				if newEvent.OldTask != switchEv.OldTask {
					return fmt.Errorf("%w: the old task of a switch event can't be updated without invalidating the event", ErrIllegalUpdate)
				}

				if newEvent.NewTask != switchEv.NewTask {
					exists, err := taskExists(tx, newEvent.NewTask)
					if err != nil {
						return err
					}
					if !exists {
						return ErrNoSuchTask
					}
				}

				if err := validator.DefaultValidator.ValidateSequence(evs[:len(evs)-1], newEvent); err != nil {
					return fmt.Errorf("%w: %w", ErrInvalidEventSequence, err)
				}

				dbEv, err := newDBSwitchEventWithProxy(newEvent, latestEv.proxyEvent())
				if err != nil {
					return &InvalidParamsError{Err: err}
				}

				res = tx.Model(&dbEv).Clauses(clause.Returning{}).Where("id = ?", latestEv.relativeID()).Updates(map[string]any{"OldTask": newEvent.OldTask, "NewTask": newEvent.NewTask})
				if res.Error != nil {
					break
				}

				if newEvent.NewTask != switchEv.NewTask {
					// The task is necessarily valid because it exists,
					// so no errors can be returned here.
					task, _ := newDBTask(newEvent.NewTask)
					res = tx.Model(&task).Where("name = ?", newEvent.NewTask).Update("state", tasks.RunningState)
					if res.Error != nil {
						break
					}
	
					res = tx.Model(&task).Where("name = ?", switchEv.NewTask).Update("state", tasks.StoppedState)
					if res.Error != nil {
						break
					}
				}

				resDBEv = dbEv
			default:
				panic(fmt.Sprintf("handling for %s events not implemented", newEvent.Name()))
			}

			if res.Error != nil {
				return &DatabaseGeneralError{Err: res.Error}
			}

			resEv = TaskEvent{
				ID:        resDBEv.absoluteID(),
				Time:      resDBEv.at(),
				TaskEvent: resDBEv.event(),
			}
			db.publish(resEv)

			return nil
		})
	})
	return resEv, err
}

// UndoEvent undoes the event corresponding to the specified id.
func (db *sqlDatabase) UndoEvent(id int) error {
	err := db.lockedRetrier.retry(func() error {
		return db.db.Transaction(func(tx *gorm.DB) error {
			latestEv, err := getLatestDBEvent(tx)
			if err != nil {
				return err
			}

			if latestEv.absoluteID() != id {
				return ErrNotLatestEvent
			}

			if res := tx.Delete(&latestEv, latestEv.relativeID()); res.Error != nil {
				return &DatabaseGeneralError{Err: res.Error}
			}

			if res := tx.Delete(&proxyEvent{}, latestEv.absoluteID()); res.Error != nil {
				return &DatabaseGeneralError{Err: res.Error}
			}

			var res *gorm.DB
			switch dbEv := latestEv.(type) {
			case dbStartEvent:
				// No need to check the error; the task is necessarily
				// valid if the event was in the database in the
				// first place.
				task, _ := newDBTask(dbEv.Event.Task)
				res = tx.Model(&task).Where("name = ?", dbEv.Event.Task).Update("state", tasks.StoppedState)
			case dbStopEvent:
				task, _ := newDBTask(dbEv.Event.Task)
				res = tx.Model(&task).Where("name = ?", dbEv.Event.Task).Update("state", tasks.RunningState)
			case dbSwitchEvent:
				oldTask, _ := newDBTask(dbEv.Event.OldTask)
				res = tx.Model(&oldTask).Where("name = ?", dbEv.Event.OldTask).Update("state", tasks.RunningState)
				if res.Error != nil {
					return &DatabaseGeneralError{Err: res.Error}
				}

				newTask, _ := newDBTask(dbEv.Event.NewTask)
				res = tx.Model(&newTask).Where("name = ?", dbEv.Event.NewTask).Update("state", tasks.StoppedState)
			default:
				panic(fmt.Sprintf("handling for %s events not implemented", latestEv.event().Name()))
			}

			if res.Error != nil {
				return &DatabaseGeneralError{Err: res.Error}
			}

			db.publish(OperationEvent{
				Time:           time.Now(),
				OperationEvent: events.NewUndoEvent(),
			})

			return nil
		})
	})
	return err
}

func getTasks(db *gorm.DB) ([]tasks.Task, error) {
	ctx := context.Background()

	dbTasks, err := gorm.G[dbTask](db).Find(ctx)
	if err != nil {
		return nil, &DatabaseGeneralError{Err: err}
	}

	var tasks []tasks.Task
	for _, task := range dbTasks {
		tasks = append(tasks, task.task())
	}

	return tasks, nil
}

func getTaskByName(db *gorm.DB, name string) (tasks.Task, error) {
	var resTask tasks.Task

	ctx := context.Background()

	dbTask, err := gorm.G[dbTask](db).Where("name = ?", name).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resTask, ErrNoSuchTask
		}
		return resTask, &DatabaseGeneralError{Err: err}
	}

	return dbTask.task(), nil
}

// GetTask returns the task identified by the specified name.
func (db *sqlDatabase) GetTask(name string) (tasks.Task, error) {
	var resTask tasks.Task
	err := db.lockedRetrier.retry(func() error {
		return db.db.Transaction(func(tx *gorm.DB) error {
			var err error
			resTask, err = getTaskByName(tx, name)
			return err
		})
	})
	return resTask, err
}

// GetTasks returns the full list of tasks.
func (db *sqlDatabase) GetTasks() ([]tasks.Task, error) {
	var resTasks []tasks.Task
	err := db.lockedRetrier.retry(func() error {
		return db.db.Transaction(func(tx *gorm.DB) error {
			var err error
			resTasks, err = getTasks(tx)
			return err
		})
	})
	return resTasks, err
}

// RenameTask renames the specified task to newName.
func (db *sqlDatabase) RenameTask(task string, newName string) (tasks.Task, error) {
	resTask, err := newDBTask(task)
	if err != nil {
		return tasks.Task{}, &InvalidParamsError{Err: err}
	}

	if _, err := newDBTask(newName); err != nil {
		return tasks.Task{}, &InvalidParamsError{Err: err}
	}

	if task == newName {
		return tasks.Task{}, ErrRenameTaskSameName
	}

	err = db.lockedRetrier.retry(func() error {
		return db.db.Transaction(func(tx *gorm.DB) error {
			res := tx.Model(&resTask).Clauses(clause.Returning{}).Where("name = ?", task).Update("name", newName)
			if res.Error != nil {
				if errors.Is(res.Error, gorm.ErrDuplicatedKey) {
					return ErrTaskNameNotAvailable
				}
				return &DatabaseGeneralError{Err: res.Error}
			}

			if res.RowsAffected == 0 {
				return ErrNoSuchTask
			}

			ev, _ := events.NewRenameTaskEvent(task, newName)
			db.publish(OperationEvent{
				Time:           time.Now(),
				OperationEvent: ev,
			})

			return nil
		})
	})

	return resTask.task(), err
}

// DeleteTask deletes the task identified by the specified name.
func (db *sqlDatabase) DeleteTask(name string) error {
	_, err := newDBTask(name)
	if err != nil {
		return &InvalidParamsError{Err: err}
	}

	err = db.lockedRetrier.retry(func() error {
		return db.db.Transaction(func(tx *gorm.DB) error {
			var task dbTask
			res := tx.Model(&dbTask{}).Where("name = ?", name).First(&task)
			if res.Error != nil {
				if errors.Is(res.Error, gorm.ErrRecordNotFound) {
					return ErrNoSuchTask
				}
				return res.Error
			}

			if task.State == tasks.RunningState {
				return ErrDeleteTaskRunning
			}

			res = tx.Delete(&task)
			if res.Error != nil {
				return res.Error
			}

			ev, _ := events.NewDeleteTaskEvent(name)
			db.publish(OperationEvent{
				Time:           time.Now(),
				OperationEvent: ev,
			})

			return nil
		})
	})

	return err
}

type sqliteLockedRetrier struct{}

func newSQLiteLockedRetrier() *sqliteLockedRetrier {
	return &sqliteLockedRetrier{}
}

func (*sqliteLockedRetrier) retry(f func() error) error {
	for {
		err := f()
		var sqliteErr gosqlite.Error
		if errors.As(err, &sqliteErr) {
			if sqliteErr.Code == gosqlite.ErrBusy || sqliteErr.Code == gosqlite.ErrLocked {
				time.Sleep(time.Second * 2)
				continue
			}
		}
		return err
	}
}

// An SQLiteDatabase represents a SQLite database.
type SQLiteDatabase struct {
	*sqlDatabase
}

// NewSQLiteDatabase returns an SQLiteDatabase stored at the specified filename.
func NewSQLiteDatabase(filename string) (*SQLiteDatabase, error) {
	dsn := fmt.Sprintf("%s?_txlock=immediate&_foreign_keys=on", filename)
	baseDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Discard, TranslateError: true})
	if err != nil {
		return nil, &DatabaseGeneralError{Err: err}
	}

	db, err := newSQLDatabase(baseDB, newSQLiteLockedRetrier())
	if err != nil {
		return nil, &DatabaseGeneralError{Err: err}
	}

	return &SQLiteDatabase{
		sqlDatabase: db,
	}, nil
}

const (
	startEventsTable  = "start_events"
	stopEventsTable   = "stop_events"
	switchEventsTable = "switch_events"
)

type polymorphicEventModel interface {
	event() events.TaskEvent
	proxyEvent() proxyEvent
	relativeID() int
	absoluteID() int
	at() time.Time
}

type proxyEvent struct {
	ID        *int `gorm:"primaryKey;autoIncrement:false"`
	EventID   int
	EventType string
}

func (proxyEvent) TableName() string {
	return "events"
}

type BaseTaskEvent[T events.TaskEvent] struct {
	ID        int `gorm:"primaryKey"`
	CreatedAt time.Time
	DeletedAt gorm.DeletedAt
	Event     T `gorm:"embedded"`
	// In order for polymorphism to work without the base type having any extra
	// fields, this needs to be a pointer and set to a non-nil value when used.
	ProxyEvent *proxyEvent `gorm:"polymorphic:Event"`
}

func (ev BaseTaskEvent[T]) BeforeCreate(tx *gorm.DB) error {
	if err := ev.Event.Validate(); err != nil {
		return err
	}

	return nil
}

func (ev BaseTaskEvent[T]) BeforeSave(tx *gorm.DB) error {
	if err := ev.Event.Validate(); err != nil {
		return err
	}

	return nil
}

func (ev BaseTaskEvent[T]) event() events.TaskEvent {
	return ev.Event
}

func (ev BaseTaskEvent[T]) proxyEvent() proxyEvent {
	return *ev.ProxyEvent
}

func (ev BaseTaskEvent[T]) relativeID() int {
	return ev.ID
}

func (ev BaseTaskEvent[T]) absoluteID() int {
	return *ev.ProxyEvent.ID
}

func (ev BaseTaskEvent[T]) at() time.Time {
	return ev.CreatedAt
}

type dbStartEvent struct {
	BaseTaskEvent[events.StartEvent]
	TaskInstance dbTask `gorm:"foreignKey:Task;constraint:OnUpdate:CASCADE"` // effectively replaces the task field with a foreign key column.
}

func (dbStartEvent) TableName() string {
	return startEventsTable
}

func newDBStartEvent(ev events.StartEvent) (dbStartEvent, error) {
	if err := ev.Validate(); err != nil {
		return dbStartEvent{}, err
	}

	return dbStartEvent{
		BaseTaskEvent: BaseTaskEvent[events.StartEvent]{
			Event:      ev,
			ProxyEvent: &proxyEvent{},
		},
	}, nil
}

func newDBStartEventWithProxy(ev events.StartEvent, proxy proxyEvent) (dbStartEvent, error) {
	dbEv, err := newDBStartEvent(ev)
	if err != nil {
		return dbStartEvent{}, err
	}

	dbEv.ProxyEvent = &proxy
	return dbEv, nil
}

type dbStopEvent struct {
	BaseTaskEvent[events.StopEvent]
	TaskInstance dbTask `gorm:"foreignKey:Task;constraint:OnUpdate:CASCADE"`
}

func (dbStopEvent) TableName() string {
	return stopEventsTable
}

func newDBStopEvent(ev events.StopEvent) (dbStopEvent, error) {
	if err := ev.Validate(); err != nil {
		return dbStopEvent{}, err
	}

	return dbStopEvent{
		BaseTaskEvent: BaseTaskEvent[events.StopEvent]{
			Event:      ev,
			ProxyEvent: &proxyEvent{},
		},
	}, nil
}

func newDBStopEventsWithProxy(ev events.StopEvent, proxy proxyEvent) (dbStopEvent, error) {
	dbEv, err := newDBStopEvent(ev)
	if err != nil {
		return dbStopEvent{}, err
	}

	dbEv.ProxyEvent = &proxy
	return dbEv, nil
}

type dbSwitchEvent struct {
	BaseTaskEvent[events.SwitchEvent]
	OldTaskInstance dbTask `gorm:"foreignKey:OldTask;constraint:OnUpdate:CASCADE"`
	NewTaskInstance dbTask `gorm:"foreignKey:NewTask;constraint:OnUpdate:CASCADE"`
}

func (dbSwitchEvent) TableName() string {
	return switchEventsTable
}

func newDBSwitchEvent(ev events.SwitchEvent) (dbSwitchEvent, error) {
	if err := ev.Validate(); err != nil {
		return dbSwitchEvent{}, err
	}
	return dbSwitchEvent{
		BaseTaskEvent: BaseTaskEvent[events.SwitchEvent]{
			Event:      ev,
			ProxyEvent: &proxyEvent{},
		},
	}, nil
}

func newDBSwitchEventWithProxy(ev events.SwitchEvent, proxy proxyEvent) (dbSwitchEvent, error) {
	dbEv, err := newDBSwitchEvent(ev)
	if err != nil {
		return dbSwitchEvent{}, err
	}

	dbEv.ProxyEvent = &proxy
	return dbEv, nil
}

type dbTask struct {
	CreatedAt time.Time
	DeletedAt gorm.DeletedAt
	tasks.Task
	Name string `gorm:"primaryKey"` // guarantees that the name field is unique
}

func (dbTask) TableName() string {
	return "tasks"
}

func (task dbTask) BeforeCreate(*gorm.DB) error {
	task.Task.Name = task.Name
	if err := task.Validate(); err != nil {
		return err
	}

	return nil
}

func (task dbTask) BeforeSave(*gorm.DB) error {
	task.Task.Name = task.Name
	if err := task.Validate(); err != nil {
		return err
	}

	return nil
}

func newDBTask(name string) (dbTask, error) {
	task, err := tasks.New(name, tasks.StoppedState)
	if err != nil {
		return dbTask{}, err
	}

	return dbTask{
		Task: task,
		Name: task.Name,
	}, nil
}

func (task dbTask) task() tasks.Task {
	task.Task.Name = task.Name
	return task.Task
}
