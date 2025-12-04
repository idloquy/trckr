package util

import (
	"slices"
	"testing"
)

func TestParseStringArraySingle(t *testing.T) {
	expected := []string{"foo"}
	arr, err := ParseStringArray("foo")
	if err != nil {
		t.Fatalf("unexpected error when parsing string array: %v", err)
	}

	if !slices.Equal(arr, expected) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, arr)
	}
}

func TestParseStringArraySingleQuoted(t *testing.T) {
	expected := []string{"foo"}
	arr, err := ParseStringArray(`"foo"`)
	if err != nil {
		t.Fatalf("unexpected error when parsing string array: %v", err)
	}

	if !slices.Equal(arr, expected) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, arr)
	}
}

func TestParseStringArraySingleTrailingComma(t *testing.T) {
	expected := []string{"foo"}
	arr, err := ParseStringArray("foo,")
	if err != nil {
		t.Fatalf("unexpected error when parsing string array: %v", err)
	}

	if !slices.Equal(arr, expected) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, arr)
	}
}

func TestParseStringArraySingleQuotedTrailingComma(t *testing.T) {
	expected := []string{"foo"}
	arr, err := ParseStringArray(`"foo",`)
	if err != nil {
		t.Fatalf("unexpected error when parsing string array: %v", err)
	}

	if !slices.Equal(arr, expected) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, arr)
	}
}

func TestParseStringArraySingleQuotedInnerTrailingComma(t *testing.T) {
	expected := []string{"\"foo", "\""}
	arr, err := ParseStringArray(`"foo,"`)
	if err != nil {
		t.Fatalf("unexpected error when parsing string array: %v", err)
	}

	if !slices.Equal(arr, expected) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, arr)
	}
}

func TestParseStringArraySingleLeadingComma(t *testing.T) {
	expected := []string{"foo"}
	arr, err := ParseStringArray(`,foo`)
	if err != nil {
		t.Fatalf("unexpected error when parsing string array: %v", err)
	}

	if !slices.Equal(arr, expected) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, arr)
	}
}

func TestParseStringArraySingleQuotedLeadingComma(t *testing.T) {
	expected := []string{"foo"}
	arr, err := ParseStringArray(`,"foo"`)
	if err != nil {
		t.Fatalf("unexpected error when parsing string array: %v", err)
	}

	if !slices.Equal(arr, expected) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, arr)
	}
}

func TestParseStringArraySingleQuotedInnerLeadingComma(t *testing.T) {
	expected := []string{"\"", "foo\""}
	arr, err := ParseStringArray(`",foo"`)
	if err != nil {
		t.Fatalf("unexpected error when parsing string array: %v", err)
	}

	if !slices.Equal(arr, expected) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, arr)
	}
}

func TestParseStringArraySingleOpeningQuote(t *testing.T) {
	expected := []string{`"foo`}
	arr, err := ParseStringArray(`"foo`)
	if err != nil {
		t.Fatalf("unexpected error when parsing string array: %v", err)
	}

	if !slices.Equal(arr, expected) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, arr)
	}
}

func TestParseStringArraySingleOpeningQuoteLeadingComma(t *testing.T) {
	expected := []string{`"foo`}
	arr, err := ParseStringArray(`,"foo`)
	if err != nil {
		t.Fatalf("unexpected error when parsing string array: %v", err)
	}

	if !slices.Equal(arr, expected) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, arr)
	}
}

func TestParseStringArraySingleOpeningQuoteInnerLeadingComma(t *testing.T) {
	expected := []string{"\"", `foo`}
	arr, err := ParseStringArray(`",foo`)
	if err != nil {
		t.Fatalf("unexpected error when parsing string array: %v", err)
	}

	if !slices.Equal(arr, expected) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, arr)
	}
}

func TestParseStringArraySingleOpeningQuoteTrailingComma(t *testing.T) {
	expected := []string{`"foo`}
	arr, err := ParseStringArray(`"foo,`)
	if err != nil {
		t.Fatalf("unexpected error when parsing string array: %v", err)
	}

	if !slices.Equal(arr, expected) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, arr)
	}
}

func TestParseStringArraySingleClosingQuote(t *testing.T) {
	expected := []string{`foo"`}
	arr, err := ParseStringArray(`foo"`)
	if err != nil {
		t.Fatalf("unexpected error when parsing string array: %v", err)
	}

	if !slices.Equal(arr, expected) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, arr)
	}
}

func TestParseStringArraySingleClosingQuoteLeadingComma(t *testing.T) {
	expected := []string{`foo"`}
	arr, err := ParseStringArray(`,foo"`)
	if err != nil {
		t.Fatalf("unexpected error when parsing string array: %v", err)
	}

	if !slices.Equal(arr, expected) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, arr)
	}
}

func TestParseStringArraySingleClosingQuoteTrailingComma(t *testing.T) {
	expected := []string{`foo"`}
	arr, err := ParseStringArray(`foo",`)
	if err != nil {
		t.Fatalf("unexpected error when parsing string array: %v", err)
	}

	if !slices.Equal(arr, expected) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, arr)
	}
}

func TestParseStringArraySingleClosingQuoteInnerTrailingComma(t *testing.T) {
	expected := []string{"foo", "\""}
	arr, err := ParseStringArray(`foo,"`)
	if err != nil {
		t.Fatalf("unexpected error when parsing string array: %v", err)
	}

	if !slices.Equal(arr, expected) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, arr)
	}
}

func TestParseStringArrayMultiple(t *testing.T) {
	expected := []string{"foo", "bar"}
	arr, err := ParseStringArray("foo,bar")
	if err != nil {
		t.Fatalf("unexpected error when parsing string array: %v", err)
	}

	if !slices.Equal(arr, expected) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, arr)
	}
}

func TestParseStringArrayMultipleOneQuoted(t *testing.T) {
	expected := []string{"foo", "bar"}
	arr, err := ParseStringArray(`"foo",bar`)
	if err != nil {
		t.Fatalf("unexpected error when parsing string array: %v", err)
	}

	if !slices.Equal(arr, expected) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, arr)
	}
}

func TestParseStringArrayMultipleAllQuoted(t *testing.T) {
	expected := []string{"foo", "bar"}
	arr, err := ParseStringArray(`"foo","bar"`)
	if err != nil {
		t.Fatalf("unexpected error when parsing string array: %v", err)
	}

	if !slices.Equal(arr, expected) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, arr)
	}
}

func TestParseStringArrayMultipleCollectiveQuote(t *testing.T) {
	expected := []string{"\"foo", "bar\""}
	arr, err := ParseStringArray(`"foo,bar"`)
	if err != nil {
		t.Fatalf("unexpected error when parsing string array: %v", err)
	}

	if !slices.Equal(arr, expected) {
		t.Fatalf("unexpected value returned: should be %v but is %v", expected, arr)
	}
}
