package typeassert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBool(t *testing.T) {
	expected := true
	t.Run("AssertBool", func(t *testing.T) {
		v, err := Bool(true)
		if assert.NoError(t, err) {
			assert.EqualValues(t, expected, v)
		}
	})
	t.Run("AssertNonBool", func(t *testing.T) {
		expectedError := "Interface is not a bool"
		_, err := Bool("true")
		if assert.Error(t, err) {
			assert.Equal(t, expectedError, err.Error())
		}
	})
	t.Run("AssertEmptyInterface", func(t *testing.T) {
		_, err := Bool(nil)
		assert.NoError(t, err)
	})
}

func TestFloat64(t *testing.T) {
	expected := 0.000
	t.Run("AssertFloat64", func(t *testing.T) {
		v, err := Float64(0.000)
		if assert.NoError(t, err) {
			assert.EqualValues(t, expected, v)
		}
	})
	t.Run("AssertNonFloat64", func(t *testing.T) {
		expectedError := "Interface is not a float64"
		_, err := Float64("0.000")
		if assert.Error(t, err) {
			assert.Equal(t, expectedError, err.Error())
		}
	})
	t.Run("AssertEmptyInterface", func(t *testing.T) {
		_, err := Float64(nil)
		assert.NoError(t, err)
	})
}

func TestString(t *testing.T) {
	expected := "string"
	t.Run("AssertString", func(t *testing.T) {
		v, err := String("string")
		if assert.NoError(t, err) {
			assert.EqualValues(t, expected, v)
		}
	})
	t.Run("AssertNonString", func(t *testing.T) {
		expectedError := "Interface is not a string"
		_, err := String(0)
		if assert.Error(t, err) {
			assert.Equal(t, expectedError, err.Error())
		}
	})
	t.Run("AssertEmptyInterface", func(t *testing.T) {
		_, err := String(nil)
		assert.NoError(t, err)
	})
}

func TestStringSlice(t *testing.T) {
	t.Run("AssertStringSlice", func(t *testing.T) {
		expected := []string{"Test"}
		tData := []string{"Test"}
		tDataIf := make([]interface{}, len(tData))
		for i := range tData {
			tDataIf[i] = tData[i]
		}
		v, err := StringSlice(tDataIf)
		if assert.NoError(t, err) {
			assert.EqualValues(t, expected, v)
		}
	})
	t.Run("NonStringSlice", func(t *testing.T) {
		expected := "Interface is not a string"
		tData := []int32{1, 2}
		tDataIf := make([]interface{}, len(tData))
		for i := range tData {
			tDataIf[i] = tData[i]
		}
		_, err := StringSlice(tDataIf)
		if assert.Error(t, err) {
			assert.EqualValues(t, expected, err.Error())
		}
	})
	t.Run("AssertEmptyInterface", func(t *testing.T) {
		_, err := StringSlice(nil)
		assert.NoError(t, err)
	})
}
