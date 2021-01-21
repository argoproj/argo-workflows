package main

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

func Test_run(t *testing.T) {
	tmp, err := ioutil.TempDir("", "")
	assert.NoError(t, err)

	varArgo = func(x string) string {
		return filepath.Join(tmp, x)
	}

	wd, err := os.Getwd()
	assert.NoError(t, err)

	x := filepath.Join(wd, "../../dist/argosay")

	err = ioutil.WriteFile(varArgo("template"), []byte(`
{
	"outputs": {
		"parameters": [
			{
				"valueFrom": {"path": "/tmp/parameter"}
			}
		],
		"artifacts": [
			{"path": "/tmp/artifact"}
		]
	}
}
`), 0600)
	assert.NoError(t, err)

	t.Run("Exit0", func(t *testing.T) {
		err := run(x, []string{"exit"})
		assert.NoError(t, err)
		data, err := ioutil.ReadFile(varArgo("exitcode"))
		assert.NoError(t, err)
		assert.Equal(t, "0", string(data))
	})
	t.Run("Exit1", func(t *testing.T) {
		err := run(x, []string{"exit", "1"})
		assert.Equal(t, 1, err.(*exec.ExitError).ExitCode())
		data, err := ioutil.ReadFile(varArgo("exitcode"))
		assert.NoError(t, err)
		assert.Equal(t, "1", string(data))
	})
	t.Run("Stdout", func(t *testing.T) {
		err := run(x, []string{"echo", "hello", "/dev/stdout"})
		assert.NoError(t, err)
		data, err := ioutil.ReadFile(varArgo("stdout"))
		assert.NoError(t, err)
		assert.Equal(t, "hello", string(data))
	})
	t.Run("Stderr", func(t *testing.T) {
		err := run(x, []string{"echo", "hello", "/dev/stderr"})
		assert.NoError(t, err)
		data, err := ioutil.ReadFile(varArgo("stderr"))
		assert.NoError(t, err)
		assert.Equal(t, "hello", string(data))
	})
	t.Run("Artifact", func(t *testing.T) {
		err := run(x, []string{"echo", "hello", "/tmp/artifact"})
		assert.NoError(t, err)
		data, err := ioutil.ReadFile(varArgo("outputs/tmp/artifact"))
		assert.NoError(t, err)
		assert.NotEmpty(t, string(data))
	})
	t.Run("Parameter", func(t *testing.T) {
		err := run(x, []string{"echo", "hello", "/tmp/parameter"})
		assert.NoError(t, err)
		data, err := ioutil.ReadFile(varArgo("outputs/tmp/parameter"))
		assert.NoError(t, err)
		assert.NotEmpty(t, string(data))
	})
	t.Run("Signal", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := run(x, []string{"sleep", "10s"})
			assert.EqualError(t, err, "signal: terminated")
		}()
		time.Sleep(2 * time.Second)
		err := ioutil.WriteFile(varArgo("signal"), []byte(strconv.Itoa(int(syscall.SIGTERM))), 0600)
		assert.NoError(t, err)
		wg.Wait()
	})

}
