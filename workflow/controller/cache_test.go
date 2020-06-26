package controller

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/controller/mocks"
	"github.com/stretchr/testify/assert"
	"testing"
)

var MockParamValue string = "Hello world"

var MockParam = wfv1.Parameter{
	Name: "hello",
	Value: &MockParamValue,
}

func TestCacheLoad(t *testing.T) {
	mockCache := mocks.MemoizationCache{}
	entry, ok := mockCache.Load("")
	assert.Greater(t, len(entry.Parameters), 0)
	assert.True(t, ok)
}

func TestCacheSave(t *testing.T) {
	outputs := wfv1.Outputs{}
	outputs.Parameters = append(outputs.Parameters, MockParam)
	mockCache := mocks.MemoizationCache{}
	ok := mockCache.Save("", &outputs)
	assert.True(t, ok)
}
