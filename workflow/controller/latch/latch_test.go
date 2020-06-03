package latch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/argoproj/argo/test"
)

func Test_latch(t *testing.T) {
	a := test.LoadUnstructuredFromBytes([]byte((`
kind: Workflow 
metadata:
  resourceVersion: a`)))

	b := test.LoadUnstructuredFromBytes([]byte((`
kind: Workflow 
metadata:
  resourceVersion: b`)))

	latch := New()
	latch.Remove(&unstructured.Unstructured{})

	latch.Update(a)
	assert.True(t, latch.Pass(a))

	latch.Update(b)
	assert.False(t, latch.Pass(a))
	assert.True(t, latch.Pass(b))

	latch.Remove(b)
	assert.True(t, latch.Pass(a))
}
