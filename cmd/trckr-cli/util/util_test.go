package util

import (
	"net/url"
	"testing"
	"time"
)

func TestFormatDurationStripIrrelevant(t *testing.T) {
	expected := "0m"
	res := FormatDuration(time.Duration(0))
	if res != expected {
		t.Fatalf("unexpected value returned: should be %s but is %s", expected, res)
	}

	expected = "1h"
	res = FormatDuration(time.Duration(time.Hour))
	if res != expected {
		t.Fatalf("unexpected value returned: should be %s but is %s", expected, res)
	}
}

func TestFormatDurationZero(t *testing.T) {
	expected := "0m"
	res := FormatDuration(time.Duration(0))

	if res != expected {
		t.Fatalf("unexpected value returned: should be %s but is %s", expected, res)
	}
}

func TestFormatDurationRoundDown(t *testing.T) {
	expected := "0m"
	res := FormatDuration(time.Second * 29)

	if res != expected {
		t.Fatalf("unexpected value returned: should be %s but is %s", expected, res)
	}
}

func TestFormatDurationRoundUp(t *testing.T) {
	expected := "1m"
	res := FormatDuration(time.Second * 30)

	if res != expected {
		t.Fatalf("unexpected value returned: should be %s but is %s", expected, res)
	}
}

func TestGetStartOfDay(t *testing.T) {
	expected := time.Date(1970, 0, 1, 0, 0, 0, 0, time.Local)
	dayTime := time.Date(1970, 0, 1, 0, 0, 0, 0, time.Local)

	resTime := GetStartOfDay(dayTime)
	if !resTime.Equal(expected) {
		t.Fatalf("unexpected value returned: should be %s but is %s", expected, resTime)
	}

	dayTime = time.Date(1970, 0, 1, 12, 0, 0, 0, time.Local)
	resTime = GetStartOfDay(dayTime)
	if !resTime.Equal(expected) {
		t.Fatalf("unexpected value returned: should be %s but is %s", expected, resTime)
	}

	dayTime = time.Date(1970, 0, 1, 23, 59, 59, 999, time.Local)
	resTime = GetStartOfDay(dayTime)
	if !resTime.Equal(expected) {
		t.Fatalf("unexpected value returned: should be %s but is %s", expected, resTime)
	}
}

func TestParseWithSchemeNoScheme(t *testing.T) {
	expected, _ := url.Parse("http://127.0.0.1")

	res, err := ParseWithScheme("127.0.0.1", "http")
	if err != nil {
		t.Fatalf("unexpected error while parsing URL: %v", err)
	}

	if *res != *expected {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, res)
	}
}

func TestParseWithSchemeExistingScheme(t *testing.T) {
	expected, _ := url.Parse("http://127.0.0.1")

	res, err := ParseWithScheme("http://127.0.0.1", "http")
	if err != nil {
		t.Fatalf("unexpected error while parsing URL: %v", err)
	}

	if *res != *expected {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, res)
	}
}

func TestParseWithSchemeWrongScheme(t *testing.T) {
	_, err := ParseWithScheme("http://127.0.0.1", "ws")
	if err == nil {
		t.Fatal("URL with wrong scheme allowed")
	}
}

func TestMapToStringQuote(t *testing.T) {
	expected := "key1=\"value1\""
	fields := map[string]any{"key1": "value1"}
	res := MapToString(fields)
	if res != expected {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, res)
	}
}

func TestMapToStringNonString(t *testing.T) {
	expected := "key1=1"
	fields := map[string]any{"key1": 1}
	res := MapToString(fields)
	if res != expected {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, res)
	}
}

func TestMapToStringMultipleString(t *testing.T) {
	expected := "key1=\"value1\" key2=\"value2\""
	fields := map[string]any{"key1": "value1", "key2": "value2"}
	res := MapToString(fields)
	if res != expected {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, res)
	}
}

func TestMapToStringMultipleNonString(t *testing.T) {
	expected := "key1=1 key2=2"
	fields := map[string]any{"key1": 1, "key2": 2}
	res := MapToString(fields)
	if res != expected {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, res)
	}
}

func TestMapToStringMutipleStringNonString(t *testing.T) {
	expected := "key1=\"value1\" key2=2"
	fields := map[string]any{"key1": "value1", "key2": 2}
	res := MapToString(fields)
	if res != expected {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, res)
	}
}

func TestMapToStringMutipleNonStringString(t *testing.T) {
	expected := "key1=1 key2=\"value2\""
	fields := map[string]any{"key1": 1, "key2": "value2"}
	res := MapToString(fields)
	if res != expected {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, res)
	}
}

func TestMapToStringMultipleInterleavedString(t *testing.T) {
	expected := "key1=1 key2=\"value2\" key3=3"
	fields := map[string]any{"key1": 1, "key2": "value2", "key3": 3}
	res := MapToString(fields)
	if res != expected {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, res)
	}
}

func TestMapToStringMultipleInterleavedNonString(t *testing.T) {
	expected := "key1=\"value1\" key2=2 key3=\"value3\""
	fields := map[string]any{"key1": "value1", "key2": 2, "key3": "value3"}
	res := MapToString(fields)
	if res != expected {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, res)
	}
}
