//go:build windows

package os_specific

import (
	"os/exec"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKill(t *testing.T) {
	shell := "pwsh.exe"
	cmd := exec.Command(shell, "-c", `while(1) { sleep 600000 }`)

	_, err := StartCommand(cmd)
	require.NoError(t, err)

	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		defer wg.Done()

		err = cmd.Wait()
		// we'll get an exit code
		assert.Error(t, err)
	}()

	err = Kill(cmd.Process.Pid, syscall.SIGTERM)
	require.NoError(t, err)

	wg.Wait()
}
