package client

import (
	"slices"
	"testing"
	"time"

	"github.com/idloquy/trckr/pkg/api"
	"github.com/idloquy/trckr/pkg/events"
)

func TestHistoryFromTaskEventsPartialProductiveStart(t *testing.T) {
	baseEvTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	evs := []api.EventContainer[api.TaskEvent]{
		{
			EventContainerMeta: api.EventContainerMeta{At: baseEvTime},
			Event: api.TaskEvent{
				TaskEventMeta: api.TaskEventMeta{ID: 1},
				TaskEvent: events.StartEvent{
					Task: "foo",
				},
			},
		},
	}

	expectedPeriods := []Period{
		PartialProductivePeriod{
			Task:      "foo",
			StartTime: baseEvTime,
		},
	}

	history, err := historyFromTaskEvents(evs)
	if err != nil {
		t.Fatalf("unexpected error when getting history from task events: %v", err)
	}

	if !slices.Equal(history.events, evs) {
		t.Fatalf("returned history has wrong events slice: should be %v but is %v", evs, history.events)
	}

	if !slices.Equal(history.periods, expectedPeriods) {
		t.Fatalf("returned history has wrong periods slice: should be %v but is %v", expectedPeriods, history.periods)
	}
}

func TestHistoryFromTaskEventsPartialProductiveSwitch(t *testing.T) {
	baseEvTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	evs := []api.EventContainer[api.TaskEvent]{
		{
			EventContainerMeta: api.EventContainerMeta{At: baseEvTime},
			Event: api.TaskEvent{
				TaskEventMeta: api.TaskEventMeta{ID: 1},
				TaskEvent: events.SwitchEvent{
					OldTask: "bar",
					NewTask: "foo",
				},
			},
		},
	}

	expectedPeriods := []Period{
		PartialProductivePeriod{
			Task:      "foo",
			StartTime: baseEvTime,
		},
	}

	history, err := historyFromTaskEvents(evs)
	if err != nil {
		t.Fatalf("unexpected error when getting history from task events: %v", err)
	}

	if !slices.Equal(history.events, evs) {
		t.Fatalf("returned history has wrong events slice: should be %v but is %v", evs, history.events)
	}

	if !slices.Equal(history.periods, expectedPeriods) {
		t.Fatalf("returned history has wrong periods slice: should be %v but is %v", expectedPeriods, history.periods)
	}
}

func TestHistoryFromTaskEventsProductiveStart(t *testing.T) {
	baseEvTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	evs := []api.EventContainer[api.TaskEvent]{
		{
			EventContainerMeta: api.EventContainerMeta{At: baseEvTime},
			Event: api.TaskEvent{
				TaskEventMeta: api.TaskEventMeta{ID: 1},
				TaskEvent: events.StartEvent{
					Task: "foo",
				},
			},
		},
		{
			EventContainerMeta: api.EventContainerMeta{At: baseEvTime.Add(1)},
			Event: api.TaskEvent{
				TaskEventMeta: api.TaskEventMeta{ID: 2},
				TaskEvent: events.StopEvent{
					Task: "foo",
				},
			},
		},
	}

	expectedPeriods := []Period{
		ProductivePeriod{
			Task:      "foo",
			StartTime: baseEvTime,
			StopTime:  baseEvTime.Add(1),
		},
	}

	history, err := historyFromTaskEvents(evs)
	if err != nil {
		t.Fatalf("unexpected error when getting history from task events: %v", err)
	}

	if !slices.Equal(history.events, evs) {
		t.Fatalf("returned history has wrong events slice: should be %v but is %v", evs, history.events)
	}

	if !slices.Equal(history.periods, expectedPeriods) {
		t.Fatalf("returned history has wrong periods slice: should be %v but is %v", expectedPeriods, history.periods)
	}
}

func TestHistoryFromTaskEventsProductiveSwitch(t *testing.T) {
	baseEvTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	evs := []api.EventContainer[api.TaskEvent]{
		{
			EventContainerMeta: api.EventContainerMeta{At: baseEvTime},
			Event: api.TaskEvent{
				TaskEventMeta: api.TaskEventMeta{ID: 1},
				TaskEvent: events.SwitchEvent{
					OldTask: "bar",
					NewTask: "foo",
				},
			},
		},
		{
			EventContainerMeta: api.EventContainerMeta{At: baseEvTime.Add(1)},
			Event: api.TaskEvent{
				TaskEventMeta: api.TaskEventMeta{ID: 2},
				TaskEvent: events.StopEvent{
					Task: "foo",
				},
			},
		},
	}

	expectedPeriods := []Period{
		ProductivePeriod{
			Task:      "foo",
			StartTime: baseEvTime,
			StopTime:  baseEvTime.Add(1),
		},
	}

	history, err := historyFromTaskEvents(evs)
	if err != nil {
		t.Fatalf("unexpected error when getting history from task events: %v", err)
	}

	if !slices.Equal(history.events, evs) {
		t.Fatalf("returned history has wrong events slice: should be %v but is %v", evs, history.events)
	}

	if !slices.Equal(history.periods, expectedPeriods) {
		t.Fatalf("returned history has wrong periods slice: should be %v but is %v", expectedPeriods, history.periods)
	}
}

func TestHistoryFromTaskEventsProductiveStartNonProductive(t *testing.T) {
	baseEvTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	evs := []api.EventContainer[api.TaskEvent]{
		{
			EventContainerMeta: api.EventContainerMeta{At: baseEvTime},
			Event: api.TaskEvent{
				TaskEventMeta: api.TaskEventMeta{ID: 1},
				TaskEvent: events.StartEvent{
					Task: "foo",
				},
			},
		},
		{
			EventContainerMeta: api.EventContainerMeta{At: baseEvTime.Add(1)},
			Event: api.TaskEvent{
				TaskEventMeta: api.TaskEventMeta{ID: 2},
				TaskEvent: events.StopEvent{
					Task: "foo",
				},
			},
		},
		{
			EventContainerMeta: api.EventContainerMeta{At: baseEvTime.Add(2)},
			Event: api.TaskEvent{
				TaskEventMeta: api.TaskEventMeta{ID: 3},
				TaskEvent: events.StartEvent{
					Task:       "foo",
					StopReason: "lorem ipsum",
				},
			},
		},
	}

	expectedPeriods := []Period{
		ProductivePeriod{
			Task:      "foo",
			StartTime: baseEvTime,
			StopTime:  baseEvTime.Add(1),
		},
		NonProductivePeriod{
			Reason:    "lorem ipsum",
			StartTime: baseEvTime.Add(1),
			StopTime:  baseEvTime.Add(2),
		},
		PartialProductivePeriod{
			Task:      "foo",
			StartTime: baseEvTime.Add(2),
		},
	}

	history, err := historyFromTaskEvents(evs)
	if err != nil {
		t.Fatalf("unexpected error when getting history from task events: %v", err)
	}

	if !slices.Equal(history.events, evs) {
		t.Fatalf("returned history has wrong events slice: should be %v but is %v", evs, history.events)
	}

	if !slices.Equal(history.periods, expectedPeriods) {
		t.Fatalf("returned history has wrong periods slice: should be %v but is %v", expectedPeriods, history.periods)
	}
}

func TestHistoryFromTaskEventsProductiveSwitchNonProductive(t *testing.T) {
	baseEvTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	evs := []api.EventContainer[api.TaskEvent]{
		{
			EventContainerMeta: api.EventContainerMeta{At: baseEvTime},
			Event: api.TaskEvent{
				TaskEventMeta: api.TaskEventMeta{ID: 1},
				TaskEvent: events.SwitchEvent{
					OldTask: "bar",
					NewTask: "foo",
				},
			},
		},
		{
			EventContainerMeta: api.EventContainerMeta{At: baseEvTime.Add(1)},
			Event: api.TaskEvent{
				TaskEventMeta: api.TaskEventMeta{ID: 2},
				TaskEvent: events.StopEvent{
					Task: "foo",
				},
			},
		},
		{
			EventContainerMeta: api.EventContainerMeta{At: baseEvTime.Add(2)},
			Event: api.TaskEvent{
				TaskEventMeta: api.TaskEventMeta{ID: 3},
				TaskEvent: events.StartEvent{
					Task:       "foo",
					StopReason: "lorem ipsum",
				},
			},
		},
	}

	expectedPeriods := []Period{
		ProductivePeriod{
			Task:      "foo",
			StartTime: baseEvTime,
			StopTime:  baseEvTime.Add(1),
		},
		NonProductivePeriod{
			Reason:    "lorem ipsum",
			StartTime: baseEvTime.Add(1),
			StopTime:  baseEvTime.Add(2),
		},
		PartialProductivePeriod{
			Task:      "foo",
			StartTime: baseEvTime.Add(2),
		},
	}

	history, err := historyFromTaskEvents(evs)
	if err != nil {
		t.Fatalf("unexpected error when getting history from task events: %v", err)
	}

	if !slices.Equal(history.events, evs) {
		t.Fatalf("returned history has wrong events slice: should be %v but is %v", evs, history.events)
	}

	if !slices.Equal(history.periods, expectedPeriods) {
		t.Fatalf("returned history has wrong periods slice: should be %v but is %v", expectedPeriods, history.periods)
	}
}

func TestHistoryFromTaskEventsProductiveStartPartialProductiveSwitch(t *testing.T) {
	baseEvTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	evs := []api.EventContainer[api.TaskEvent]{
		{
			EventContainerMeta: api.EventContainerMeta{At: baseEvTime},
			Event: api.TaskEvent{
				TaskEventMeta: api.TaskEventMeta{ID: 1},
				TaskEvent: events.StartEvent{
					Task: "foo",
				},
			},
		},
		{
			EventContainerMeta: api.EventContainerMeta{At: baseEvTime.Add(1)},
			Event: api.TaskEvent{
				TaskEventMeta: api.TaskEventMeta{ID: 2},
				TaskEvent: events.SwitchEvent{
					OldTask: "foo",
					NewTask: "bar",
				},
			},
		},
	}

	expectedPeriods := []Period{
		ProductivePeriod{
			Task:      "foo",
			StartTime: baseEvTime,
			StopTime:  baseEvTime.Add(1),
		},
		PartialProductivePeriod{
			Task:      "bar",
			StartTime: baseEvTime.Add(1),
		},
	}

	history, err := historyFromTaskEvents(evs)
	if err != nil {
		t.Fatalf("unexpected error when getting history from task events: %v", err)
	}

	if !slices.Equal(history.events, evs) {
		t.Fatalf("returned history has wrong events slice: should be %v but is %v", evs, history.events)
	}

	if !slices.Equal(history.periods, expectedPeriods) {
		t.Fatalf("returned history has wrong periods slice: should be %v but is %v", expectedPeriods, history.periods)
	}
}

func TestHistoryFromTaskEventsProductiveSwitchProductiveSwitch(t *testing.T) {
	baseEvTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	evs := []api.EventContainer[api.TaskEvent]{
		{
			EventContainerMeta: api.EventContainerMeta{At: baseEvTime},
			Event: api.TaskEvent{
				TaskEventMeta: api.TaskEventMeta{ID: 1},
				TaskEvent: events.SwitchEvent{
					OldTask: "bar",
					NewTask: "foo",
				},
			},
		},
		{
			EventContainerMeta: api.EventContainerMeta{At: baseEvTime.Add(1)},
			Event: api.TaskEvent{
				TaskEventMeta: api.TaskEventMeta{ID: 2},
				TaskEvent: events.SwitchEvent{
					OldTask: "foo",
					NewTask: "bar",
				},
			},
		},
	}

	expectedPeriods := []Period{
		ProductivePeriod{
			Task:      "foo",
			StartTime: baseEvTime,
			StopTime:  baseEvTime.Add(1),
		},
		PartialProductivePeriod{
			Task:      "bar",
			StartTime: baseEvTime.Add(1),
		},
	}

	history, err := historyFromTaskEvents(evs)
	if err != nil {
		t.Fatalf("unexpected error when getting history from task events: %v", err)
	}

	if !slices.Equal(history.events, evs) {
		t.Fatalf("returned history has wrong events slice: should be %v but is %v", evs, history.events)
	}

	if !slices.Equal(history.periods, expectedPeriods) {
		t.Fatalf("returned history has wrong periods slice: should be %v but is %v", expectedPeriods, history.periods)
	}
}

func TestHistorySplitByTaskPartialProductiveDividerOnly(t *testing.T) {
	dividerTask := "foo"
	baseTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	periods := []Period{
		PartialProductivePeriod{
			Task:      dividerTask,
			StartTime: baseTime,
		},
	}

	h := &TaskHistory{
		periods: periods,
	}
	expectedHistories := []*TaskHistory{h}

	histories := h.SplitByTask(dividerTask)

	if len(histories) != len(expectedHistories) {
		t.Fatalf("unexpected value returned: length of the returned history should be %d but is %d", len(expectedHistories), len(histories))
	}
	for i, history := range histories {
		expectedHistory := expectedHistories[i]
		if !slices.Equal(expectedHistory.periods, history.periods) {
			t.Fatalf("unexpected value for periods in history at index %d: should be %v but is %v", i, expectedHistory, history)
		}
	}
}

func TestHistorySplitByTaskProductiveDividerOnly(t *testing.T) {
	dividerTask := "foo"
	baseTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	periods := []Period{
		ProductivePeriod{
			Task:      dividerTask,
			StartTime: baseTime,
			StopTime:  baseTime.Add(1),
		},
	}

	h := &TaskHistory{
		periods: periods,
	}
	expectedHistories := []*TaskHistory{h}

	histories := h.SplitByTask(dividerTask)

	if len(histories) != len(expectedHistories) {
		t.Fatalf("unexpected value returned: length of the returned history should be %d but is %d", len(expectedHistories), len(histories))
	}
	for i, history := range histories {
		expectedHistory := expectedHistories[i]
		if !slices.Equal(expectedHistory.periods, history.periods) {
			t.Fatalf("unexpected value for periods in history at index %d: should be %v but is %v", i, expectedHistory, history)
		}
	}
}

func TestHistorySplitByTaskProductiveDividerProductive(t *testing.T) {
	dividerTask := "foo"
	baseTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	periods := []Period{
		ProductivePeriod{
			Task:      dividerTask,
			StartTime: baseTime,
			StopTime:  baseTime.Add(1),
		},
		ProductivePeriod{
			Task:      "bar",
			StartTime: baseTime.Add(1),
			StopTime:  baseTime.Add(2),
		},
	}

	h := &TaskHistory{
		periods: periods,
	}
	expectedHistories := []*TaskHistory{
		&TaskHistory{
			periods: periods[:1],
		},
		&TaskHistory{
			periods: periods[1:],
		},
	}

	histories := h.SplitByTask(dividerTask)

	if len(histories) != len(expectedHistories) {
		t.Fatalf("unexpected value returned: length of the returned history should be %d but is %d", len(expectedHistories), len(histories))
	}
	for i, history := range histories {
		expectedHistory := expectedHistories[i]
		if !slices.Equal(expectedHistory.periods, history.periods) {
			t.Fatalf("unexpected value for periods in history at index %d: should be %v but is %v", i, expectedHistory.periods, history.periods)
		}
	}
}

func TestHistorySplitByTaskProductiveDividerProductiveDivider(t *testing.T) {
	dividerTask := "foo"
	baseTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	periods := []Period{
		ProductivePeriod{
			Task:      dividerTask,
			StartTime: baseTime,
			StopTime:  baseTime.Add(1),
		},
		ProductivePeriod{
			Task:      dividerTask,
			StartTime: baseTime.Add(1),
			StopTime:  baseTime.Add(2),
		},
	}

	h := &TaskHistory{
		periods: periods,
	}
	expectedHistories := []*TaskHistory{
		&TaskHistory{
			periods: periods[:1],
		},
		&TaskHistory{
			periods: periods[1:],
		},
	}

	histories := h.SplitByTask(dividerTask)

	if len(histories) != len(expectedHistories) {
		t.Fatalf("unexpected value returned: length of the returned history should be %d but is %d", len(expectedHistories), len(histories))
	}
	for i, history := range histories {
		expectedHistory := expectedHistories[i]
		if !slices.Equal(expectedHistory.periods, history.periods) {
			t.Fatalf("unexpected value for periods in history at index %d: should be %v but is %v", i, expectedHistory.periods, history.periods)
		}
	}
}

func TestHistorySplitByTaskProductiveDividerNonProductiveProductive(t *testing.T) {
	dividerTask := "foo"
	baseTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	periods := []Period{
		ProductivePeriod{
			Task:      dividerTask,
			StartTime: baseTime,
			StopTime:  baseTime.Add(1),
		},
		NonProductivePeriod{
			StartTime: baseTime.Add(1),
			StopTime:  baseTime.Add(2),
		},
		ProductivePeriod{
			Task:      "bar",
			StartTime: baseTime.Add(2),
			StopTime:  baseTime.Add(3),
		},
	}

	h := &TaskHistory{
		periods: periods,
	}
	expectedHistories := []*TaskHistory{
		&TaskHistory{
			periods: periods[:2],
		},
		&TaskHistory{
			periods: periods[2:],
		},
	}

	histories := h.SplitByTask(dividerTask)

	if len(histories) != len(expectedHistories) {
		t.Fatalf("unexpected value returned: length of the returned history should be %d but is %d", len(expectedHistories), len(histories))
	}
	for i, history := range histories {
		expectedHistory := expectedHistories[i]
		if !slices.Equal(expectedHistory.periods, history.periods) {
			t.Fatalf("unexpected value for periods in history at index %d: should be %v but is %v", i, expectedHistory.periods, history.periods)
		}
	}
}

func TestHistorySplitByTaskProductiveDividerNonProductiveProductiveDivider(t *testing.T) {
	dividerTask := "foo"
	baseTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	periods := []Period{
		ProductivePeriod{
			Task:      dividerTask,
			StartTime: baseTime,
			StopTime:  baseTime.Add(1),
		},
		NonProductivePeriod{
			StartTime: baseTime.Add(1),
			StopTime:  baseTime.Add(2),
		},
		ProductivePeriod{
			Task:      dividerTask,
			StartTime: baseTime.Add(2),
			StopTime:  baseTime.Add(3),
		},
	}

	h := &TaskHistory{
		periods: periods,
	}
	expectedHistories := []*TaskHistory{
		&TaskHistory{
			periods: periods[:2],
		},
		&TaskHistory{
			periods: periods[2:],
		},
	}

	histories := h.SplitByTask(dividerTask)

	if len(histories) != len(expectedHistories) {
		t.Fatalf("unexpected value returned: length of the returned history should be %d but is %d", len(expectedHistories), len(histories))
	}
	for i, history := range histories {
		expectedHistory := expectedHistories[i]
		if !slices.Equal(expectedHistory.periods, history.periods) {
			t.Fatalf("unexpected value for periods in history at index %d: should be %v but is %v", i, expectedHistory.periods, history.periods)
		}
	}
}

func TestHistorySplitByDayPartialProductiveOnly(t *testing.T) {
	baseTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	periods := []Period{
		PartialProductivePeriod{
			Task:      "foo",
			StartTime: baseTime,
		},
	}

	h := &TaskHistory{
		periods: periods,
	}
	expectedHistories := []*TaskHistory{
		&TaskHistory{
			periods: periods,
		},
	}

	histories := h.SplitByDay()

	if len(histories) != len(expectedHistories) {
		t.Fatalf("unexpected value returned: length of the returned history should be %d but is %d", len(expectedHistories), len(histories))
	}
	for i, history := range histories {
		expectedHistory := expectedHistories[i]
		if !slices.Equal(expectedHistory.periods, history.periods) {
			t.Fatalf("unexpected value for periods in history at index %d: should be %v but is %v", i, expectedHistory.periods, history.periods)
		}
	}
}

func TestHistorySplitByDayProductiveOnly(t *testing.T) {
	baseTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	periods := []Period{
		ProductivePeriod{
			Task:      "foo",
			StartTime: baseTime,
			StopTime:  baseTime.Add(1),
		},
	}

	h := &TaskHistory{
		periods: periods,
	}
	expectedHistories := []*TaskHistory{
		&TaskHistory{
			periods: periods,
		},
	}

	histories := h.SplitByDay()

	if len(histories) != len(expectedHistories) {
		t.Fatalf("unexpected value returned: length of the returned history should be %d but is %d", len(expectedHistories), len(histories))
	}
	for i, history := range histories {
		expectedHistory := expectedHistories[i]
		if !slices.Equal(expectedHistory.periods, history.periods) {
			t.Fatalf("unexpected value for periods in history at index %d: should be %v but is %v", i, expectedHistory.periods, history.periods)
		}
	}
}

func TestHistorySplitByDayProductiveProductive(t *testing.T) {
	baseTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	periods := []Period{
		ProductivePeriod{
			Task:      "foo",
			StartTime: baseTime,
			StopTime:  baseTime.Add(1),
		},
		ProductivePeriod{
			StartTime: baseTime.Add(1),
			StopTime:  baseTime.Add(2),
		},
	}

	h := &TaskHistory{
		periods: periods,
	}
	expectedHistories := []*TaskHistory{
		&TaskHistory{
			periods: periods,
		},
	}

	histories := h.SplitByDay()

	if len(histories) != len(expectedHistories) {
		t.Fatalf("unexpected value returned: length of the returned history should be %d but is %d", len(expectedHistories), len(histories))
	}
	for i, history := range histories {
		expectedHistory := expectedHistories[i]
		if !slices.Equal(expectedHistory.periods, history.periods) {
			t.Fatalf("unexpected value for periods in history at index %d: should be %v but is %v", i, expectedHistory.periods, history.periods)
		}
	}
}

func TestHistorySplitByDayProductiveProductiveDiffDay(t *testing.T) {
	baseTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	periods := []Period{
		ProductivePeriod{
			Task:      "foo",
			StartTime: baseTime,
			StopTime:  baseTime.Add(1),
		},
		ProductivePeriod{
			StartTime: baseTime.Add(time.Hour * 24),
			StopTime:  baseTime.Add(time.Hour*24 + 1),
		},
	}

	h := &TaskHistory{
		periods: periods,
	}
	expectedHistories := []*TaskHistory{
		&TaskHistory{
			periods: periods[:1],
		},
		&TaskHistory{
			periods: periods[1:],
		},
	}

	histories := h.SplitByDay()

	if len(histories) != len(expectedHistories) {
		t.Fatalf("unexpected value returned: length of the returned history should be %d but is %d", len(expectedHistories), len(histories))
	}
	for i, history := range histories {
		expectedHistory := expectedHistories[i]
		if !slices.Equal(expectedHistory.periods, history.periods) {
			t.Fatalf("unexpected value for periods in history at index %d: should be %v but is %v", i, expectedHistory.periods, history.periods)
		}
	}
}

func TestHistorySplitByDayProductiveNonProductive(t *testing.T) {
	baseTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	periods := []Period{
		ProductivePeriod{
			Task:      "foo",
			StartTime: baseTime,
			StopTime:  baseTime.Add(1),
		},
		NonProductivePeriod{
			StartTime: baseTime.Add(1),
			StopTime:  baseTime.Add(2),
		},
		ProductivePeriod{
			StartTime: baseTime.Add(2),
			StopTime:  baseTime.Add(3),
		},
	}

	h := &TaskHistory{
		periods: periods,
	}
	expectedHistories := []*TaskHistory{
		&TaskHistory{
			periods: periods,
		},
	}

	histories := h.SplitByDay()

	if len(histories) != len(expectedHistories) {
		t.Fatalf("unexpected value returned: length of the returned history should be %d but is %d", len(expectedHistories), len(histories))
	}
	for i, history := range histories {
		expectedHistory := expectedHistories[i]
		if !slices.Equal(expectedHistory.periods, history.periods) {
			t.Fatalf("unexpected value for periods in history at index %d: should be %v but is %v", i, expectedHistory.periods, history.periods)
		}
	}
}

func TestHistorySplitByDayProductiveNonProductiveDiffDay(t *testing.T) {
	baseTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	periods := []Period{
		ProductivePeriod{
			Task:      "foo",
			StartTime: baseTime,
			StopTime:  baseTime.Add(time.Hour * 24),
		},
		NonProductivePeriod{
			StartTime: baseTime.Add(time.Hour * 24),
			StopTime:  baseTime.Add(time.Hour*24 + 1),
		},
		ProductivePeriod{
			StartTime: baseTime.Add(time.Hour*24 + 1),
			StopTime:  baseTime.Add(time.Hour*24 + 2),
		},
	}

	h := &TaskHistory{
		periods: periods,
	}
	expectedHistories := []*TaskHistory{
		&TaskHistory{
			periods: periods[:2],
		},
		&TaskHistory{
			periods: periods[2:],
		},
	}

	histories := h.SplitByDay()

	if len(histories) != len(expectedHistories) {
		t.Fatalf("unexpected value returned: length of the returned history should be %d but is %d", len(expectedHistories), len(histories))
	}
	for i, history := range histories {
		expectedHistory := expectedHistories[i]
		if !slices.Equal(expectedHistory.periods, history.periods) {
			t.Fatalf("unexpected value for periods in history at index %d: should be %v but is %v", i, expectedHistory.periods, history.periods)
		}
	}
}

func TestHistorySplitByDayProductivePartialProductive(t *testing.T) {
	baseTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	periods := []Period{
		ProductivePeriod{
			Task:      "foo",
			StartTime: baseTime,
			StopTime:  baseTime.Add(1),
		},
		PartialProductivePeriod{
			StartTime: baseTime.Add(1),
		},
	}

	h := &TaskHistory{
		periods: periods,
	}
	expectedHistories := []*TaskHistory{
		&TaskHistory{
			periods: periods,
		},
	}

	histories := h.SplitByDay()

	if len(histories) != len(expectedHistories) {
		t.Fatalf("unexpected value returned: length of the returned history should be %d but is %d", len(expectedHistories), len(histories))
	}
	for i, history := range histories {
		expectedHistory := expectedHistories[i]
		if !slices.Equal(expectedHistory.periods, history.periods) {
			t.Fatalf("unexpected value for periods in history at index %d: should be %v but is %v", i, expectedHistory.periods, history.periods)
		}
	}
}

func TestHistorySplitByDayProductivePartialProductiveDiffDay(t *testing.T) {
	baseTime := time.Date(1970, 0, 0, 0, 0, 0, 0, time.Local)

	periods := []Period{
		ProductivePeriod{
			Task:      "foo",
			StartTime: baseTime,
			StopTime:  baseTime.Add(time.Hour * 24),
		},
		PartialProductivePeriod{
			StartTime: baseTime.Add(time.Hour * 24),
		},
	}

	h := &TaskHistory{
		periods: periods,
	}
	expectedHistories := []*TaskHistory{
		&TaskHistory{
			periods: periods[:1],
		},
		&TaskHistory{
			periods: periods[1:],
		},
	}

	histories := h.SplitByDay()

	if len(histories) != len(expectedHistories) {
		t.Fatalf("unexpected value returned: length of the returned history should be %d but is %d", len(expectedHistories), len(histories))
	}
	for i, history := range histories {
		expectedHistory := expectedHistories[i]
		if !slices.Equal(expectedHistory.periods, history.periods) {
			t.Fatalf("unexpected value for periods in history at index %d: should be %v but is %v", i, expectedHistory.periods, history.periods)
		}
	}
}
