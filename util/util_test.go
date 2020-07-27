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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RecoverWorkflowNameFromSelectorString(tt.args.selector)
			assert.NoError(t, err)
			if got != tt.want {
				t.Errorf("RecoverWorkflowNameFromSelectorString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecoverWorkflowNameFromSelectorStringError(t *testing.T) {
	name, err := RecoverWorkflowNameFromSelectorString("whatever=whalesay")
	assert.NotNil(t, err)
	assert.Equal(t, name, "")
}
