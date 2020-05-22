package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		assert.NoError(t, argosay())
		assert.Error(t, argosay("garbage"))
	})
	t.Run("assert_contains", func(t *testing.T) {
		assert.NoError(t, argosay("echo", "foo", "/tmp/foo"))
		assert.Error(t, argosay("assert_contains"))
		assert.Error(t, argosay("assert_contains", "/tmp/foo"))
		assert.Error(t, argosay("assert_contains", "/tmp/not-exists", "foo"))
		assert.NoError(t, argosay("assert_contains", "/tmp/foo", "foo"))
		assert.Error(t, argosay("assert_contains", "/tmp/foo", "bar"))
	})
	t.Run("echo", func(t *testing.T) {
		assert.NoError(t, argosay("echo"))
		assert.NoError(t, argosay("echo", "foo"))
		assert.NoError(t, argosay("echo", "foo", "/tmp/foo"))
		assert.Error(t, argosay("echo", "foo", "/tmp/foo", "garbage"))
	})
	t.Run("cat", func(t *testing.T) {
		assert.NoError(t, argosay("cat", "/tmp/foo", "/tmp/foo"))
		assert.Error(t, argosay("cat", "/tmp/non"))
	})
	t.Run("sleep", func(t *testing.T) {
		assert.NoError(t, argosay("sleep", "1s"))
		assert.Error(t, argosay("sleep", "garbage"))
	})
	t.Run("exit", func(t *testing.T) {
		assert.NoError(t, argosay("exit"))
		assert.NoError(t, argosay("exit", "0"))
		assert.Equal(t, exitError{1}, argosay("exit", "1"))
	})
}
