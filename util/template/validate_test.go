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
