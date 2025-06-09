package osspecific

import (
	"bytes"
	"io"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleStartCloser(t *testing.T) {
	shell := "sh"
	if runtime.GOOS == "windows" {
		shell = "pwsh.exe"
	}
	cmd := exec.Command(shell, "-c", `echo "A123456789B123456789C123456789D123456789E123456789\c"`)
	var stdoutWriter bytes.Buffer
	slowWriter := SlowWriter{
		&stdoutWriter,
	}
	// Command outputs are asynchronously written to cmd.Stdout.
	// Using SlowWriter causes the situation where the invoked command has exited but its outputs have not been written yet.
	cmd.Stdout = slowWriter

	closer, err := StartCommand(cmd)
	require.NoError(t, err)
	err = cmd.Wait()
	require.NoError(t, err)
	// Wait for echo command to exit before calling closer
	time.Sleep(100 * time.Millisecond)
	closer()

	expected := "A123456789B123456789C123456789D123456789E123456789"
	if runtime.GOOS == "windows" {
		expected = "A123456789B123456789C123456789D123456789E123456789\\c\r\n"
	}
	assert.Equal(t, expected, stdoutWriter.String())
}

type SlowWriter struct {
	Writer io.Writer
}

func (s SlowWriter) Write(data []byte) (n int, err error) {
	for i := range data {
		_, err := s.Writer.Write(data[i : i+1])
		if err != nil {
			return i, err
		}
		time.Sleep(7 * time.Millisecond)
	}
	return len(data), nil
}
