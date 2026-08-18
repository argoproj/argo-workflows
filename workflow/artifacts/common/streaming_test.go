package common

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBufferReaderToTempFile(t *testing.T) {
	t.Run("writes reader content to the temp file", func(t *testing.T) {
		path, cleanup, err := BufferReaderToTempFile(strings.NewReader("hello world"), "streaming-test-*")
		require.NoError(t, err)
		defer cleanup()

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(content))
	})

	t.Run("temp file name matches the given pattern", func(t *testing.T) {
		path, cleanup, err := BufferReaderToTempFile(strings.NewReader("x"), "my-driver-upload-*")
		require.NoError(t, err)
		defer cleanup()

		assert.Contains(t, path, "my-driver-upload-")
	})

	t.Run("cleanup removes the temp file and is safe to call more than once", func(t *testing.T) {
		path, cleanup, err := BufferReaderToTempFile(strings.NewReader("x"), "streaming-test-*")
		require.NoError(t, err)

		cleanup()
		_, statErr := os.Stat(path)
		assert.True(t, os.IsNotExist(statErr))

		// second call must not panic or error
		cleanup()
	})

	t.Run("reader error leaves no temp file behind", func(t *testing.T) {
		path, cleanup, err := BufferReaderToTempFile(&errorReader{}, "streaming-test-*")
		require.Error(t, err)
		assert.Empty(t, path)

		// cleanup is safe to call even though nothing was created
		cleanup()
	})
}

// errorReader is an io.Reader that always fails, used to exercise BufferReaderToTempFile's
// cleanup-on-error path.
type errorReader struct{}

func (r *errorReader) Read(_ []byte) (int, error) {
	return 0, assert.AnError
}

func TestSaveStreamViaTempFile(t *testing.T) {
	t.Run("save receives a path holding the reader's content and the file is removed after", func(t *testing.T) {
		var savedPath string
		err := SaveStreamViaTempFile(strings.NewReader("stream content"), "delegate-test-*", func(path string) error {
			savedPath = path
			content, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			assert.Equal(t, "stream content", string(content))
			return nil
		})
		require.NoError(t, err)

		_, statErr := os.Stat(savedPath)
		assert.True(t, os.IsNotExist(statErr), "temp file must be removed after save returns")
	})

	t.Run("save error is propagated and the temp file is removed", func(t *testing.T) {
		var savedPath string
		err := SaveStreamViaTempFile(strings.NewReader("x"), "delegate-test-*", func(path string) error {
			savedPath = path
			return assert.AnError
		})
		require.ErrorIs(t, err, assert.AnError)

		_, statErr := os.Stat(savedPath)
		assert.True(t, os.IsNotExist(statErr), "temp file must be removed even when save fails")
	})

	t.Run("reader error is returned without calling save", func(t *testing.T) {
		saveCalled := false
		err := SaveStreamViaTempFile(&errorReader{}, "delegate-test-*", func(string) error {
			saveCalled = true
			return nil
		})
		require.Error(t, err)
		assert.False(t, saveCalled)
	})
}
