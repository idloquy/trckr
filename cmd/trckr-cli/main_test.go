package main

import (
	"testing"
	"time"

	"github.com/idloquy/trckr/pkg/client"
)

func TestFormatPeriodMultiDayProductive(t *testing.T) {
	expected := "foo 00:00-\n|1970-01-02\nfoo -00:00 (24h)"

	startTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.Local)
	stopTime := time.Date(1970, 1, 2, 0, 0, 0, 0, time.Local)
	dur := client.ProductivePeriod{Task: "foo", StartTime: startTime, StopTime: stopTime}

	res := formatPeriod(dur, false)
	if res != expected {
		t.Fatalf("unexpected value returned: should be %s but is %s", expected, res)
	}

	expected = "foo 00:00-\n|1970-01-02\n|1970-01-03\nfoo -00:00 (48h)"

	startTime = time.Date(1970, 1, 1, 0, 0, 0, 0, time.Local)
	stopTime = time.Date(1970, 1, 3, 0, 0, 0, 0, time.Local)
	dur = client.ProductivePeriod{Task: "foo", StartTime: startTime, StopTime: stopTime}

	res = formatPeriod(dur, false)
	if res != expected {
		t.Fatalf("unexpected value returned: should be %s but is %s", expected, res)
	}
}

func TestFormatPeriodMultiDayNonProductive(t *testing.T) {
	expected := "1m\n|1970-01-02\n1m"

	startTime := time.Date(1970, 1, 1, 23, 59, 0, 0, time.Local)
	stopTime := time.Date(1970, 1, 2, 0, 1, 0, 0, time.Local)
	dur := client.NonProductivePeriod{StartTime: startTime, StopTime: stopTime}

	res := formatPeriod(dur, false)
	if res != expected {
		t.Fatalf("unexpected value returned: should be %s but is %s", expected, res)
	}

	expected = "1m\n|1970-01-02\n|1970-01-03\n24h1m"

	startTime = time.Date(1970, 1, 1, 23, 59, 0, 0, time.Local)
	stopTime = time.Date(1970, 1, 3, 0, 1, 0, 0, time.Local)
	dur = client.NonProductivePeriod{StartTime: startTime, StopTime: stopTime}

	res = formatPeriod(dur, false)
	if res != expected {
		t.Fatalf("unexpected value returned: should be %s but is %s", expected, res)
	}
}
