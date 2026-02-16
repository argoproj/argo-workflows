//go:build windows

package commands

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cmdutil "github.com/argoproj/argo-workflows/v3/util/cmd"
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
	t.Run("Exit13", func(t *testing.T) {
		err := run("exit 13")
		assert.Equal(t, 13, err.(errors.Exited).ExitCode())
		require.EqualError(t, err, "exit status 13")
	})
}

func run(script string) error {
	cmd := NewEmissaryCommand()
	_, _, err := cmdutil.ContextWithLogger(cmd, "info", "text")
	if err != nil {
		return err
	}
	containerName = "main"
	return cmd.RunE(cmd, append([]string{"powershell", "-c"}, script))
}
