//go:build windows

package commands

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v3/util/errors"
)

func TestEmissary(t *testing.T) {
	tmp := t.TempDir()

	varRunArgo = tmp
	includeScriptOutput = true

	err := os.WriteFile(varRunArgo+"/template", []byte(`{}`), 0o600)
	assert.NoError(t, err)

	t.Run("Exit0", func(t *testing.T) {
		err := run("exit")
		assert.NoError(t, err)
		data, err := os.ReadFile(varRunArgo + "/ctr/main/exitcode")
		assert.NoError(t, err)
		assert.Equal(t, "0", string(data))
	})

	t.Run("Exit1", func(t *testing.T) {
		err := run("exit 1")
		assert.Equal(t, 1, err.(errors.Exited).ExitCode())
		data, err := os.ReadFile(varRunArgo + "/ctr/main/exitcode")
		assert.NoError(t, err)
		assert.Equal(t, "1", string(data))
	})
	t.Run("Stdout", func(t *testing.T) {
		err := run("echo hello")
		assert.NoError(t, err)
		data, err := os.ReadFile(varRunArgo + "/ctr/main/stdout")
		assert.NoError(t, err)
		assert.Contains(t, string(data), "hello")
	})
	t.Run("Combined", func(t *testing.T) {
		err := run("echo hello > /dev/stderr")
		assert.NoError(t, err)
		data, err := os.ReadFile(varRunArgo + "/ctr/main/combined")
		assert.NoError(t, err)
		assert.Contains(t, string(data), "hello")
	})
	t.Run("Signal", func(t *testing.T) {
		for signal := range map[syscall.Signal]string{
			syscall.SIGTERM: "terminated",
			syscall.SIGKILL: "killed",
		} {
			err := os.WriteFile(varRunArgo+"/ctr/main/signal", []byte(strconv.Itoa(int(signal))), 0o600)
			assert.NoError(t, err)
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := run("sleep 3")
				assert.EqualError(t, err, fmt.Sprintf("exit status %d", 128+signal))
			}()
			wg.Wait()
		}
	})
	t.Run("ExitCode", func(t *testing.T) {
		defer wg.Done()
		err := run("exit 13")
		assert.Equal(t, 13, err.(errors.Exited).ExitCode())
		assert.EqualError(t, err, "exit status 13")
	})
}

func run(script string) error {
	cmd := NewEmissaryCommand()
	containerName = "main"
	return cmd.RunE(cmd, append([]string{"powershell", "-c"}, script))
}
