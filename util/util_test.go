package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateFieldSelectorFromWorkflowName(t *testing.T) {
	type args struct {
		wfName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"TestGenerateFieldSelectorFromWorkflowName", args{"whalesay"}, "metadata.name=whalesay"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateFieldSelectorFromWorkflowName(tt.args.wfName); got != tt.want {
				t.Errorf("GenerateFieldSelectorFromWorkflowName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecoverWorkflowNameFromSelectorString(t *testing.T) {
	type args struct {
		selector string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"TestRecoverWorkflowNameFromSelectorString", args{"metadata.name=whalesay"}, "whalesay"},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"metadata.name="}, ""},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"metadata.name=whalesay,other=hello"}, "whalesay"},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"foo=bar,metadata.name=whalesay,other=hello"}, "whalesay"},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"foo=bar,metadata.name=whalesay"}, "whalesay"},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"foo=bar,metadata.name= whalesay ,other=hello"}, "whalesay"},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"foo=bar,other=hello"}, ""},
		{"TestRecoverWorkflowNameFromSelectorString", args{"metadata.name=@latest"}, "@latest"},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"metadata.name="}, ""},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"metadata.name=@latest,other=hello"}, "@latest"},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"foo=bar,metadata.name=@latest,other=hello"}, "@latest"},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"foo=bar,metadata.name=@latest"}, "@latest"},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"foo=bar,metadata.name= @latest ,other=hello"}, "@latest"},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"foo=bar,other=hello"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RecoverWorkflowNameFromSelectorStringIfAny(tt.args.selector)
			if got != tt.want {
				t.Errorf("RecoverWorkflowNameFromSelectorStringIfAny() = %v, want %v", got, tt.want)
			}
		})
	}
	name := RecoverWorkflowNameFromSelectorStringIfAny("whatever=whalesay")
	assert.Equal(t, name, "")
	assert.NotPanics(t, func() {
		_ = RecoverWorkflowNameFromSelectorStringIfAny("whatever")
	})
}
