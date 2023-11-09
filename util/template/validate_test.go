package template

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Validate(t *testing.T) {
	t.Run("InvalidTemplate", func(t *testing.T) {
		err := Validate("{{", func(tag string) error { return fmt.Errorf("") })
		assert.Error(t, err)
	})
	t.Run("InvalidTag", func(t *testing.T) {
		err := Validate("{{foo}}", func(tag string) error { return fmt.Errorf(tag) })
		assert.EqualError(t, err, "foo")
	})
	t.Run("Simple", func(t *testing.T) {
		err := Validate("{{foo}}", func(tag string) error { return nil })
		assert.NoError(t, err)
	})
	t.Run("Expression", func(t *testing.T) {
		err := Validate("{{=foo}}", func(tag string) error { return fmt.Errorf(tag) })
		assert.NoError(t, err)
	})
}
