package database

import (
	"testing"
)

func TestStringArrayValueSingle(t *testing.T) {
	expected := `"foo"`

	strArr := &StringArray{arr: []string{"foo"}}
	value, err := strArr.Value()
	if err != nil {
		t.Fatalf("unexpected error converting StringArray to string: %v", err)
	}

	s, ok := value.(string)
	if !ok {
		t.Fatalf("unexpected return type from conversion from StringArray to string: %v", s)
	}

	if s != expected {
		t.Fatalf("unexpected value returned: should be %s but is %s", expected, s)
	}
}

func TestStringArrayValueMultiple(t *testing.T) {
	expected := `"foo","bar"`

	strArr := &StringArray{arr: []string{"foo", "bar"}}
	value, err := strArr.Value()
	if err != nil {
		t.Fatalf("unexpected error converting StringArray to string: %v", err)
	}

	s, ok := value.(string)
	if !ok {
		t.Fatalf("unexpected return type from conversion from StringArray to string: %v", s)
	}

	if s != expected {
		t.Fatalf("unexpected value returned: should be %s but is %s", expected, s)
	}
}
