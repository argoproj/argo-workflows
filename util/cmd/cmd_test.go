package cmd

import (
	"reflect"
	"testing"
)

func TestMakeParseLabels(t *testing.T) {
	successCases := []struct {
		name     string
		labels   string
		expected map[string]string
	}{
		{
			name:   "test1",
			labels: "foo=false",
			expected: map[string]string{
				"foo": "false",
			},
		},
		{
			name:   "test2",
			labels: "foo=true,bar=123",
			expected: map[string]string{
				"foo": "true",
				"bar": "123",
			},
		},
	}
	for _, test := range successCases {
		got, err := ParseLabels(test.labels)
		if err != nil {
			t.Errorf("unexpected error :%v", err)
		}
		if !reflect.DeepEqual(test.expected, got) {
			t.Errorf("\nexpected:\n%v\ngot:\n%v", test.expected, got)
		}
	}

	errorCases := []struct {
		name   string
		labels interface{}
	}{
		{
			name:   "non-string",
			labels: 123,
		},
		{
			name:   "empty string",
			labels: "",
		},
		{
			name:   "error format",
			labels: "abc=456;bcd=789",
		},
		{
			name:   "error format",
			labels: "abc=456.bcd=789",
		},
		{
			name:   "error format",
			labels: "abc,789",
		},
		{
			name:   "error format",
			labels: "abc",
		},
		{
			name:   "error format",
			labels: "=abc",
		},
	}
	for _, test := range errorCases {
		_, err := ParseLabels(test.labels)
		if err == nil {
			t.Errorf("labels %s expect error, reason: %s, got nil", test.labels, test.name)
		}
	}
}
