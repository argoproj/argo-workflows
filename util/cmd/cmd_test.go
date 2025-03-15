package cmd

import (
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
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

func TestIsURL(t *testing.T) {
	tests := []struct {
		name string
		args string
		want bool
	}{
		{
			name: "test is url",
			args: "http://www.foo.com",
			want: true,
		},
		{
			name: "test is not url",
			args: "www.foo.com",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsURL(tt.args); got != tt.want {
				t.Errorf("IsURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPrintVersionMismatchWarning(t *testing.T) {
	tests := []struct {
		name          string
		clientVersion *wfv1.Version
		serverVersion string
		expectedLog   string
	}{
		{
			name: "server version not set",
			clientVersion: &wfv1.Version{
				Version: "v3.1.0",
				GitTag:  "v3.1.0",
			},
			serverVersion: "",
		},
		{
			name: "client version is untagged",
			clientVersion: &wfv1.Version{
				Version: "v3.1.0",
			},
			serverVersion: "v3.1.1",
		},
		{
			name: "version mismatch",
			clientVersion: &wfv1.Version{
				Version: "v3.1.0",
				GitTag:  "v3.1.0",
			},
			serverVersion: "v3.1.1",
			expectedLog:   "CLI version (v3.1.0) does not match server version (v3.1.1). This can lead to unexpected behavior.",
		},
	}
	hook := &logtest.Hook{}
	log.AddHook(hook)
	defer log.StandardLogger().ReplaceHooks(nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			PrintVersionMismatchWarning(*tt.clientVersion, tt.serverVersion)
			if tt.expectedLog != "" {
				assert.Equal(t, tt.expectedLog, hook.LastEntry().Message)
			} else {
				assert.Nil(t, hook.LastEntry())
			}
		})
	}
}
