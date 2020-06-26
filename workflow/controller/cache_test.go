package controller

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"testing"
)

type MockCache struct {
	mock.Mock
}

var MockParamValue string = "Hello world"

var MockParam = wfv1.Parameter{
	Name: "hello",
	Value: &MockParamValue,
}

func (_m *MockCache) Load(key []byte) (*wfv1.Outputs, bool) {
	outputs := wfv1.Outputs{}
	outputs.Parameters = append(outputs.Parameters, MockParam)
	return &outputs, true
}

func (_m *MockCache) Save(key []byte, value string) bool {
	return true
}

func TestCacheLoad(t *testing.T) {
	mc := MockCache{}
	entry, ok := mc.Load([]byte(""))
	assert.Greater(t, len(entry.Parameters), 0)
	assert.True(t, ok)
}

func TestCacheSave(t *testing.T) {
	mc := MockCache{}
	assert.True(t, mc.Save([]byte(""), ""))
}
