package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ParseStringArray parses s as a comma-separated array of optionally-quoted strings.
func ParseStringArray(s string) ([]string, error) {
	re := regexp.MustCompile(`(?:(?:[^,]|\\,?)+)`)
	matches := re.FindAllString(s, -1)

	for i, match := range matches {
		// len(match) > 1 handles a corner case where the closing quote
		// is placed after the comma.
		if strings.HasPrefix(match, "\"") && strings.HasSuffix(match, "\"") && len(match) > 1 {
			var err error
			match, err = strconv.Unquote(match)
			if err != nil {
				return nil, err
			}
		}
		match = strings.Replace(match, "\\,", ",", -1)
		matches[i] = match
	}

	return matches, nil
}
