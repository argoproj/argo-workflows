package commands

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEmissary(t *testing.T) {
	tmp := t.TempDir()

	varRunArgo = tmp
	includeScriptOutput = true

	wd, err := os.Getwd()
	assert.NoError(t, err)

	x := filepath.Join(wd, "../../../dist/argosay")

	err = ioutil.WriteFile(varRunArgo+"/template", []byte(`{}`), 0o600)
	assert.NoError(t, err)

	t.Run("Exit0", func(t *testing.T) {
		err := run(x, []string{"exit"})
		assert.NoError(t, err)
		data, err := ioutil.ReadFile(varRunArgo + "/ctr/main/exitcode")
		assert.NoError(t, err)
		assert.Equal(t, "0", string(data))
	})

	t.Run("Exit1", func(t *testing.T) {
		err := run(x, []string{"exit", "1"})
		assert.Equal(t, 1, err.(*exec.ExitError).ExitCode())
		data, err := ioutil.ReadFile(varRunArgo + "/ctr/main/exitcode")
		assert.NoError(t, err)
		assert.Equal(t, "1", string(data))
	})
	t.Run("Stdout", func(t *testing.T) {
		err := run(x, []string{"echo", "hello", "/dev/stdout"})
		assert.NoError(t, err)
		data, err := ioutil.ReadFile(varRunArgo + "/ctr/main/stdout")
		assert.NoError(t, err)
		assert.Contains(t, string(data), "hello")
	})
	t.Run("Comined", func(t *testing.T) {
		err := run(x, []string{"echo", "hello", "/dev/stderr"})
		assert.NoError(t, err)
		data, err := ioutil.ReadFile(varRunArgo + "/ctr/main/combined")
		assert.NoError(t, err)
		assert.Contains(t, string(data), "hello")
	})
	t.Run("Signal", func(t *testing.T) {
		for signal, message := range map[syscall.Signal]string{
			syscall.SIGTERM: "terminated",
			syscall.SIGKILL: "killed",
		} {
			err := ioutil.WriteFile(varRunArgo+"/ctr/main/signal", []byte(strconv.Itoa(int(signal))), 0o600)
			assert.NoError(t, err)
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := run(x, []string{"sleep", "5s"})
				assert.EqualError(t, err, "signal: "+message)
			}()
			time.Sleep(time.Second)
		}
	})
	t.Run("Artifact", func(t *testing.T) {
		err = ioutil.WriteFile(varRunArgo+"/template", []byte(`
{
	"outputs": {
		"artifacts": [
			{"path": "/tmp/artifact"}
		]
	}
}
`), 0o600)
		assert.NoError(t, err)
		err := run(x, []string{"echo", "hello", "/tmp/artifact"})
		assert.NoError(t, err)
		data, err := ioutil.ReadFile(varRunArgo + "/outputs/artifacts/tmp/artifact.tgz")
		assert.NoError(t, err)
		assert.NotEmpty(t, string(data)) // data is tgz format
	})
	t.Run("ArtifactWithTrailingAndLeadingSlash", func(t *testing.T) {
		err = ioutil.WriteFile(varRunArgo+"/template", []byte(`
{
	"outputs": {
		"artifacts": [
			{"path": "/tmp/artifact/"}
		]
	}
}
`), 0o600)
		assert.NoError(t, err)
		err := run(x, []string{"echo", "hello", "/tmp/artifact"})
		assert.NoError(t, err)
		data, err := ioutil.ReadFile(varRunArgo + "/outputs/artifacts/tmp/artifact.tgz")
		assert.NoError(t, err)
		assert.NotEmpty(t, string(data)) // data is tgz format
	})
	t.Run("Parameter", func(t *testing.T) {
		err = ioutil.WriteFile(varRunArgo+"/template", []byte(`
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
		assert.NoError(t, err)
		err := run(x, []string{"echo", "hello", "/tmp/parameter"})
		assert.NoError(t, err)
		data, err := ioutil.ReadFile(varRunArgo + "/outputs/parameters/tmp/parameter")
		assert.NoError(t, err)
		assert.Contains(t, string(data), "hello")
	})
	t.Run("RetryContainerSetFail", func(t *testing.T) {
		err = ioutil.WriteFile(varRunArgo+"/template", []byte(`
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
		assert.NoError(t, err)
		_ = os.Remove("test.txt")
		err = run(x, []string{"sh", "./test/containerSetRetryTest.sh", "/tmp/artifact"})
		assert.Error(t, err)
		data, err := ioutil.ReadFile(varRunArgo + "/outputs/artifacts/tmp/artifact.tgz")
		assert.NoError(t, err)
		assert.NotEmpty(t, string(data)) // data is tgz format
	})
	t.Run("RetryContainerSetSuccess", func(t *testing.T) {
		err = ioutil.WriteFile(varRunArgo+"/template", []byte(`
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
		assert.NoError(t, err)
		_ = os.Remove("test.txt")
		err = run(x, []string{"sh", "./test/containerSetRetryTest.sh", "/tmp/artifact"})
		assert.NoError(t, err)
		data, err := ioutil.ReadFile(varRunArgo + "/outputs/artifacts/tmp/artifact.tgz")
		assert.NoError(t, err)
		assert.NotEmpty(t, string(data)) // data is tgz format
	})
}

func run(name string, args []string) error {
	cmd := NewEmissaryCommand()
	containerName = "main"
	return cmd.RunE(cmd, append([]string{name}, args...))
}
