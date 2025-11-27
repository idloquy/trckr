package events

import (
	"errors"
	"testing"
)

func TestSwitchEventNoSameTask(t *testing.T) {
	taskName := "foo"
	if _, err := NewSwitchEvent(taskName, taskName); !errors.Is(err, ErrSwitchSameTask) {
		if err != nil {
			t.Fatalf("unexpected error while trying to create switch event to the same task: %v", err)
		}
		t.Fatal("creating a switch event to the same task allowed")
	}
}

func TestStartEventNoEmptyTaskName(t *testing.T) {
	if _, err := NewStartEvent("", ""); !errors.Is(err, ErrEmptyTaskName) {
		if err != nil {
			t.Fatalf("unexpected error while trying to create start event with an empty task name: %v", err)
		}
		t.Fatal("creating a start event with an empty task name allowed")
	}

}

func TestStopEventNoEmptyTaskName(t *testing.T) {
	if _, err := NewStopEvent(""); !errors.Is(err, ErrEmptyTaskName) {
		if err != nil {
			t.Fatalf("unexpected error while trying to create stop event with an empty task name: %v", err)
		}
		t.Fatal("creating a stop event with an empty task name allowed")
	}
}

func TestSwitchEventNoEmptyTaskNames(t *testing.T) {
	nonEmptyName := "foo"
	if _, err := NewSwitchEvent("", nonEmptyName); !errors.Is(err, ErrEmptyTaskName) {
		if err != nil {
			t.Fatalf("unexpected error while trying to create switch event with the old task name empty: %v", err)
		}
		t.Fatal("creating a switch event with the old task name empty allowed")
	}

	if _, err := NewSwitchEvent(nonEmptyName, ""); !errors.Is(err, ErrEmptyTaskName) {
		if err != nil {
			t.Fatalf("unexpected error while trying to create switch event with the new task name empty: %v", err)
		}
		t.Fatal("creating a switch event with the new task name empty allowed")
	}

	if _, err := NewSwitchEvent("", ""); !errors.Is(err, ErrEmptyTaskName) {
		if err != nil {
			t.Fatalf("unexpected error while trying to create switch event with empty task names: %v", err)
		}
		t.Fatal("creating a switch event with empty task names allowed")
	}
}
