package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"slices"
	"strconv"
	"time"

	"github.com/adrg/xdg"
	"github.com/alexflint/go-arg"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"github.com/idloquy/trckr/cmd/trckr-http/database"
	"github.com/idloquy/trckr/pkg/api"
	"github.com/idloquy/trckr/pkg/events"
	"github.com/idloquy/trckr/pkg/logging/logrus"
	"github.com/idloquy/trckr/pkg/tasks"
)

var (
	Version = "0.0.0"

	APIVersion = 1
)

type Handler struct {
	db       database.Database
	upgrader websocket.Upgrader
}

func (handler *Handler) GetInfo(c *gin.Context) {
	res := api.NewSuccessResponse(api.InfoResponse{
		Name:    api.ServerName,
		Version: APIVersion,
	})
	c.JSON(http.StatusOK, res)
}

func (handler *Handler) CreateTask(c *gin.Context) {
	var msg api.CreateTaskMsg
	if err := c.BindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, api.NewErrorResponse(err.Error()))
		return
	}

	log.WithFields(log.Fields{"name": msg.Name}).Debug("Creating task...")

	task, err := handler.db.CreateTask(msg.Name)
	if err != nil {
		var invalidParamsErr *database.InvalidParamsError
		if errors.As(err, &invalidParamsErr) {
			res := api.NewErrorResponse(fmt.Sprintf("invalid property values: %v", errors.Unwrap(err)))
			c.JSON(http.StatusBadRequest, res)
			return
		}

		switch {
		case errors.Is(err, database.ErrTaskAlreadyExists):
			c.JSON(http.StatusNoContent, nil)
		default:
			log.WithFields(log.Fields{"error": err}).Error("Error while creating task")
			c.JSON(http.StatusInternalServerError, api.NewErrorResponse("internal server error"))
		}
		return
	}

	c.Header("Location", fmt.Sprintf("/v1/tasks/%s", task.Name))
	c.JSON(http.StatusCreated, api.NewSuccessResponse(api.TaskResponse{Task: task}))
}

func (handler *Handler) ListTasks(c *gin.Context) {
	storedTasks, err := handler.db.GetTasks()
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error while querying tasks")
		c.JSON(http.StatusInternalServerError, api.NewErrorResponse("internal server error"))
		return
	}

	if state := c.Query("state"); state != "" {
		state, err := tasks.ParseState(state)
		if err != nil {
			c.JSON(http.StatusBadRequest, api.NewErrorResponse(err.Error()))
		}

		storedTasks = slices.DeleteFunc(storedTasks, func(task tasks.Task) bool {
			return task.State != state
		})
	}

	if len(storedTasks) == 0 {
		storedTasks = []tasks.Task{}
	}

	c.JSON(http.StatusOK, api.NewSuccessResponse(api.TaskListResponse{Tasks: storedTasks}))
}

func (handler *Handler) GetTask(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		panic("empty name param")
	}

	task, err := handler.db.GetTask(name)
	if err != nil {
		var invalidParamsErr *database.InvalidParamsError
		if errors.As(err, &invalidParamsErr) {
			res := api.NewErrorResponse(fmt.Sprintf("invalid property values: %v", errors.Unwrap(err)))
			c.JSON(http.StatusBadRequest, res)
			return
		}

		switch {
		case errors.Is(err, database.ErrNoSuchTask):
			c.JSON(http.StatusNotFound, api.NewErrorResponse(err.Error()))
		default:
			log.WithFields(log.Fields{"task": name, "error": err}).Error("Error while getting task")
			c.JSON(http.StatusInternalServerError, api.NewErrorResponse("internal server error"))
		}
		return
	}

	c.JSON(http.StatusOK, api.NewSuccessResponse(api.TaskResponse{Task: task}))
}

func (handler *Handler) UpdateTask(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		panic("empty name param")
	}

	var msg api.UpdateTaskMsg
	if err := c.BindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, api.NewErrorResponse(err.Error()))
		return
	}

	if msg.Name == nil {
		c.JSON(http.StatusNoContent, nil)
	}

	_, err := handler.db.RenameTask(name, *msg.Name)
	if err != nil {
		var invalidParamsErr *database.InvalidParamsError
		if errors.As(err, &invalidParamsErr) {
			res := api.NewErrorResponse(fmt.Sprintf("invalid property values: %v", errors.Unwrap(err)))
			c.JSON(http.StatusBadRequest, res)
			return
		}

		switch {
		case errors.Is(err, database.ErrRenameTaskSameName):
			c.JSON(http.StatusNoContent, nil)
		case errors.Is(err, database.ErrNoSuchTask):
			c.JSON(http.StatusNotFound, api.NewErrorResponse(err.Error()))
		case errors.Is(err, database.ErrTaskNameNotAvailable):
			c.JSON(http.StatusConflict, api.NewErrorResponse(err.Error()))
		default:
			log.WithFields(log.Fields{"name": name, "updated_name": *msg.Name, "error": err}).Error("Error while updating task")
			c.JSON(http.StatusInternalServerError, api.NewErrorResponse("internal server error"))
		}
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (handler *Handler) DeleteTask(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		panic("empty name param")
	}

	if err := handler.db.DeleteTask(name); err != nil {
		var invalidParamsErr *database.InvalidParamsError
		if errors.As(err, &invalidParamsErr) {
			res := api.NewErrorResponse(fmt.Sprintf("invalid property values: %v", errors.Unwrap(err)))
			c.JSON(http.StatusBadRequest, res)
			return
		}

		switch {
		case errors.Is(err, database.ErrNoSuchTask):
			c.JSON(http.StatusNotFound, api.NewErrorResponse(err.Error()))
		case errors.Is(err, database.ErrDeleteTaskRunning):
			c.JSON(http.StatusUnprocessableEntity, api.NewErrorResponse(err.Error()))
		default:
			log.WithFields(log.Fields{"name": name, "error": err}).Error("Error while deleting task")
			c.JSON(http.StatusInternalServerError, api.NewErrorResponse("internal server error"))
		}
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// NOTE: evs must be properly ordered.
func getTaskEventsWithinTimeRange(evs []database.TaskEvent, fromDate time.Time, toDate time.Time) []api.EventContainer[api.TaskEvent] {
	if len(evs) == 0 {
		return nil
	}

	if toDate.Unix() < fromDate.Unix() {
		panic("invalid arguments: toDate should be greater than or equal to fromDate")
	}

	firstIdx := slices.IndexFunc(evs, func(ev database.TaskEvent) bool {
		return ev.At().Compare(fromDate) >= 0
	})
	if firstIdx == -1 {
		return nil
	}

normalizeIdx:
	for {
		// We want a task-initiating event to be the first event, so return
		// an empty slice if none of the events after the original first
		// index is task-initiating.
		if firstIdx > len(evs)-1 {
			return nil
		}

		switch ev := evs[firstIdx].TaskEvent.(type) {
		case events.StartEvent:
			break normalizeIdx
		case events.SwitchEvent:
			break normalizeIdx
		case events.StopEvent:
			firstIdx++
		default:
			panic(fmt.Sprintf("handling for %s events not implemented", ev.Name()))
		}
	}

	if evs[firstIdx].At().After(toDate) {
		// There's nothing to return if the first task-initiating event
		// occurred later than toDate.
		return nil
	}

	lastIdx := -1
	for i, ev := range slices.Backward(evs) {
		if ev.At().Compare(toDate) <= 0 {
			lastIdx = i
			break
		}
	}

	// lastIdx can't be -1 at this point, because firstIdx has been validated
	// not to be -1, and lastIdx's minimum possible value is the same as firstIdx.
	// Note that, even if firstIdx has been increased, lastIdx's minimum possible
	// value will then be that of the updated firstIdx, given that higher indexes
	// have later dates.
	var convertedEvs []api.EventContainer[api.TaskEvent]
	for _, ev := range evs[firstIdx : lastIdx+1] {
		convertedEvs = append(convertedEvs, dbTaskEvToAPITaskEv(ev))
	}
	return convertedEvs
}

func getTaskEventsForDays(evs []database.TaskEvent, numDays int, dayDivider string) []api.EventContainer[api.TaskEvent] {
	if len(evs) == 0 || numDays == 0 {
		return nil
	}

	startIdx := -1

	if len(evs) > 1 {
		waitingForTaskInitiator := false
		numDaysThrough := 0
		for i, ev := range slices.Backward(evs) {
			var taskName string
			switch ev := ev.TaskEvent.(type) {
			case events.StartEvent:
				taskName = ev.Task
			case events.StopEvent:
				taskName = ev.Task
			case events.SwitchEvent:
				taskName = ev.NewTask
			default:
				panic(fmt.Sprintf("handling for %s events not implemented", ev.Name()))
			}

			if taskName == dayDivider {
				// Here's how this works:
				// If the current event is not a task initiator,
				// set this to true and treat it as the end of
				// the day. In the next iteration, where the task
				// initiator is found, we know not to treat it
				// as the end of yet another day because of this
				// variable. (Note that we're going over the slice
				// backwards, so non-task initiators come first.)
				// If, on the other hand, a task initiator is found
				// but we're not waiting for one, treat it as the
				// end of the day right away. Note that this may
				// occur both when a start event is the latest
				// event and generally when encountering switch
				// events.
				// Ignoring the above nuances, the operation is
				// simple: upon reaching the specified number of
				// days, set the index to the current index + 1,
				// so that the day-dividing event is not included
				// in the resulting list.
				var isTaskInitiator bool

				switch ev.TaskEvent.(type) {
				case events.StartEvent:
					isTaskInitiator = true
				case events.StopEvent:
					isTaskInitiator = false
				case events.SwitchEvent:
					isTaskInitiator = true
				default:
					panic(fmt.Sprintf("handling for %s events not implemented", ev.Name()))
				}

				// If we're waiting for a task initiator, the event
				// is part of the same day as the non-task initiator,
				// so don't treat it as the end of the day.
				if isTaskInitiator && waitingForTaskInitiator {
					waitingForTaskInitiator = false
					continue
				}

				// If this is the latest event and is a task initiator,
				// we still want today to be the day in which the
				// event happens, so we don't treat the event as
				// the end of the day.
				if isTaskInitiator && i == len(evs)-1 {
					continue
				}

				numDaysThrough++
				if numDaysThrough == numDays {
					startIdx = i + 1
					break
				}

				// If this is a non-task initiator, set waitingForTaskInitiator
				// so the corresponding task initiator isn't treated
				// as yet another day divider.
				if !isTaskInitiator {
					waitingForTaskInitiator = true
				}
			}
		}
	}

	// NOTE: This covers both the case where the loop doesn't set it and the
	// case where len(evs) == 1
	if startIdx == -1 {
		startIdx = 0
	}

	var convertedEvs []api.EventContainer[api.TaskEvent]
	for _, ev := range evs[startIdx:] {
		convertedEvs = append(convertedEvs, dbTaskEvToAPITaskEv(ev))
	}
	return convertedEvs
}

func (handler *Handler) AddEvent(c *gin.Context) {
	var ev api.RequestEventContainer[api.RequestTaskEvent]

	if err := c.BindJSON(&ev); err != nil {
		c.JSON(http.StatusBadRequest, api.NewErrorResponse(err.Error()))
		return
	}

	evFields := logrus.MapEvent(ev.Event.TaskEvent)
	log.WithFields(evFields).Debug("Adding event...")

	dbEv, err := handler.db.AddEvent(ev.Event.TaskEvent)
	if err != nil {
		var invalidParamsErr *database.InvalidParamsError
		if errors.As(err, &invalidParamsErr) {
			res := api.NewErrorResponse(fmt.Sprintf("invalid property values: %v", errors.Unwrap(err)))
			c.JSON(http.StatusBadRequest, res)
			return
		}

		switch {
		case errors.Is(err, database.ErrNoSuchTask):
			c.JSON(http.StatusUnprocessableEntity, api.NewErrorResponse(err.Error()))
		case errors.Is(err, database.ErrInvalidEventSequence):
			c.JSON(http.StatusConflict, api.NewErrorResponse(err.Error()))
		default:
			log.WithFields(evFields).WithFields(log.Fields{"error": err}).Error("Error while adding event")
			c.JSON(http.StatusInternalServerError, api.NewErrorResponse("internal server error"))
		}

		return
	}

	apiEv := api.EventContainer[api.TaskEvent]{
		EventContainerMeta: api.EventContainerMeta{At: dbEv.At()},
		Event: api.TaskEvent{
			TaskEventMeta: api.TaskEventMeta{ID: dbEv.ID},
			TaskEvent:     dbEv.TaskEvent,
		},
	}

	c.Header("Location", fmt.Sprintf("/v1/events/%d", dbEv.ID))
	c.JSON(http.StatusCreated, api.NewSuccessResponse(api.TaskEventResponse{TaskEvent: apiEv}))
}

func (handler *Handler) ListEvents(c *gin.Context) {
	fromDate := c.Query("from_date")
	toDate := c.Query("to_date")
	if (fromDate == "") != (toDate == "") {
		c.JSON(http.StatusBadRequest, api.NewErrorResponse("both from_date and to_date must be specified or neither of them"))
		return
	}

	days := c.Query("days")
	dayDivider := c.Query("day_divider")
	if (days == "") != (dayDivider == "") {
		c.JSON(http.StatusBadRequest, api.NewErrorResponse("both days and day_divider must be specified or neither of them"))
		return
	}

	if fromDate != "" && days != "" {
		c.JSON(http.StatusBadRequest, api.NewErrorResponse("both time-based and day-based filtering specified"))
		return
	}

	var fromTime time.Time
	var toTime time.Time
	if fromDate != "" {
		fromTimestamp, err := strconv.ParseInt(fromDate, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, api.NewErrorResponse("from_date should be a UNIX timestamp"))
			return
		}

		toTimestamp, err := strconv.ParseInt(toDate, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, api.NewErrorResponse("to_date should be a UNIX timestamp"))
			return
		}

		if fromTimestamp > toTimestamp {
			c.JSON(http.StatusBadRequest, api.NewErrorResponse("from_date should be earlier than or equal to to_date"))
			return
		}

		fromTime = time.Unix(fromTimestamp, 0)
		toTime = time.Unix(toTimestamp, 0)
	}

	var nDays int
	if days != "" {
		var err error
		nDays, err = strconv.Atoi(days)
		if err != nil {
			c.JSON(http.StatusBadRequest, api.NewErrorResponse("days should be an integer"))
			return
		}
	}

	storedEvs, err := handler.db.GetEvents()
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error while listing events")
		c.JSON(http.StatusInternalServerError, api.NewErrorResponse("internal server error"))
	}

	var filteredEvs []api.EventContainer[api.TaskEvent]

	if fromDate != "" {
		log.Debug("Filtering by time")
		filteredEvs = getTaskEventsWithinTimeRange(storedEvs, fromTime, toTime)
	} else if days != "" {
		log.Debug("Filtering by number of days")
		filteredEvs = getTaskEventsForDays(storedEvs, nDays, dayDivider)
	} else {
		log.Debug("No params specified for filtering")
		for _, ev := range storedEvs {
			filteredEvs = append(filteredEvs, dbTaskEvToAPITaskEv(ev))
		}
	}

	if len(filteredEvs) == 0 {
		filteredEvs = []api.EventContainer[api.TaskEvent]{}
	}

	c.JSON(http.StatusOK, api.NewSuccessResponse(api.TaskEventListResponse{TaskEvents: filteredEvs}))
}

func (handler *Handler) GetEvent(c *gin.Context) {
	eventID := c.Param("id")
	if eventID == "" {
		panic("GetEvent called with empty eventID param")
	}

	var dbEv database.TaskEvent
	var err error
	if eventID == "latest" {
		dbEv, err = handler.db.GetLatestEvent()
		if err != nil {
			if errors.Is(err, database.ErrNoEvents) {
				c.JSON(http.StatusNotFound, api.NewErrorResponse(err.Error()))
				return
			}
		}
	} else {
		var id int
		id, err = strconv.Atoi(eventID)
		if err != nil {
			c.JSON(http.StatusBadRequest, api.NewErrorResponse("invalid event id"))
			return
		}
		dbEv, err = handler.db.GetEvent(id)
		if err != nil {
			if errors.Is(err, database.ErrNoSuchEvent) {
				c.JSON(http.StatusNotFound, api.NewErrorResponse(err.Error()))
				return
			}
		}
	}

	if err != nil {
		log.WithFields(log.Fields{"id": eventID, "error": err, "request_path": c.Request.URL.Path}).Error("Error while getting event")
		c.JSON(http.StatusInternalServerError, api.NewErrorResponse("internal server error"))
		return
	}

	ev := api.EventContainer[api.TaskEvent]{
		EventContainerMeta: api.EventContainerMeta{At: dbEv.At()},
		Event: api.TaskEvent{
			TaskEventMeta: api.TaskEventMeta{ID: dbEv.ID},
			TaskEvent:     dbEv.TaskEvent,
		},
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(api.TaskEventResponse{TaskEvent: ev}))
}

func (handler *Handler) UpdateEvent(c *gin.Context) {
	eventIDParam := c.Param("id")
	if eventIDParam == "" {
		panic("UpdateEvent called with empty eventID param")
	}

	eventID, err := strconv.Atoi(eventIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.NewErrorResponse("invalid event id"))
		return
	}

	var newEv api.TaskEvent
	if err := c.BindJSON(&newEv); err != nil {
		c.JSON(http.StatusBadRequest, api.NewErrorResponse(err.Error()))
		return
	}

	_, err = handler.db.UpdateEvent(eventID, newEv.TaskEvent)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrNoEvents):
			c.JSON(http.StatusNotFound, api.NewErrorResponse(err.Error()))
		case errors.Is(err, database.ErrNotLatestEvent):
			c.JSON(http.StatusUnprocessableEntity, api.NewErrorResponse(err.Error()))
		case errors.Is(err, database.ErrNonUpdatableEvent):
			c.JSON(http.StatusUnprocessableEntity, api.NewErrorResponse(err.Error()))
		case errors.Is(err, database.ErrIllegalUpdate):
			c.JSON(http.StatusUnprocessableEntity, api.NewErrorResponse(err.Error()))
		case errors.Is(err, database.ErrMismatchingEventType):
			c.JSON(http.StatusUnprocessableEntity, api.NewErrorResponse(err.Error()))
		case errors.Is(err, database.ErrInvalidEventSequence):
			c.JSON(http.StatusConflict, api.NewErrorResponse(err.Error()))
		case errors.Is(err, database.ErrNoSuchTask):
			c.JSON(http.StatusUnprocessableEntity, api.NewErrorResponse(err.Error()))
		default:
			log.WithFields(logrus.MapEvent(newEv.TaskEvent)).WithFields(log.Fields{"error": err}).Error("Error while updating event")
			c.JSON(http.StatusInternalServerError, api.NewErrorResponse("internal server error"))
		}
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (handler *Handler) DeleteEvent(c *gin.Context) {
	eventIDParam := c.Param("id")
	if eventIDParam == "" {
		panic("empty id param")
	}

	eventID, err := strconv.Atoi(eventIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.NewErrorResponse(err.Error()))
		return
	}

	if err := handler.db.UndoEvent(eventID); err != nil {
		switch {
		case errors.Is(err, database.ErrNoEvents):
			c.JSON(http.StatusNotFound, api.NewErrorResponse(err.Error()))
		case errors.Is(err, database.ErrNotLatestEvent):
			c.JSON(http.StatusUnprocessableEntity, api.NewErrorResponse(err.Error()))
		default:
			log.WithFields(log.Fields{"id": eventID, "error": err}).Error("Error while undoing event")
			c.JSON(http.StatusInternalServerError, api.NewErrorResponse("internal server error"))
		}
		return
	}
	c.JSON(http.StatusNoContent, nil)
	return
}

func (handler *Handler) EventStream(c *gin.Context) {
	conn, err := handler.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	logger := log.WithFields(log.Fields{"remote_addr": conn.RemoteAddr().String()})
	logger.Debug("New websocket connection")

	ch := handler.db.Subscribe()
	defer handler.db.Unsubscribe(ch)

	for dbEv := range ch {
		var fields log.Fields
		var container any
		switch dbEv := dbEv.(type) {
		case database.TaskEvent:
			container = dbTaskEvToAPITaskEv(dbEv)
			fields = logrus.MapEvent(dbEv.TaskEvent)
		case database.OperationEvent:
			container = dbOperationEvToAPIOperationEv(dbEv)
			fields = logrus.MapEvent(dbEv.OperationEvent)
		default:
			panic(fmt.Sprintf("handling for database event type %s not implemented", reflect.TypeOf(dbEv).Name()))
		}
		logger.WithFields(fields).Debug("Transmitting event...")

		if err := conn.WriteJSON(&container); err != nil {
			// NOTE: This is just a debug log because errors reported
			// here include those related to normal connection closures.
			logger.WithFields(fields).WithFields(log.Fields{"error": err}).Debug("Error while transmitting event")
			return
		}
		logger.Debug("Successfully transmitted event")

		logger.Debug("Waiting for new events...")
	}
}

func (handler *Handler) NoRoute() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, api.NewErrorResponse("not found"))
	}
}

func (handler *Handler) NoMethod() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, api.NewErrorResponse("method not allowed"))
	}
}

func (handler *Handler) BadRequest(c *gin.Context) {
	c.JSON(http.StatusBadRequest, api.NewErrorResponse("bad request"))
}

func (handler *Handler) MediaTypeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		supportedFormat := "application/json"
		negotiatedFormat := c.NegotiateFormat(supportedFormat)
		if negotiatedFormat != supportedFormat {
			res := api.NewErrorResponse(fmt.Sprintf("only '%s' media type supported", supportedFormat))
			c.JSON(http.StatusNotAcceptable, res)
			c.Abort()
			return
		}

		acceptedFormat := "application/json"
		if c.Request.Method == http.MethodPatch {
			acceptedFormat = "application/merge-patch+json"
		}
		if c.ContentType() != acceptedFormat && slices.Contains([]string{http.MethodPost, http.MethodPut, http.MethodPatch}, c.Request.Method) {
			res := api.NewErrorResponse(fmt.Sprintf("only '%s' content type supported", acceptedFormat))
			c.JSON(http.StatusUnsupportedMediaType, res)
			c.Abort()
			return
		}

		c.Next()
	}
}

func (handler *Handler) Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Infof("%s %s", c.Request.Method, c.Request.URL)

		c.Next()
	}
}

type Args struct {
	Verbose    bool   `arg:"-v,--verbose" help:"output debug logs"`
	ListenAddr string `arg:"-l,--listen-addr" default:":8080" help:"custom listen address"`
	Database   string `arg:"-d,--database-file" help:"custom path to the database file"`
	Version bool `arg:"--version" help:"display the version"`
}

func main() {
	var args Args
	parserCfg := arg.Config{Program: "trckr-http"}
	parser, err := arg.NewParser(parserCfg, &args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(os.Args) < 1 {
		parser.WriteUsage(os.Stderr)
		os.Exit(2)
	}
	parser.MustParse(os.Args[1:])

	if args.Version {
		fmt.Printf("trckr %s\n", Version)
		return
	}

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.DateTime,
	})
	if args.Verbose {
		log.SetLevel(log.DebugLevel)
	}

	var dbPath string
	if args.Database != "" {
		dbPath = args.Database
	} else {
		var err error
		dbPath, err = xdg.DataFile("trckr/http.sqlite")
		if err != nil {
			log.Fatal("error opening database: ", err)
		}
	}

	db, err := database.NewSQLiteDatabase(dbPath)
	if err != nil {
		log.Fatal("error opening database: ", err)
	}

	handler := &Handler{
		db:       db,
		upgrader: websocket.Upgrader{},
	}

	router := gin.New()
	router.Use(handler.Logger())

	router.HandleMethodNotAllowed = true
	router.NoMethod(handler.NoMethod())
	router.NoRoute(handler.NoRoute())

	router.GET("/info", handler.GetInfo)

	v1 := router.Group("/v1")
	v1.Use(handler.MediaTypeHandler())

	v1.POST("/tasks", handler.CreateTask)
	v1.GET("/tasks", handler.ListTasks)

	v1.GET("/tasks/:name", handler.GetTask)
	v1.PATCH("/tasks/:name", handler.UpdateTask)
	v1.DELETE("/tasks/:name", handler.DeleteTask)

	v1.POST("/events", handler.AddEvent)
	v1.GET("/events", handler.ListEvents)

	v1.GET("/events/:id", handler.GetEvent)
	v1.PATCH("/events/:id", handler.UpdateEvent)
	v1.DELETE("/events/:id", handler.DeleteEvent)

	router.GET("/ws/events", handler.EventStream)

	log.Info("Listening on ", args.ListenAddr)
	router.Run(args.ListenAddr)
}
