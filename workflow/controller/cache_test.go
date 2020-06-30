package controller

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestConfigMapCacheLoad(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(&sampleConfigMapCacheEntry)
	assert.NoError(t, err)
	c := NewConfigMapCache("whalesay-cache", "default", controller.kubeclientset)
	entry, ok := c.Load("hi-there-world")
	assert.True(t, ok)
	assert.Equal(t, "hello", entry.Parameters[0].Name)
	assert.Equal(t, sampleOutput, *entry.Parameters[0].Value)
}

func TestConfigMapCacheSave(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	c := NewConfigMapCache("whalesay-cache", "default", controller.kubeclientset)
	outputs := wfv1.Outputs{}
	outputs.Parameters = append(outputs.Parameters, MockParam)
	ok := c.Save("hi-there-world", &outputs)
	assert.True(t, ok)
	cm, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Get("whalesay-cache", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, cm)
}

