package main

import (
	"fmt"

	"github.com/idloquy/trckr/cmd/trckr-http/database"
	"github.com/idloquy/trckr/pkg/api"
	"github.com/idloquy/trckr/pkg/events"
)

func dbTaskEvToAPITaskEv(ev database.TaskEvent) api.EventContainer[api.TaskEvent] {
	taskEv := ev.TaskEvent
	switch ev := taskEv.(type) {
	case events.StartEvent:
		if ev.StopTags == nil {
			ev.StopTags = []string{}
		}
		taskEv = ev
	case events.StopEvent:
	case events.SwitchEvent:
	default:
		panic(fmt.Sprintf("handling for %s events not implemented", ev.Name()))
	}

	return api.EventContainer[api.TaskEvent]{
		EventContainerMeta: api.EventContainerMeta{
			At: ev.At(),
		},
		Event: api.TaskEvent{
			TaskEventMeta: api.TaskEventMeta{
				ID: ev.ID,
			},
			TaskEvent: taskEv,
		},
	}
}

func dbOperationEvToAPIOperationEv(ev database.OperationEvent) api.EventContainer[api.OperationEvent] {
	return api.EventContainer[api.OperationEvent]{
		EventContainerMeta: api.EventContainerMeta{
			At: ev.At(),
		},
		Event: api.OperationEvent{
			OperationEvent: ev.OperationEvent,
		},
	}
}
