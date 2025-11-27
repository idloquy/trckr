package main

import (
	"github.com/idloquy/trckr/cmd/trckr-http/database"
	"github.com/idloquy/trckr/pkg/api"
)

func dbTaskEvToAPITaskEv(ev database.TaskEvent) api.EventContainer[api.TaskEvent] {
	return api.EventContainer[api.TaskEvent]{
		EventContainerMeta: api.EventContainerMeta{
			At: ev.At(),
		},
		Event: api.TaskEvent{
			TaskEventMeta: api.TaskEventMeta{
				ID: ev.ID,
			},
			TaskEvent: ev.TaskEvent,
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
