package main

import (
	"slices"
	"testing"
	"time"

	"github.com/idloquy/trckr/cmd/trckr-http/database"
	"github.com/idloquy/trckr/pkg/api"
	"github.com/idloquy/trckr/pkg/events"
)

func TestGetTaskEventsWithinTimeRangeInexactFromExactTo(t *testing.T) {
	fromDate := time.Date(1970, 0, 0, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC)

	evs := []database.TaskEvent{
		{
			ID:        1,
			TaskEvent: events.StartEvent{},
			Time:      time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC),
		},
	}
	var expectedEvs []api.EventContainer[api.TaskEvent]
	for _, ev := range evs {
		expectedEvs = append(expectedEvs, dbTaskEvToAPITaskEv(ev))
	}

	filteredEvs := getTaskEventsWithinTimeRange(evs, fromDate, toDate)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetTaskEventsWithinTimeRangeInexactFromExactToMultiAll(t *testing.T) {
	fromDate := time.Date(1970, 0, 0, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(1972, 0, 0, 0, 0, 0, 0, time.UTC)

	evs := []database.TaskEvent{
		{
			ID:        1,
			TaskEvent: events.StartEvent{},
			Time:      time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        2,
			TaskEvent: events.SwitchEvent{},
			Time:      time.Date(1972, 0, 0, 0, 0, 0, 0, time.UTC),
		},
	}
	var expectedEvs []api.EventContainer[api.TaskEvent]
	for _, ev := range evs {
		expectedEvs = append(expectedEvs, dbTaskEvToAPITaskEv(ev))
	}

	filteredEvs := getTaskEventsWithinTimeRange(evs, fromDate, toDate)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetTaskEventsWithinTimeRangeInexactFromExactToMultiFirstOnly(t *testing.T) {
	fromDate := time.Date(1970, 0, 0, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC)

	evs := []database.TaskEvent{
		{
			ID:        1,
			TaskEvent: events.StartEvent{},
			Time:      time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC),
		},
		database.TaskEvent{
			ID:        2,
			TaskEvent: events.SwitchEvent{},
			Time:      time.Date(1972, 0, 0, 0, 0, 0, 0, time.UTC),
		},
	}
	expectedEvs := []api.EventContainer[api.TaskEvent]{dbTaskEvToAPITaskEv(evs[0])}

	filteredEvs := getTaskEventsWithinTimeRange(evs, fromDate, toDate)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetTaskEventsWithinTimeRangeInexactFromExactToMultiSecondOnly(t *testing.T) {
	fromDate := time.Date(1971, 5, 0, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(1972, 0, 0, 0, 0, 0, 0, time.UTC)

	evs := []database.TaskEvent{
		{
			ID:        1,
			TaskEvent: events.StartEvent{},
			Time:      time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC),
		},
		database.TaskEvent{
			ID:        2,
			TaskEvent: events.SwitchEvent{},
			Time:      time.Date(1972, 0, 0, 0, 0, 0, 0, time.UTC),
		},
	}
	expectedEvs := []api.EventContainer[api.TaskEvent]{dbTaskEvToAPITaskEv(evs[1])}

	filteredEvs := getTaskEventsWithinTimeRange(evs, fromDate, toDate)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetTaskEventsWithinTimeRangeExactFromInexactTo(t *testing.T) {
	fromDate := time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(1972, 0, 0, 0, 0, 0, 0, time.UTC)

	evs := []database.TaskEvent{
		{
			ID:        1,
			TaskEvent: events.StartEvent{},
			Time:      time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC),
		},
	}
	var expectedEvs []api.EventContainer[api.TaskEvent]
	for _, ev := range evs {
		expectedEvs = append(expectedEvs, dbTaskEvToAPITaskEv(ev))
	}

	filteredEvs := getTaskEventsWithinTimeRange(evs, fromDate, toDate)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetTaskEventsWithinTimeRangeExactFromInexactToMultiAll(t *testing.T) {
	fromDate := time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(1974, 0, 0, 0, 0, 0, 0, time.UTC)

	evs := []database.TaskEvent{
		{
			ID:        1,
			TaskEvent: events.StartEvent{},
			Time:      time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        2,
			TaskEvent: events.SwitchEvent{},
			Time:      time.Date(1973, 0, 0, 0, 0, 0, 0, time.UTC),
		},
	}
	var expectedEvs []api.EventContainer[api.TaskEvent]
	for _, ev := range evs {
		expectedEvs = append(expectedEvs, dbTaskEvToAPITaskEv(ev))
	}

	filteredEvs := getTaskEventsWithinTimeRange(evs, fromDate, toDate)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetTaskEventsWithinTimeRangeExactFromInexactToMultiFirst(t *testing.T) {
	fromDate := time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(1972, 0, 0, 0, 0, 0, 0, time.UTC)

	evs := []database.TaskEvent{
		{
			ID:        1,
			TaskEvent: events.StartEvent{},
			Time:      time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        2,
			TaskEvent: events.SwitchEvent{},
			Time:      time.Date(1973, 0, 0, 0, 0, 0, 0, time.UTC),
		},
	}
	expectedEvs := []api.EventContainer[api.TaskEvent]{dbTaskEvToAPITaskEv(evs[0])}

	filteredEvs := getTaskEventsWithinTimeRange(evs, fromDate, toDate)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetTaskEventsWithinTimeRangeExactFromInexactToMultiSecond(t *testing.T) {
	fromDate := time.Date(1973, 0, 0, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(1974, 0, 0, 0, 0, 0, 0, time.UTC)

	evs := []database.TaskEvent{
		{
			ID:        1,
			TaskEvent: events.StartEvent{},
			Time:      time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        2,
			TaskEvent: events.SwitchEvent{},
			Time:      time.Date(1973, 0, 0, 0, 0, 0, 0, time.UTC),
		},
	}
	expectedEvs := []api.EventContainer[api.TaskEvent]{dbTaskEvToAPITaskEv(evs[1])}

	filteredEvs := getTaskEventsWithinTimeRange(evs, fromDate, toDate)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetTaskEventsWithinTimeRangeInexactFromInexactTo(t *testing.T) {
	fromDate := time.Date(1970, 0, 0, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(1972, 0, 0, 0, 0, 0, 0, time.UTC)

	evs := []database.TaskEvent{
		{
			ID:        1,
			TaskEvent: events.StartEvent{},
			Time:      time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC),
		},
	}
	var expectedEvs []api.EventContainer[api.TaskEvent]
	for _, ev := range evs {
		expectedEvs = append(expectedEvs, dbTaskEvToAPITaskEv(ev))
	}

	filteredEvs := getTaskEventsWithinTimeRange(evs, fromDate, toDate)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetTaskEventsWithinTimeRangeInexactFromInexactToMultiAll(t *testing.T) {
	fromDate := time.Date(1970, 0, 0, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(1974, 0, 0, 0, 0, 0, 0, time.UTC)

	evs := []database.TaskEvent{
		{
			ID:        1,
			TaskEvent: events.StartEvent{},
			Time:      time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        2,
			TaskEvent: events.SwitchEvent{},
			Time:      time.Date(1973, 0, 0, 0, 0, 0, 0, time.UTC),
		},
	}
	var expectedEvs []api.EventContainer[api.TaskEvent]
	for _, ev := range evs {
		expectedEvs = append(expectedEvs, dbTaskEvToAPITaskEv(ev))
	}

	filteredEvs := getTaskEventsWithinTimeRange(evs, fromDate, toDate)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetTaskEventsWithinTimeRangeInexactFromInexactToMultiFirst(t *testing.T) {
	fromDate := time.Date(1970, 0, 0, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(1972, 0, 0, 0, 0, 0, 0, time.UTC)

	evs := []database.TaskEvent{
		{
			ID:        1,
			TaskEvent: events.StartEvent{},
			Time:      time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        2,
			TaskEvent: events.SwitchEvent{},
			Time:      time.Date(1973, 0, 0, 0, 0, 0, 0, time.UTC),
		},
	}
	expectedEvs := []api.EventContainer[api.TaskEvent]{dbTaskEvToAPITaskEv(evs[0])}

	filteredEvs := getTaskEventsWithinTimeRange(evs, fromDate, toDate)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetTaskEventsWithinTimeRangeInexactFromInexactToMultiSecond(t *testing.T) {
	fromDate := time.Date(1972, 0, 0, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(1974, 0, 0, 0, 0, 0, 0, time.UTC)

	evs := []database.TaskEvent{
		{
			ID:        1,
			TaskEvent: events.StartEvent{},
			Time:      time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        2,
			TaskEvent: events.SwitchEvent{},
			Time:      time.Date(1973, 0, 0, 0, 0, 0, 0, time.UTC),
		},
	}
	expectedEvs := []api.EventContainer[api.TaskEvent]{dbTaskEvToAPITaskEv(evs[1])}

	filteredEvs := getTaskEventsWithinTimeRange(evs, fromDate, toDate)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetTaskEventsWithinTimeRangeExactFromExactTo(t *testing.T) {
	fromDate := time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC)

	evs := []database.TaskEvent{
		{
			ID:        1,
			TaskEvent: events.StartEvent{},
			Time:      time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC),
		},
	}

	var expectedEvs []api.EventContainer[api.TaskEvent]
	for _, ev := range evs {
		expectedEvs = append(expectedEvs, dbTaskEvToAPITaskEv(ev))
	}

	filteredEvs := getTaskEventsWithinTimeRange(evs, fromDate, toDate)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetTaskEventsWithinTimeRangeExactFromExactToMultiAll(t *testing.T) {
	fromDate := time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(1972, 0, 0, 0, 0, 0, 0, time.UTC)

	evs := []database.TaskEvent{
		{
			ID:        1,
			TaskEvent: events.StartEvent{},
			Time:      time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        2,
			TaskEvent: events.SwitchEvent{},
			Time:      time.Date(1972, 0, 0, 0, 0, 0, 0, time.UTC),
		},
	}

	var expectedEvs []api.EventContainer[api.TaskEvent]
	for _, ev := range evs {
		expectedEvs = append(expectedEvs, dbTaskEvToAPITaskEv(ev))
	}

	filteredEvs := getTaskEventsWithinTimeRange(evs, fromDate, toDate)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetTaskEventsWithinTimeRangeExactFromExactToMultiFirst(t *testing.T) {
	fromDate := time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC)
	evs := []database.TaskEvent{
		{
			ID:        1,
			TaskEvent: events.StartEvent{},
			Time:      time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        2,
			TaskEvent: events.SwitchEvent{},
			Time:      time.Date(1972, 0, 0, 0, 0, 0, 0, time.UTC),
		},
	}
	expectedEvs := []api.EventContainer[api.TaskEvent]{dbTaskEvToAPITaskEv(evs[0])}

	filteredEvs := getTaskEventsWithinTimeRange(evs, fromDate, toDate)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetTaskEventsWithinTimeRangeExactFromExactToMultiSecond(t *testing.T) {
	fromDate := time.Date(1972, 0, 0, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(1972, 0, 0, 0, 0, 0, 0, time.UTC)

	evs := []database.TaskEvent{
		{
			ID:        1,
			TaskEvent: events.StartEvent{},
			Time:      time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        2,
			TaskEvent: events.SwitchEvent{},
			Time:      time.Date(1972, 0, 0, 0, 0, 0, 0, time.UTC),
		},
	}
	expectedEvs := []api.EventContainer[api.TaskEvent]{dbTaskEvToAPITaskEv(evs[1])}

	filteredEvs := getTaskEventsWithinTimeRange(evs, fromDate, toDate)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetEventsWithinTimeRangeSkipNonInitiatingFirst(t *testing.T) {
	fromDate := time.Date(1970, 0, 0, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(1972, 0, 0, 0, 0, 0, 0, time.UTC)

	evs := []database.TaskEvent{
		{
			ID:        1,
			TaskEvent: events.StopEvent{},
			Time:      time.Date(1970, 0, 0, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        2,
			TaskEvent: events.StartEvent{},
			Time:      time.Date(1971, 0, 0, 0, 0, 0, 0, time.UTC),
		},
	}
	expectedEvs := []api.EventContainer[api.TaskEvent]{dbTaskEvToAPITaskEv(evs[1])}

	filteredEvs := getTaskEventsWithinTimeRange(evs, fromDate, toDate)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}

	evs = []database.TaskEvent{
		{
			ID:        1,
			TaskEvent: events.StopEvent{},
			Time:      time.Date(1970, 0, 0, 0, 0, 0, 0, time.UTC),
		},
	}
	expectedEvs = []api.EventContainer[api.TaskEvent]{}

	filteredEvs = getTaskEventsWithinTimeRange(evs, fromDate, toDate)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetEventsForDaysPartialDivider(t *testing.T) {
	numDays := 1
	dayDivider := "foo"

	evs := []database.TaskEvent{
		{
			ID: 1,
			TaskEvent: events.StartEvent{
				Task: dayDivider,
			},
		},
	}
	expectedEvs := []api.EventContainer[api.TaskEvent]{dbTaskEvToAPITaskEv(evs[0])}

	filteredEvs := getTaskEventsForDays(evs, numDays, dayDivider)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetEventsForDaysDividerFirstDividerOnly(t *testing.T) {
	numDays := 1
	dayDivider := "foo"

	evs := []database.TaskEvent{
		{
			ID: 1,
			TaskEvent: events.StartEvent{
				Task: dayDivider,
			},
		},
		{
			ID: 2,
			TaskEvent: events.StopEvent{
				Task: dayDivider,
			},
		},
	}
	expectedEvs := []api.EventContainer[api.TaskEvent]{}

	filteredEvs := getTaskEventsForDays(evs, numDays, dayDivider)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}

	numDays = 2

	for _, ev := range evs {
		expectedEvs = append(expectedEvs, dbTaskEvToAPITaskEv(ev))
	}

	filteredEvs = getTaskEventsForDays(evs, numDays, dayDivider)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetEventsForDaysDividerFirst(t *testing.T) {
	numDays := 1
	dayDivider := "foo"

	evs := []database.TaskEvent{
		{
			ID: 1,
			TaskEvent: events.StartEvent{
				Task: dayDivider,
			},
		},
		{
			ID: 2,
			TaskEvent: events.StopEvent{
				Task: dayDivider,
			},
		},
		{
			ID: 3,
			TaskEvent: events.StartEvent{
				Task: "bar",
			},
		},
	}
	expectedEvs := []api.EventContainer[api.TaskEvent]{dbTaskEvToAPITaskEv(evs[2])}

	filteredEvs := getTaskEventsForDays(evs, numDays, dayDivider)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}

	numDays = 2
	expectedEvs = []api.EventContainer[api.TaskEvent]{}
	for _, ev := range evs {
		expectedEvs = append(expectedEvs, dbTaskEvToAPITaskEv(ev))
	}

	filteredEvs = getTaskEventsForDays(evs, numDays, dayDivider)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetEventsForDaysSwitchDividerLast(t *testing.T) {
	numDays := 1
	dayDivider := "bar"

	evs := []database.TaskEvent{
		{
			ID: 1,
			TaskEvent: events.StartEvent{
				Task: "foo",
			},
		},
		{
			ID: 2,
			TaskEvent: events.SwitchEvent{
				OldTask: "foo",
				NewTask: dayDivider,
			},
		},
	}
	var expectedEvs []api.EventContainer[api.TaskEvent]
	for _, ev := range evs {
		expectedEvs = append(expectedEvs, dbTaskEvToAPITaskEv(ev))
	}

	filteredEvs := getTaskEventsForDays(evs, numDays, dayDivider)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetEventsForDaysSwitchDividerMiddle(t *testing.T) {
	numDays := 1
	dayDivider := "bar"

	evs := []database.TaskEvent{
		{
			ID: 1,
			TaskEvent: events.StartEvent{
				Task: "foo",
			},
		},
		{
			ID: 2,
			TaskEvent: events.SwitchEvent{
				OldTask: "foo",
				NewTask: dayDivider,
			},
		},
		{
			ID: 3,
			TaskEvent: events.SwitchEvent{
				OldTask: dayDivider,
				NewTask: "baz",
			},
		},
	}
	expectedEvs := []api.EventContainer[api.TaskEvent]{dbTaskEvToAPITaskEv(evs[2])}

	filteredEvs := getTaskEventsForDays(evs, numDays, dayDivider)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}

	numDays = 2

	expectedEvs = []api.EventContainer[api.TaskEvent]{}
	for _, ev := range evs {
		expectedEvs = append(expectedEvs, dbTaskEvToAPITaskEv(ev))
	}

	filteredEvs = getTaskEventsForDays(evs, numDays, dayDivider)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}

func TestGetEventsForDaysUnknownDivider(t *testing.T) {
	numDays := 1
	dayDivider := "bar"

	evs := []database.TaskEvent{
		{
			ID: 1,
			TaskEvent: events.StartEvent{
				Task: "foo",
			},
		},
	}
	expectedEvs := []api.EventContainer[api.TaskEvent]{dbTaskEvToAPITaskEv(evs[0])}

	filteredEvs := getTaskEventsForDays(evs, numDays, dayDivider)
	if !slices.Equal(filteredEvs, expectedEvs) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expectedEvs, filteredEvs)
	}
}
