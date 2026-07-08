//go:build !windows

package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	cmdutil "github.com/argoproj/argo-workflows/v4/util/cmd"
	"github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestEmissary(t *testing.T) {
	tmp := t.TempDir()

	varRunArgo = tmp
	includeScriptOutput = true

	err := os.WriteFile(varRunArgo+"/template", []byte(`{}`), 0o600)
	require.NoError(t, err)

	t.Run("Exit0", func(t *testing.T) {
		err = run("exit")
		require.NoError(t, err)
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/ctr/main/exitcode")
		require.NoError(t, err)
		assert.Equal(t, "0", string(data))
	})

	t.Run("Exit1", func(t *testing.T) {
		err = run("exit 1")
		assert.Equal(t, 1, err.(errors.Exited).ExitCode())
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/ctr/main/exitcode")
		require.NoError(t, err)
		assert.Equal(t, "1", string(data))
	})
	t.Run("Stdout", func(t *testing.T) {
		_ = os.Remove(varRunArgo + "/ctr/main/stdout")
		err = run("echo hello")
		require.NoError(t, err)
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/ctr/main/stdout")
		require.NoError(t, err)
		assert.Contains(t, string(data), "hello")
	})
	t.Run("Sub-process", func(t *testing.T) {
		_ = os.Remove(varRunArgo + "/ctr/main/stdout")
		err = run(`(sleep 60; echo 'should not wait for sub-process')& echo "hello\c"`)
		require.NoError(t, err)
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/ctr/main/stdout")
		require.NoError(t, err)
		assert.Equal(t, "hello", string(data))
	})
	t.Run("Combined", func(t *testing.T) {
		err = run("echo hello > /dev/stderr")
		require.NoError(t, err)
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/ctr/main/combined")
		require.NoError(t, err)
		assert.Contains(t, string(data), "hello")
	})
	t.Run("Signal", func(t *testing.T) {
		for signal := range map[syscall.Signal]string{
			syscall.SIGTERM: "terminated",
			syscall.SIGKILL: "killed",
		} {
			err = os.WriteFile(varRunArgo+"/ctr/main/signal", []byte(strconv.Itoa(int(signal))), 0o600)
			require.NoError(t, err)
			var wg sync.WaitGroup
			wg.Go(func() {
				runErr := run("sleep 3")
				assert.EqualError(t, runErr, fmt.Sprintf("exit status %d", 128+signal))
			})
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
		err = run("echo hello > /tmp/artifact")
		require.NoError(t, err)
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/outputs/artifacts/tmp/artifact.tgz")
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
		err = run("echo hello > /tmp/artifact")
		require.NoError(t, err)
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/outputs/artifacts/tmp/artifact.tgz")
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
		err = run("echo hello > /tmp/parameter")
		require.NoError(t, err)
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/outputs/parameters/tmp/parameter")
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
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/outputs/artifacts/tmp/artifact.tgz")
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
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/outputs/artifacts/tmp/artifact.tgz")
		require.NoError(t, err)
		assert.NotEmpty(t, string(data)) // data is tgz format
	})
}

func TestSaveParameterPathTraversal(t *testing.T) {
	tmp := t.TempDir()
	varRunArgo = tmp
	template = &wfv1.Template{} // isolate from any template left by other tests
	ctx := logging.TestContext(t.Context())
	// Open source paths relative to tmp so legitimate writes resolve there.
	t.Chdir(tmp)

	t.Run("LegitimateRelativePath", func(t *testing.T) {
		require.NoError(t, os.WriteFile("result.txt", []byte("hello"), 0o644))
		err := saveParameter(ctx, "result.txt")
		require.NoError(t, err)
		// The write path must actually run: the parameter is copied under outputs/parameters.
		data, err := os.ReadFile(filepath.Join(tmp, "outputs/parameters/result.txt"))
		require.NoError(t, err)
		assert.Equal(t, "hello", string(data))
	})
	t.Run("LegitimateInternalDotDot", func(t *testing.T) {
		require.NoError(t, os.MkdirAll("sub", 0o755))
		require.NoError(t, os.WriteFile("sub/p.txt", []byte("world"), 0o644))
		// sub/../sub/p.txt cleans to sub/p.txt, which stays in bounds and must succeed.
		err := saveParameter(ctx, "sub/../sub/p.txt")
		require.NoError(t, err)
		data, err := os.ReadFile(filepath.Join(tmp, "outputs/parameters/sub/p.txt"))
		require.NoError(t, err)
		assert.Equal(t, "world", string(data))
	})
	t.Run("TraversalToArgoexec", func(t *testing.T) {
		err := saveParameter(ctx, "../../argoexec")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "path traversal")
	})
	t.Run("TraversalToSidecarExitCode", func(t *testing.T) {
		err := saveParameter(ctx, "../../ctr/sidecar/exitcode")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "path traversal")
	})
	t.Run("TraversalToTemplate", func(t *testing.T) {
		err := saveParameter(ctx, "../template")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "path traversal")
	})
	t.Run("RootPathCleansToDot", func(t *testing.T) {
		// "/" and "x/.." both clean to "." which filepath.IsLocal accepts; the
		// guard must still reject them (never a valid single output path).
		for _, p := range []string{"/", "/tmp/..", ""} {
			err := saveParameter(ctx, p)
			require.Error(t, err, "expected rejection for %q", p)
			assert.Contains(t, err.Error(), "path traversal")
		}
	})
	t.Run("SymlinkedDestComponentBlocked", func(t *testing.T) {
		// "escape/p.txt" passes the lexical guard (IsLocal), so it reaches the
		// filesystem layer. If the main container planted outputs/parameters/escape
		// as a symlink out of the tree, the os.Root write must refuse to follow it.
		// This is the only test that exercises the os.Root sandbox: reverting to
		// os.Create/os.MkdirAll would let this write escape and must fail here.
		external := t.TempDir() // outside varRunArgo
		require.NoError(t, os.MkdirAll("escape", 0o755))
		require.NoError(t, os.WriteFile("escape/p.txt", []byte("data"), 0o644))
		require.NoError(t, os.MkdirAll(filepath.Join(tmp, "outputs/parameters"), 0o755))
		require.NoError(t, os.Symlink(external, filepath.Join(tmp, "outputs/parameters/escape")))

		err := saveParameter(ctx, "escape/p.txt")
		require.Error(t, err)
		require.NoFileExists(t, filepath.Join(external, "p.txt")) // must not escape the tree
	})
}

func TestSaveArtifactPathTraversal(t *testing.T) {
	tmp := t.TempDir()
	varRunArgo = tmp
	template = &wfv1.Template{} // isolate from any template left by other tests
	ctx := logging.TestContext(t.Context())
	t.Chdir(tmp)

	t.Run("LegitimateAbsolutePath", func(t *testing.T) {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, "artifact"), []byte("hello"), 0o644))
		err := saveArtifact(ctx, filepath.Join(tmp, "artifact"))
		require.NoError(t, err)
		// The tarball must actually be written under outputs/artifacts.
		data, err := os.ReadFile(filepath.Join(tmp, "outputs/artifacts", strings.TrimPrefix(tmp, "/"), "artifact.tgz"))
		require.NoError(t, err)
		assert.NotEmpty(t, data) // tgz content
	})
	t.Run("LegitimateInternalDotDot", func(t *testing.T) {
		require.NoError(t, os.MkdirAll("dir", 0o755))
		require.NoError(t, os.WriteFile("dir/a", []byte("hi"), 0o644))
		// dir/../dir/a cleans to dir/a, in bounds, must succeed and write the tarball.
		err := saveArtifact(ctx, "dir/../dir/a")
		require.NoError(t, err)
		data, err := os.ReadFile(filepath.Join(tmp, "outputs/artifacts/dir/a.tgz"))
		require.NoError(t, err)
		assert.NotEmpty(t, data)
	})
	t.Run("TraversalToArgoexec", func(t *testing.T) {
		err := saveArtifact(ctx, "../../argoexec")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "path traversal")
	})
	t.Run("TraversalToSidecarExitCode", func(t *testing.T) {
		err := saveArtifact(ctx, "../../ctr/sidecar/exitcode")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "path traversal")
	})
	t.Run("TraversalToTemplate", func(t *testing.T) {
		err := saveArtifact(ctx, "../template")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "path traversal")
	})
	t.Run("RootPathCleansToDot", func(t *testing.T) {
		// "/" cleans to "."; without the guard this would tar the whole source
		// root. "/tmp/.." and "" also clean to "." and must be rejected.
		for _, p := range []string{"/", "/tmp/..", ""} {
			err := saveArtifact(ctx, p)
			require.Error(t, err, "expected rejection for %q", p)
			assert.Contains(t, err.Error(), "path traversal")
		}
	})
	t.Run("SymlinkedDestComponentBlocked", func(t *testing.T) {
		// "escape/a" passes the lexical guard (IsLocal), so it reaches the
		// filesystem layer. If the main container planted outputs/artifacts/escape
		// as a symlink out of the tree, the os.Root write must refuse to follow it.
		// This is the only test that exercises the os.Root sandbox: reverting to
		// os.Create/os.MkdirAll would let this write escape and must fail here.
		external := t.TempDir() // outside varRunArgo
		require.NoError(t, os.MkdirAll("escape", 0o755))
		require.NoError(t, os.WriteFile("escape/a", []byte("data"), 0o644))
		require.NoError(t, os.MkdirAll(filepath.Join(tmp, "outputs/artifacts"), 0o755))
		require.NoError(t, os.Symlink(external, filepath.Join(tmp, "outputs/artifacts/escape")))

		err := saveArtifact(ctx, "escape/a")
		require.Error(t, err)
		require.NoFileExists(t, filepath.Join(external, "a.tgz")) // must not escape the tree
	})
}

func run(script string) error {
	cmd := NewEmissaryCommand()
	_, _, err := cmdutil.ContextWithLogger(cmd, string(logging.Info), string(logging.Text))
	if err != nil {
		return err
	}
	containerName = "main"
	return cmd.RunE(cmd, append([]string{"sh", "-c"}, script))
}
