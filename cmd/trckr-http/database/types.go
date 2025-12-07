package database

import (
	"database/sql/driver"
	"fmt"
	"strconv"

	"github.com/idloquy/trckr/pkg/util"
)

// StringArray represents a string array stored as a string where each item is quoted and comma-separated.
type StringArray struct {
	arr []string
}

func NewStringArray(arr []string) *StringArray {
	return &StringArray{arr: arr}
}

// Scan parses value as a string array.
func (a *StringArray) Scan(value any) error {
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to unmarshal StringArray value of unexpected type: %s", value)
	}

	var err error
	a.arr, err = util.ParseStringArray(s)
	if err != nil {
		return fmt.Errorf("failed to unmarshal StringArray value with incorrect format: %s", s)
	}

	return nil
}

// Value returns the string representation of the array.
func (a *StringArray) Value() (driver.Value, error) {
	if len(a.arr) == 0 {
		return nil, nil
	}

	var quotedJoinedArr string
	for i, s := range a.arr {
		if i < len(s)-1 {
			quotedJoinedArr += fmt.Sprintf("%s,", strconv.Quote(s))
		} else {
			quotedJoinedArr += strconv.Quote(s)
		}
	}

	return quotedJoinedArr, nil
}

func (StringArray) GormDataType() string {
	return "text"
}
