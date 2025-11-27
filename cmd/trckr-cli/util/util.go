package util

import (
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

func MapToString(fields map[string]any) string {
	var res string

	keys := maps.Keys(fields)
	sortedKeys := slices.Sorted(keys)

	isFirst := true
	for _, key := range sortedKeys {
		value := fields[key]

		var fieldStr string
		if s, ok := value.(string); ok {
			fieldStr = fmt.Sprintf("%s=%s", key, strconv.Quote(s))
		} else {
			fieldStr = fmt.Sprintf("%s=%v", key, value)
		}

		if !isFirst {
			fieldStr = " " + fieldStr
		} else {
			isFirst = false
		}

		res += fieldStr
	}

	return res
}

func FormatDuration(duration time.Duration) string {
	s := duration.Round(time.Minute).String()

	s = strings.TrimSuffix(s, "0s")
	if s == "" {
		return "0m"
	}

	minIsZero := int(duration.Seconds())/60%60 == 0
	if minIsZero {
		s = strings.TrimSuffix(s, "0m")
		if s == "" {
			return "0m"
		}
	}

	return s
}

func GetStartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

func ParseWithScheme(s string, scheme string) (*url.URL, error) {
	u, err := url.Parse(s)
	if err != nil || u.Scheme == "" {
		u, err = url.Parse(fmt.Sprintf("%s://%s", scheme, s))
	} else if u.Scheme != scheme {
		return nil, fmt.Errorf("unexpected scheme %s: should be %s or left unspecified", u.Scheme, scheme)
	}
	return u, err
}
