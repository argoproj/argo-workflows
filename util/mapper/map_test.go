package mapper

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"
	"testing"
)

type s struct {
	Exported   string
	unexported string
}

func TestMap(t *testing.T) {
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
	t.Run("struct", func(t *testing.T) {
		x, err := Map(s{Exported: "foo"}, v)
		assert.NoError(t, err)
		assert.Equal(t, s{Exported: "bar"}, x)
	})
	t.Run("*struct", func(t *testing.T) {
		x, err := Map(&s{Exported: "foo"}, v)
		assert.NoError(t, err)
		assert.Equal(t, &s{Exported: "bar"}, x)
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
			Templates:    []wfv1.Template{{Name: "foo"}},
			Entrypoint:   "foo",
			Arguments:    wfv1.Arguments{},
			Priority:     pointer.Int32(1),
			Executor:     &wfv1.ExecutorConfig{},
			NodeSelector: map[string]string{"foo": "foo"},
		}, v)
		assert.NoError(t, err)
		assert.Equal(t, &wfv1.WorkflowSpec{
			Templates:    []wfv1.Template{{Name: "bar"}},
			Entrypoint:   "bar",
			Arguments:    wfv1.Arguments{},
			Priority:     pointer.Int32(1),
			Executor:     &wfv1.ExecutorConfig{},
			NodeSelector: map[string]string{"foo": "bar"},
		}, y)
	})
}
