// Package logrus provides utilities for integration with logrus.
package logrus

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/idloquy/trckr/pkg/events"
	"github.com/idloquy/trckr/pkg/tasks"
)

// MapEvent returns a representation of ev that can be used with logrus directly.
func MapEvent(ev events.Event) log.Fields {
	switch ev := ev.(type) {
	case events.TaskEvent:
		return mapTaskEvent(ev)
	case events.OperationEvent:
		return mapOperationEvent(ev)
	default:
		panic(fmt.Sprintf("handling for %s events not implemented", ev.Kind()))
	}
}

func mapTaskEvent(ev events.TaskEvent) log.Fields {
	switch ev := ev.(type) {
	case events.StartEvent:
		return log.Fields{
			"type":        ev.Name(),
			"task":        ev.Task,
			"stop_reason": ev.StopReason,
		}
	case events.StopEvent:
		return log.Fields{
			"type": ev.Name(),
			"task": ev.Task,
		}
	case events.SwitchEvent:
		return log.Fields{
			"type":     ev.Name(),
			"old_task": ev.OldTask,
			"new_task": ev.NewTask,
		}
	default:
		panic(fmt.Sprintf("handling for task %s events not implemented", ev.Name()))
	}
}

func mapOperationEvent(ev events.Event) log.Fields {
	switch ev := ev.(type) {
	case events.CreateTaskEvent:
		return log.Fields{
			"type": ev.Name(),
			"task": ev.Task,
		}
	case events.RenameTaskEvent:
		return log.Fields{
			"type":     ev.Name(),
			"task":     ev.Task,
			"new_name": ev.NewName,
		}
	case events.DeleteTaskEvent:
		return log.Fields{
			"type": ev.Name(),
			"task": ev.Task,
		}
	case events.UndoEvent:
		return log.Fields{
			"type": ev.Name(),
		}
	default:
		panic(fmt.Sprintf("handling for %s events not implemented", ev.Name()))
	}
}

// MapTask returns a representation of the specified task that can be used with
// logrus directly.
func MapTask(task tasks.Task) log.Fields {
	return log.Fields{
		"name":  task.Name,
		"state": task.State,
	}
}
