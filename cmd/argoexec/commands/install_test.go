//go:build !windows

package commands

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// TestInstall runs `argoexec install` as a pool pod's init container would:
// only the shared volume, no kubeconfig, no service account and no task
// env. It must copy the binary and write nothing else.
func TestInstall(t *testing.T) {
	shared := t.TempDir()
	prevVarRunArgo := varRunArgo
	varRunArgo = shared
	t.Cleanup(func() { varRunArgo = prevVarRunArgo })

	bin := t.TempDir()
	binary := []byte("#!/bin/sh\necho argoexec\n")
	require.NoError(t, os.WriteFile(filepath.Join(bin, "argoexec"), binary, 0o755))
	t.Setenv("PATH", bin)

	t.Setenv("KUBECONFIG", filepath.Join(t.TempDir(), "missing"))
	for _, name := range []string{common.EnvVarTemplate, common.EnvVarPodName, common.EnvVarWorkflowName, common.EnvVarNodeID, common.EnvVarDeadline, "KUBERNETES_SERVICE_HOST", "KUBERNETES_SERVICE_PORT"} {
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"install"})
	require.NoError(t, cmd.Execute())

	installed := filepath.Join(shared, "argoexec")
	data, err := os.ReadFile(installed)
	require.NoError(t, err)
	assert.Equal(t, binary, data)
	info, err := os.Stat(installed)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o555), info.Mode().Perm(), "main may run as a non-root user")

	entries, err := os.ReadDir(shared)
	require.NoError(t, err)
	require.Len(t, entries, 1, "install writes only the binary")
	assert.NoFileExists(t, filepath.Join(shared, "template"))
}

func TestInstall_RejectsArgs(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"install", "extra"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	require.Error(t, cmd.Execute())
}
