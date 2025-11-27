package validator

import (
	"errors"
	"testing"

	"github.com/idloquy/trckr/pkg/events"
)

func TestStartStop(t *testing.T) {
	v := DefaultValidator

	evs := []events.TaskEvent{
		events.StartEvent{
			Task: "foo",
		},
	}
	ev := events.StopEvent{
		Task: "foo",
	}

	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to stop a running task: %v", err)
	}
}

func TestPartialStartStop(t *testing.T) {
	v := ForPartialSequence()

	evs := []events.TaskEvent{
		events.StartEvent{
			Task: "foo",
		},
	}
	ev := events.StopEvent{
		Task: "foo",
	}

	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to stop a running task with a partial sequence validator: %v", err)
	}
}

func TestStartSwitch(t *testing.T) {
	v := DefaultValidator

	evs := []events.TaskEvent{
		events.StartEvent{
			Task: "foo",
		},
	}
	ev := events.SwitchEvent{
		OldTask: "foo",
		NewTask: "bar",
	}

	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to switch from a running task: %v", err)
	}
}

func TestPartialStartSwitch(t *testing.T) {
	v := ForPartialSequence()

	evs := []events.TaskEvent{
		events.StartEvent{
			Task: "foo",
		},
	}
	ev := events.SwitchEvent{
		OldTask: "foo",
		NewTask: "bar",
	}

	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to switch from a running task with a partial sequence validator: %v", err)
	}
}

func TestStopStart(t *testing.T) {
	v := DefaultValidator

	evs := []events.TaskEvent{
		events.StopEvent{
			Task: "foo",
		},
	}
	ev := events.StartEvent{
		Task: "foo",
	}

	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to start a stopped task: %v", err)
	}

	ev.StopReason = "lorem ipsum"
	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to start a stopped task while specifying a stop reason: %v", err)
	}
}

func TestPartialStopStart(t *testing.T) {
	v := ForPartialSequence()

	evs := []events.TaskEvent{
		events.StopEvent{
			Task: "foo",
		},
	}
	ev := events.StartEvent{
		Task: "foo",
	}

	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to start a stopped task with a partial sequence validator: %v", err)
	}

	ev.StopReason = "lorem ipsum"
	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to start a stopped task while specifying a stop reason with a partial sequence validator: %v", err)
	}
}

func TestStopStartDiffTask(t *testing.T) {
	v := DefaultValidator

	evs := []events.TaskEvent{
		events.StopEvent{
			Task: "foo",
		},
	}
	ev := events.StartEvent{
		Task: "bar",
	}

	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to start a task: %v", err)
	}

	ev.StopReason = "lorem ipsum"
	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to start a task while specifying a stop reason: %v", err)
	}
}

func TestPartialStopStartDiffTask(t *testing.T) {
	v := ForPartialSequence()

	evs := []events.TaskEvent{
		events.StopEvent{
			Task: "foo",
		},
	}
	ev := events.StartEvent{
		Task: "bar",
	}

	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to start a task: %v", err)
	}

	ev.StopReason = "lorem ipsum"
	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to start a task while specifying a stop reason: %v", err)
	}
}

func TestSwitchSwitch(t *testing.T) {
	v := DefaultValidator

	evs := []events.TaskEvent{
		events.SwitchEvent{
			OldTask: "bar",
			NewTask: "foo",
		},
	}
	ev := events.SwitchEvent{
		OldTask: "foo",
		NewTask: "bar",
	}

	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to switch tasks: %v", err)
	}
}

func TestPartialSwitchSwitch(t *testing.T) {
	v := ForPartialSequence()

	evs := []events.TaskEvent{
		events.SwitchEvent{
			OldTask: "bar",
			NewTask: "foo",
		},
	}
	ev := events.SwitchEvent{
		OldTask: "foo",
		NewTask: "bar",
	}

	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to start a task: %v", err)
	}
}

func TestSwitchStop(t *testing.T) {
	v := DefaultValidator

	evs := []events.TaskEvent{
		events.SwitchEvent{
			OldTask: "bar",
			NewTask: "foo",
		},
	}
	ev := events.StopEvent{
		Task: "foo",
	}

	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to stop a task: %v", err)
	}
}

func TestPartialSwitchStop(t *testing.T) {
	v := ForPartialSequence()

	evs := []events.TaskEvent{
		events.SwitchEvent{
			OldTask: "bar",
			NewTask: "foo",
		},
	}
	ev := events.StopEvent{
		Task: "foo",
	}

	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to stop a task with a partial sequence validator: %v", err)
	}
}

func TestNoStopSwitch(t *testing.T) {
	v := DefaultValidator

	evs := []events.TaskEvent{
		events.StopEvent{
			Task: "bar",
		},
	}
	ev := events.SwitchEvent{
		OldTask: "bar",
		NewTask: "foo",
	}

	if err := v.ValidateSequence(evs, ev); !errors.Is(err, ErrUnexpectedEvent) {
		if err != nil {
			t.Fatalf("unexpected error when trying to switch after stopping: %v", err)
		}
		t.Fatalf("invalid event sequence allowed: switch event allowed after stop event")
	}
}

func TestPartialNoStopSwitch(t *testing.T) {
	v := ForPartialSequence()

	evs := []events.TaskEvent{
		events.StopEvent{
			Task: "bar",
		},
	}
	ev := events.SwitchEvent{
		OldTask: "bar",
		NewTask: "foo",
	}

	if err := v.ValidateSequence(evs, ev); !errors.Is(err, ErrUnexpectedEvent) {
		if err != nil {
			t.Fatalf("unexpected error when trying to switch after stopping with a partial sequence validator: %v", err)
		}
		t.Fatalf("invalid event sequence allowed: switch event allowed after stop event with a partial sequence validator")
	}
}

func TestNoSwitchStart(t *testing.T) {
	v := DefaultValidator

	evs := []events.TaskEvent{
		events.SwitchEvent{
			OldTask: "foo",
			NewTask: "bar",
		},
	}
	ev := events.StartEvent{
		Task: "foo",
	}

	if err := v.ValidateSequence(evs, ev); !errors.Is(err, ErrUnexpectedEvent) {
		if err != nil {
			t.Fatalf("unexpected error when trying to start a task after switching: %v", err)
		}
		t.Fatalf("invalid event sequence allowed: starting a task allowed after switching")
	}
}

func TestPartialNoSwitchStart(t *testing.T) {
	v := ForPartialSequence()

	evs := []events.TaskEvent{
		events.SwitchEvent{
			OldTask: "foo",
			NewTask: "bar",
		},
	}
	ev := events.StartEvent{
		Task: "foo",
	}

	if err := v.ValidateSequence(evs, ev); !errors.Is(err, ErrUnexpectedEvent) {
		if err != nil {
			t.Fatalf("unexpected error when trying to start a task after switching with a partial sequence validator: %v", err)
		}
		t.Fatalf("invalid event sequence allowed: starting a task allowed after switching with a partial sequence validator")
	}
}

func TestNoStopAsFirst(t *testing.T) {
	v := DefaultValidator

	evs := []events.TaskEvent{}
	ev := events.StopEvent{
		Task: "foo",
	}

	if err := v.ValidateSequence(evs, ev); !errors.Is(err, ErrUnexpectedEvent) {
		if err != nil {
			t.Fatalf("unexpected error when trying to use stop event as the first event: %v", err)
		}
		t.Fatalf("invalid event sequence allowed: stop event allowed as the first event")
	}
}

func TestPartialStopAsFirst(t *testing.T) {
	v := ForPartialSequence()

	evs := []events.TaskEvent{}
	ev := events.StopEvent{
		Task: "foo",
	}

	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to use stop event as the first event with a partial sequence validator: %v", err)
	}
}

func TestNoSwitchAsFirst(t *testing.T) {
	v := DefaultValidator

	evs := []events.TaskEvent{}
	ev := events.SwitchEvent{
		OldTask: "foo",
		NewTask: "bar",
	}

	if err := v.ValidateSequence(evs, ev); !errors.Is(err, ErrUnexpectedEvent) {
		if err != nil {
			t.Fatalf("unexpected error when trying to use switch event as the first event: %v", err)
		}
		t.Fatalf("invalid event sequence allowed: switch event allowed as the first event")
	}
}

func TestPartialSwitchEventAsFirst(t *testing.T) {
	v := ForPartialSequence()

	evs := []events.TaskEvent{}
	ev := events.SwitchEvent{
		OldTask: "foo",
		NewTask: "bar",
	}

	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to use switch event as the first event with a partial sequence validator: %v", err)
	}
}

func TestStartAsFirst(t *testing.T) {
	v := DefaultValidator

	evs := []events.TaskEvent{}
	ev := events.StartEvent{
		Task: "foo",
	}

	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to use start event as the first event: %v", err)
	}
}

func TestPartialStartAsFirst(t *testing.T) {
	v := ForPartialSequence()

	evs := []events.TaskEvent{}
	ev := events.StartEvent{
		Task: "foo",
	}

	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to use start event as the first event: %v", err)
	}
}

func TestNoStopReasonInFirst(t *testing.T) {
	v := DefaultValidator

	evs := []events.TaskEvent{}
	ev := events.StartEvent{
		Task:       "foo",
		StopReason: "lorem ipsum",
	}

	if err := v.ValidateSequence(evs, ev); !errors.Is(err, ErrUnexpectedStopReason) {
		if err != nil {
			t.Fatalf("unexpected error when trying to use start event with stop reason as the first event: %v", err)
		}
		t.Fatalf("invalid event sequence allowed: start event with stop reason allowed as the first event")
	}
}

func TestPartialStopReasonInFirst(t *testing.T) {
	v := ForPartialSequence()

	evs := []events.TaskEvent{}
	ev := events.StartEvent{
		Task:       "foo",
		StopReason: "lorem ipsum",
	}

	if err := v.ValidateSequence(evs, ev); err != nil {
		t.Fatalf("unexpected error when trying to use start event with stop reason as the first event with a partial sequence validator: %v", err)
	}
}

func TestNoDoubleStart(t *testing.T) {
	v := DefaultValidator

	evs := []events.TaskEvent{
		events.StartEvent{
			Task: "foo",
		},
	}
	ev := events.StartEvent{
		Task: "foo",
	}

	if err := v.ValidateSequence(evs, ev); !errors.Is(err, ErrUnexpectedEvent) {
		if err != nil {
			t.Fatalf("unexpected error when trying to use two start events consecutively: %v", err)
		}
		t.Fatalf("invalid event sequence allowed: two consecutive start events allowed")
	}
}

func TestPartialNoDoubleStart(t *testing.T) {
	v := ForPartialSequence()

	evs := []events.TaskEvent{
		events.StartEvent{
			Task: "foo",
		},
	}
	ev := events.StartEvent{
		Task: "foo",
	}

	if err := v.ValidateSequence(evs, ev); !errors.Is(err, ErrUnexpectedEvent) {
		if err != nil {
			t.Fatalf("unexpected error when trying to use two start events consecutively: %v", err)
		}
		t.Fatalf("invalid event sequence allowed: two consecutive start events allowed")
	}
}

func TestNoDoubleStartDiffTask(t *testing.T) {
	v := DefaultValidator

	evs := []events.TaskEvent{
		events.StartEvent{
			Task: "foo",
		},
	}
	ev := events.StartEvent{
		Task: "bar",
	}

	if err := v.ValidateSequence(evs, ev); !errors.Is(err, ErrUnexpectedEvent) {
		if err != nil {
			t.Fatalf("unexpected error when trying to use two start events consecutively: %v", err)
		}
		t.Fatalf("invalid event sequence allowed: two consecutive start events allowed")
	}
}

func TestPartialNoDoubleStartDiffTask(t *testing.T) {
	v := ForPartialSequence()

	evs := []events.TaskEvent{
		events.StartEvent{
			Task: "foo",
		},
	}
	ev := events.StartEvent{
		Task: "bar",
	}

	if err := v.ValidateSequence(evs, ev); !errors.Is(err, ErrUnexpectedEvent) {
		if err != nil {
			t.Fatalf("unexpected error when trying to use two start events consecutively: %v", err)
		}
		t.Fatalf("invalid event sequence allowed: two consecutive start events allowed")
	}
}

func TestNoNonrunningStop(t *testing.T) {
	v := DefaultValidator

	evs := []events.TaskEvent{
		events.StartEvent{
			Task: "foo",
		},
	}
	ev := events.StopEvent{
		Task: "bar",
	}

	if err := v.ValidateSequence(evs, ev); !errors.Is(err, ErrTaskNotRunning) {
		if err != nil {
			t.Fatalf("unexpected error when trying to stop non-running task: %v", err)
		}
		t.Fatalf("invalid event sequence allowed: stopping non-running tasks allowed")
	}
}

func TestPartialNoNonrunningStop(t *testing.T) {
	v := ForPartialSequence()

	evs := []events.TaskEvent{
		events.StartEvent{
			Task: "foo",
		},
	}
	ev := events.StopEvent{
		Task: "bar",
	}

	if err := v.ValidateSequence(evs, ev); !errors.Is(err, ErrTaskNotRunning) {
		if err != nil {
			t.Fatalf("unexpected error when trying to stop non-running task: %v", err)
		}
		t.Fatalf("invalid event sequence allowed: stopping non-running tasks allowed")
	}
}

func TestNoNonrunningSwitch(t *testing.T) {
	v := DefaultValidator

	evs := []events.TaskEvent{
		events.StartEvent{
			Task: "foo",
		},
	}
	ev := events.SwitchEvent{
		OldTask: "bar",
		NewTask: "baz",
	}

	if err := v.ValidateSequence(evs, ev); !errors.Is(err, ErrTaskNotRunning) {
		if err != nil {
			t.Fatalf("unexpected error when trying to switch from non-running task: %v", err)
		}
		t.Fatalf("invalid event sequence allowed: switching non-running tasks allowed")
	}
}

func TestPartialNoNonrunningSwitch(t *testing.T) {
	v := ForPartialSequence()

	evs := []events.TaskEvent{
		events.StartEvent{
			Task: "foo",
		},
	}
	ev := events.SwitchEvent{
		OldTask: "bar",
		NewTask: "baz",
	}

	if err := v.ValidateSequence(evs, ev); !errors.Is(err, ErrTaskNotRunning) {
		if err != nil {
			t.Fatalf("unexpected error when trying to switch from non-running task: %v", err)
		}
		t.Fatalf("invalid event sequence allowed: switching non-running tasks allowed")
	}
}
