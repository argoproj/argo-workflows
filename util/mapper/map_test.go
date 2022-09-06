package mapper

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"testing"
)

type s struct {
	String string
}

func TestVisit(t *testing.T) {
	v := func(x any) (any, error) {
		s, ok := x.(string)
		if ok && s == "foo" {
			return "bar", nil
		}
		return x, nil
	}
	t.Run("string", func(t *testing.T) {
		x, err := Map("foo", v)
		assert.NoError(t, err)
		assert.Equal(t, "bar", x)
	})
	t.Run("Struct", func(t *testing.T) {
		x, err := Map(s{String: "foo"}, v)
		assert.NoError(t, err)
		assert.Equal(t, "bar", x.(s).String)
	})
	t.Run("array", func(t *testing.T) {
		x, err := Map([]string{"foo"}, v)
		assert.NoError(t, err)
		assert.Equal(t, []string{"bar"}, x)
	})
	t.Run("map", func(t *testing.T) {
		x, err := Map(map[string]string{"x": "foo"}, v)
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{"x": "bar"}, x)
	})
	t.Run("WorkflowSpec", func(t *testing.T) {
		y, err := Map(wfv1.WorkflowSpec{}, v)
		assert.NoError(t, err)
		assert.Equal(t, wfv1.WorkflowSpec{}, y)
	})
	t.Run("*WorkflowSpec", func(t *testing.T) {
		y, err := Map(&wfv1.WorkflowSpec{
			Entrypoint: "foo",
		}, v)
		assert.NoError(t, err)
		assert.Equal(t, &wfv1.WorkflowSpec{Entrypoint: "bar"}, y)
	})
}
