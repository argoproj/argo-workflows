//go:build !windows

package executor

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	executorplugins "github.com/argoproj/argo-workflows/v4/pkg/plugins/executor"
	"github.com/argoproj/argo-workflows/v4/util/archive"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// A plugin can leave a named pipe at an output artifact's path, which the
// agent must not wait on.
func TestAgentPluginOutputArtifactNamedPipe(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	driver := &bucketDriver{dir: t.TempDir()}
	ae := newArtifactAgent(t, driver, pluginFunc(func(tmpl *wfv1.Template) (executorplugins.ExecuteTemplateReply, error) {
		return succeeded(nil), syscall.Mkfifo(tmpl.Outputs.Artifacts[0].Path, 0o644)
	}))

	result, _, err := ae.processTask(ctx, "my-wf-1", pluginTemplate(nil, wfv1.Artifacts{{Name: "out", Path: "/out.txt"}}))
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeError, result.Phase)
	assert.Contains(t, result.Message, "/out.txt is not a regular file, directory or symlink")
	assert.Zero(t, driver.saves)
}

func TestAgentPluginInputArtifactModes(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	driver := &bucketDriver{dir: t.TempDir()}
	require.NoError(t, os.WriteFile(filepath.Join(driver.dir, "file.txt"), []byte("file"), 0o600))
	src := filepath.Join(t.TempDir(), "dir")
	require.NoError(t, os.MkdirAll(src, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "file.txt"), []byte("in a directory"), 0o600))
	var tarball bytes.Buffer
	require.NoError(t, archive.TarGzToWriter(ctx, src, 6, &tarball))
	require.NoError(t, os.WriteFile(filepath.Join(driver.dir, "dir.tgz"), tarball.Bytes(), 0o600))

	modes := map[string]fs.FileMode{}
	ae := newArtifactAgent(t, driver, pluginFunc(func(tmpl *wfv1.Template) (executorplugins.ExecuteTemplateReply, error) {
		for _, art := range tmpl.Inputs.Artifacts {
			err := filepath.WalkDir(art.Path, func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				info, err := d.Info()
				if err != nil {
					return err
				}
				rel, err := filepath.Rel(filepath.Dir(art.Path), p)
				modes[rel] = info.Mode().Perm()
				return err
			})
			if err != nil {
				return executorplugins.ExecuteTemplateReply{}, err
			}
		}
		return succeeded(nil), nil
	}))
	mode := func(m int32) *int32 { return &m }

	result, _, err := ae.processTask(ctx, "my-wf-1", pluginTemplate(wfv1.Artifacts{
		{Name: "shared", Path: "/in/shared", ArtifactLocation: s3Key("dir.tgz")},
		{Name: "file", Path: "/in/file.txt", Mode: mode(0o440), ArtifactLocation: s3Key("file.txt")},
		{Name: "dir", Path: "/in/dir", Mode: mode(0o750), ArtifactLocation: s3Key("dir.tgz")},
		{Name: "recursed", Path: "/in/recursed", Mode: mode(0o750), RecurseMode: true, ArtifactLocation: s3Key("dir.tgz")},
	}, nil))
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, result.Phase, result.Message)
	assert.Equal(t, map[string]fs.FileMode{
		// open to plugins running as any user
		"shared":          0o777,
		"shared/file.txt": 0o666,
		"file.txt":        0o440,
		// as loaded, under the directory with the artifact's mode
		"dir":               0o750,
		"dir/file.txt":      0o600,
		"recursed":          0o750,
		"recursed/file.txt": 0o750,
	}, modes)
}
