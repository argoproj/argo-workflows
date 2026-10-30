//go:build !windows

package maindriver

import (
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// TestPodStagerSharedDirs checks that with SharedDirs every directory the
// stager creates is writable by everyone, as in a pool pod's main, whose
// umask is zeroed: the worker, which may run as another user, removes them
// after the task. Without it they stay rwxr-xr-x, as in a pod.
func TestPodStagerSharedDirs(t *testing.T) {
	defer syscall.Umask(syscall.Umask(0))
	for _, shared := range []bool{true, false} {
		ctx := logging.TestContext(t.Context())
		src := t.TempDir()
		param := filepath.Join(src, "param")
		art := filepath.Join(src, "art")
		require.NoError(t, os.WriteFile(param, []byte("p"), 0o644))
		require.NoError(t, os.WriteFile(art, []byte("a"), 0o644))
		varRunArgo := t.TempDir()
		s := PodStager{VarRunArgo: varRunArgo, ContainerName: common.MainContainerName, Template: &wfv1.Template{}, SharedDirs: shared}
		require.NoError(t, s.Stage(ctx, "", Output{Kind: OutputParameter, Path: param}))
		require.NoError(t, s.Stage(ctx, "", Output{Kind: OutputArtifact, Path: art}))

		want := fs.FileMode(0o755)
		if shared {
			want = 0o777
		}
		var dirs int
		require.NoError(t, filepath.WalkDir(filepath.Join(varRunArgo, "outputs"), func(path string, d fs.DirEntry, err error) error {
			require.NoError(t, err)
			if d.IsDir() {
				dirs++
				info, err := d.Info()
				require.NoError(t, err)
				assert.Equal(t, want, info.Mode().Perm(), "shared=%v %s", shared, path)
			}
			return nil
		}))
		assert.Positive(t, dirs)
	}
}
