//go:build !windows

package commands

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v3/util/errors"
)

func TestEmissary(t *testing.T) {
	tmp := t.TempDir()

	varRunArgo = tmp
	includeScriptOutput = true

	err := os.WriteFile(varRunArgo+"/template", []byte(`{}`), 0o600)
	require.NoError(t, err)

	t.Run("Exit0", func(t *testing.T) {
		err := run("exit")
		require.NoError(t, err)
		data, err := os.ReadFile(varRunArgo + "/ctr/main/exitcode")
		require.NoError(t, err)
		assert.Equal(t, "0", string(data))
	})

	t.Run("Exit1", func(t *testing.T) {
		err := run("exit 1")
		assert.Equal(t, 1, err.(errors.Exited).ExitCode())
		data, err := os.ReadFile(varRunArgo + "/ctr/main/exitcode")
		require.NoError(t, err)
		assert.Equal(t, "1", string(data))
	})
	t.Run("Stdout", func(t *testing.T) {
		_ = os.Remove(varRunArgo + "/ctr/main/stdout")
		err := run("echo hello")
		require.NoError(t, err)
		data, err := os.ReadFile(varRunArgo + "/ctr/main/stdout")
		require.NoError(t, err)
		assert.Contains(t, string(data), "hello")
	})
	t.Run("Sub-process", func(t *testing.T) {
		_ = os.Remove(varRunArgo + "/ctr/main/stdout")
		err := run(`(sleep 60; echo 'should not wait for sub-process')& echo "hello\c"`)
		require.NoError(t, err)
		data, err := os.ReadFile(varRunArgo + "/ctr/main/stdout")
		require.NoError(t, err)
		assert.Equal(t, "hello", string(data))
	})
	t.Run("Combined", func(t *testing.T) {
		err := run("echo hello > /dev/stderr")
		require.NoError(t, err)
		data, err := os.ReadFile(varRunArgo + "/ctr/main/combined")
		require.NoError(t, err)
		assert.Contains(t, string(data), "hello")
	})
	t.Run("Signal", func(t *testing.T) {
		for signal := range map[syscall.Signal]string{
			syscall.SIGTERM: "terminated",
			syscall.SIGKILL: "killed",
		} {
			err := os.WriteFile(varRunArgo+"/ctr/main/signal", []byte(strconv.Itoa(int(signal))), 0o600)
			require.NoError(t, err)
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := run("sleep 3")
				require.EqualError(t, err, fmt.Sprintf("exit status %d", 128+signal))
			}()
			wg.Wait()
		}
	})
	t.Run("Artifact", func(t *testing.T) {
		err = os.WriteFile(varRunArgo+"/template", []byte(`
{
	"outputs": {
		"artifacts": [
			{"path": "/tmp/artifact"}
		]
	}
}
`), 0o600)
		require.NoError(t, err)
		err := run("echo hello > /tmp/artifact")
		require.NoError(t, err)
		data, err := os.ReadFile(varRunArgo + "/outputs/artifacts/tmp/artifact.tgz")
		require.NoError(t, err)
		assert.NotEmpty(t, string(data)) // data is tgz format
	})
	t.Run("ArtifactWithTrailingAndLeadingSlash", func(t *testing.T) {
		err = os.WriteFile(varRunArgo+"/template", []byte(`
{
	"outputs": {
		"artifacts": [
			{"path": "/tmp/artifact/"}
		]
	}
}
`), 0o600)
		require.NoError(t, err)
		err := run("echo hello > /tmp/artifact")
		require.NoError(t, err)
		data, err := os.ReadFile(varRunArgo + "/outputs/artifacts/tmp/artifact.tgz")
		require.NoError(t, err)
		assert.NotEmpty(t, string(data)) // data is tgz format
	})
	t.Run("Parameter", func(t *testing.T) {
		err = os.WriteFile(varRunArgo+"/template", []byte(`
{
	"outputs": {
		"parameters": [
			{
				"valueFrom": {"path": "/tmp/parameter"}
			}
		]
	}
}
`), 0o600)
		require.NoError(t, err)
		err := run("echo hello > /tmp/parameter")
		require.NoError(t, err)
		data, err := os.ReadFile(varRunArgo + "/outputs/parameters/tmp/parameter")
		require.NoError(t, err)
		assert.Contains(t, string(data), "hello")
	})
	t.Run("RetryContainerSetFail", func(t *testing.T) {
		err = os.WriteFile(varRunArgo+"/template", []byte(`
{
	"outputs": {
		"artifacts": [
			{
				"path": "/tmp/artifact/"
			}
		]
	},
	"containerSet": {
		"containers": [
			{	"name": "main"
			}
		],
		"retryStrategy":
		{
			"retries": 1
		}
	}
}
`), 0o600)
		require.NoError(t, err)
		_ = os.Remove("test.txt")
		err = run("sh ./test/containerSetRetryTest.sh /tmp/artifact")
		require.Error(t, err)
		data, err := os.ReadFile(varRunArgo + "/outputs/artifacts/tmp/artifact.tgz")
		require.NoError(t, err)
		assert.NotEmpty(t, string(data)) // data is tgz format
	})
	t.Run("RetryContainerSetSuccess", func(t *testing.T) {
		err = os.WriteFile(varRunArgo+"/template", []byte(`
{
	"outputs": {
		"artifacts": [
			{
				"path": "/tmp/artifact/"
			}
		]
	},
	"containerSet": {
		"containers": [
			{	"name": "main"
			}
		],
		"retryStrategy":
		{
			"retries": 2
		}
	}
}
`), 0o600)
		require.NoError(t, err)
		_ = os.Remove("test.txt")
		err = run("sh ./test/containerSetRetryTest.sh /tmp/artifact")
		require.NoError(t, err)
		data, err := os.ReadFile(varRunArgo + "/outputs/artifacts/tmp/artifact.tgz")
		require.NoError(t, err)
		assert.NotEmpty(t, string(data)) // data is tgz format
	})
}

func run(script string) error {
	cmd := NewEmissaryCommand()
	containerName = "main"
	return cmd.RunE(cmd, append([]string{"sh", "-c"}, script))
}
