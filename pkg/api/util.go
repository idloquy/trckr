package api

import (
	"encoding/json"
)

func unmarshal[T any](data []byte) (T, error) {
	var value T
	err := json.Unmarshal(data, &value)
	return value, err
}
