package template

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Validate(t *testing.T) {
	t.Run("InvalidTemplate", func(t *testing.T) {
		err := Validate("{{", func(tag string) error { return fmt.Errorf("") })
		require.Error(t, err)
	})
	t.Run("InvalidTag", func(t *testing.T) {
		err := Validate("{{foo}}", func(tag string) error { return fmt.Errorf("%s", tag) })
		require.EqualError(t, err, "foo")
	})
	t.Run("Simple", func(t *testing.T) {
		err := Validate("{{foo}}", func(tag string) error { return nil })
		require.NoError(t, err)
	})
	t.Run("Expression", func(t *testing.T) {
		err := Validate("{{=foo}}", func(tag string) error { return fmt.Errorf("%s", tag) })
		require.NoError(t, err)
	})
}

func TestVariables(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   string
		want []string
	}{
		{"none", "plain", nil},
		{"simple", "{{pod.name}}-{{ inputs.parameters.x }}", []string{"pod.name", "inputs.parameters.x"}},
		{"expression", "{{=sprig.trunc(5, pod.name)}}", []string{"sprig", "sprig.trunc", "pod", "pod.name"}},
		{"indexed", `{{=pod["name"]}}`, []string{"pod", "pod.name"}},
		{"JSON-escaped", `"{{=pod[\"name\"]}}"`, []string{"pod", "pod.name"}},
		{"unparseable expression", "{{=pod.}}", nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Variables(tt.in)
			require.NoError(t, err)
			require.ElementsMatch(t, tt.want, got)
		})
	}
	_, err := Variables("{{")
	require.Error(t, err)
}
