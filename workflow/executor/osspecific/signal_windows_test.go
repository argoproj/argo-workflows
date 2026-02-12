//go:build windows

package osspecific

import (
	"os/exec"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func TestKill(t *testing.T) {
	shell := "pwsh.exe"
	cmd := exec.Command(shell, "-c", `while(1) { sleep 600000 }`)

	_, err := StartCommand(logging.TestContext(t.Context()), cmd)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Go(func() {
		err = cmd.Wait()
		// we'll get an exit code
		assert.Error(t, err)
	})

	err = Kill(cmd.Process.Pid, syscall.SIGTERM)
	require.NoError(t, err)

	wg.Wait()
}
