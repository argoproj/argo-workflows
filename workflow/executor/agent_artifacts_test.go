package executor

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	argoerrs "github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	executorplugins "github.com/argoproj/argo-workflows/v4/pkg/plugins/executor"
	"github.com/argoproj/argo-workflows/v4/util/archive"
	"github.com/argoproj/argo-workflows/v4/util/file"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	artifactcommon "github.com/argoproj/argo-workflows/v4/workflow/artifacts/common"
	"github.com/argoproj/argo-workflows/v4/workflow/artifacts/resource"
)

const testArchiveKey = "my-wf/my-wf-plugin-1234"

// bucketDriver is an artifact driver that stores artifacts under a directory,
// by key.
type bucketDriver struct {
	dir   string
	mu    sync.Mutex
	loads int
	saves int
}

func (d *bucketDriver) path(art *wfv1.Artifact) (string, error) {
	if !art.HasLocation() {
		return "", errors.New("the artifact was not relocated to the archive location")
	}
	key, err := art.GetKey()
	return filepath.Join(d.dir, key), err
}

func (d *bucketDriver) Load(_ context.Context, art *wfv1.Artifact, path string) error {
	d.mu.Lock()
	d.loads++
	d.mu.Unlock()
	src, err := d.path(art)
	if err != nil {
		return err
	}
	if !file.Exists(src) {
		return argoerrs.Errorf(argoerrs.CodeNotFound, "%s not found", src)
	}
	return copyTree(src, path)
}

func (d *bucketDriver) Save(_ context.Context, path string, art *wfv1.Artifact) error {
	d.mu.Lock()
	d.saves++
	d.mu.Unlock()
	dst, err := d.path(art)
	if err != nil {
		return err
	}
	return copyTree(path, dst)
}

func (d *bucketDriver) OpenStream(context.Context, *wfv1.Artifact) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func (d *bucketDriver) SaveStream(context.Context, io.Reader, *wfv1.Artifact) error {
	return errors.New("not implemented")
}

func (d *bucketDriver) Delete(context.Context, *wfv1.Artifact) error {
	return errors.New("not implemented")
}

func (d *bucketDriver) ListObjects(context.Context, *wfv1.Artifact) ([]string, error) {
	return nil, errors.New("not implemented")
}

func (d *bucketDriver) IsDirectory(context.Context, *wfv1.Artifact) (bool, error) {
	return false, errors.New("not implemented")
}

// copyTree copies the file or directory at src to dst, following symlinks as
// artifact drivers do.
func copyTree(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if !info.IsDir() {
		var data []byte
		if data, err = os.ReadFile(src); err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o644)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		if err := copyTree(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// pluginFunc is a plugin executing every template it is offered.
type pluginFunc func(tmpl *wfv1.Template) (executorplugins.ExecuteTemplateReply, error)

func (f pluginFunc) ExecuteTemplate(_ context.Context, args executorplugins.ExecuteTemplateArgs, reply *executorplugins.ExecuteTemplateReply) error {
	r, err := f(args.Template)
	*reply = r
	return err
}

func succeeded(outputs *wfv1.Outputs) executorplugins.ExecuteTemplateReply {
	return executorplugins.ExecuteTemplateReply{Node: &wfv1.NodeResult{Phase: wfv1.NodeSucceeded, Outputs: outputs}}
}

// writeOutputs is a plugin writing each output artifact's name to its path.
func writeOutputs(tmpl *wfv1.Template) (executorplugins.ExecuteTemplateReply, error) {
	for _, art := range tmpl.Outputs.Artifacts {
		if err := os.WriteFile(art.Path, []byte(art.Name), 0o644); err != nil {
			return executorplugins.ExecuteTemplateReply{}, err
		}
	}
	return succeeded(nil), nil
}

func newArtifactAgent(t *testing.T, driver *bucketDriver, plugin executorplugins.TemplateExecutor) *AgentExecutor {
	return &AgentExecutor{
		consideredTasks: &sync.Map{},
		plugins:         []executorplugins.TemplateExecutor{plugin},
		workspace:       t.TempDir(),
		outputsDir:      t.TempDir(),
		preparedTasks:   &sync.Map{},
		newDriver: func(context.Context, *wfv1.Artifact, resource.Interface) (artifactcommon.ArtifactDriver, error) {
			return driver, nil
		},
	}
}

func pluginTemplate(inputs, outputs wfv1.Artifacts) wfv1.Template {
	return wfv1.Template{
		Name:    "plugin",
		Plugin:  &wfv1.Plugin{Object: wfv1.Object{Value: json.RawMessage(`{"my-plugin": {}}`)}},
		Inputs:  wfv1.Inputs{Artifacts: inputs},
		Outputs: wfv1.Outputs{Artifacts: outputs},
		ArchiveLocation: &wfv1.ArtifactLocation{S3: &wfv1.S3Artifact{
			S3Bucket: wfv1.S3Bucket{Endpoint: "minio:9000", Bucket: "my-bucket"},
			Key:      testArchiveKey,
		}},
	}
}

func s3Key(key string) wfv1.ArtifactLocation {
	return wfv1.ArtifactLocation{S3: &wfv1.S3Artifact{Key: key}}
}

var archiveNone = &wfv1.ArchiveStrategy{None: &wfv1.NoneStrategy{}}

// untarFile returns the content of the single file in a tarball.
func untarFile(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	gzr, err := file.GetGzipReader(f)
	require.NoError(t, err)
	tr := tar.NewReader(gzr)
	_, err = tr.Next()
	require.NoError(t, err)
	data, err := io.ReadAll(tr)
	require.NoError(t, err)
	return string(data)
}

func TestAgentPluginSavesOutputArtifacts(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	driver := &bucketDriver{dir: t.TempDir()}
	var paths []string
	ae := newArtifactAgent(t, driver, pluginFunc(func(tmpl *wfv1.Template) (executorplugins.ExecuteTemplateReply, error) {
		for _, art := range tmpl.Outputs.Artifacts {
			paths = append(paths, art.Path)
		}
		return writeOutputs(tmpl)
	}))

	result, requeue, err := ae.processTask(ctx, "my-wf-1", pluginTemplate(nil, wfv1.Artifacts{
		{Name: "tarred", Path: "/tmp/tarred.txt"},
		{Name: "raw", Path: "/tmp/raw.txt", Archive: archiveNone},
		{Name: "keyed", Path: "/keyed.txt", Archive: archiveNone, ArtifactLocation: s3Key("elsewhere/keyed.txt")},
	}))
	require.NoError(t, err)
	assert.Zero(t, requeue)
	assert.Equal(t, wfv1.NodeSucceeded, result.Phase)

	// the plugin was told to write into the task's directory
	taskDir := filepath.Join(ae.workspace, "my-wf-1")
	assert.Equal(t, []string{
		filepath.Join(taskDir, "tmp", "tarred.txt"),
		filepath.Join(taskDir, "tmp", "raw.txt"),
		filepath.Join(taskDir, "keyed.txt"),
	}, paths)

	require.NotNil(t, result.Outputs)
	assert.Equal(t, wfv1.Artifacts{
		{Name: "tarred", Path: "/tmp/tarred.txt", ArtifactLocation: s3Key(testArchiveKey + "/tarred.tgz")},
		{Name: "raw", Path: "/tmp/raw.txt", Archive: archiveNone, ArtifactLocation: s3Key(testArchiveKey + "/raw.txt")},
		{Name: "keyed", Path: "/keyed.txt", Archive: archiveNone, ArtifactLocation: s3Key("elsewhere/keyed.txt")},
	}, result.Outputs.Artifacts)
	assert.Equal(t, "tarred", untarFile(t, filepath.Join(driver.dir, testArchiveKey, "tarred.tgz")))
	data, err := os.ReadFile(filepath.Join(driver.dir, testArchiveKey, "raw.txt"))
	require.NoError(t, err)
	assert.Equal(t, "raw", string(data))
	assert.FileExists(t, filepath.Join(driver.dir, "elsewhere", "keyed.txt"))

	// the task is done with
	assert.NoDirExists(t, taskDir)
	assert.NoDirExists(t, filepath.Join(ae.outputsDir, "my-wf-1"))
}

func TestAgentPluginSavesDirectoryOutputArtifact(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	driver := &bucketDriver{dir: t.TempDir()}
	ae := newArtifactAgent(t, driver, pluginFunc(func(tmpl *wfv1.Template) (executorplugins.ExecuteTemplateReply, error) {
		dir := tmpl.Outputs.Artifacts[0].Path
		if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
			return executorplugins.ExecuteTemplateReply{}, err
		}
		if err := os.WriteFile(filepath.Join(dir, "sub", "file.txt"), []byte("in a directory"), 0o644); err != nil {
			return executorplugins.ExecuteTemplateReply{}, err
		}
		return succeeded(nil), nil
	}))

	result, _, err := ae.processTask(ctx, "my-wf-1", pluginTemplate(nil, wfv1.Artifacts{
		{Name: "dir", Path: "/out/dir", Archive: archiveNone},
	}))
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, result.Phase)
	data, err := os.ReadFile(filepath.Join(driver.dir, testArchiveKey, "dir", "sub", "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "in a directory", string(data))
}

func TestAgentPluginLoadsInputArtifacts(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	driver := &bucketDriver{dir: t.TempDir()}
	require.NoError(t, os.MkdirAll(filepath.Join(driver.dir, "inputs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(driver.dir, "inputs", "plain.txt"), []byte("plain"), 0o600))
	// as a container template's output artifact is stored by default
	src := filepath.Join(t.TempDir(), "tarred.txt")
	require.NoError(t, os.WriteFile(src, []byte("tarred"), 0o600))
	var tarball bytes.Buffer
	require.NoError(t, archive.TarGzToWriter(ctx, src, 6, &tarball))
	require.NoError(t, os.WriteFile(filepath.Join(driver.dir, "inputs", "tarred.tgz"), tarball.Bytes(), 0o600))

	var read []string
	ae := newArtifactAgent(t, driver, pluginFunc(func(tmpl *wfv1.Template) (executorplugins.ExecuteTemplateReply, error) {
		for _, art := range tmpl.Inputs.Artifacts {
			if art.Optional {
				assert.NoFileExists(t, art.Path)
				continue
			}
			data, err := os.ReadFile(art.Path)
			if err != nil {
				return executorplugins.ExecuteTemplateReply{}, err
			}
			read = append(read, string(data))
			if runtime.GOOS == "windows" {
				// chmod does not work in windows
				continue
			}
			info, err := os.Stat(art.Path)
			if err != nil {
				return executorplugins.ExecuteTemplateReply{}, err
			}
			// open to plugins running as any user
			assert.Equal(t, os.FileMode(0o666), info.Mode().Perm())
			info, err = os.Stat(filepath.Dir(art.Path))
			if err != nil {
				return executorplugins.ExecuteTemplateReply{}, err
			}
			assert.Equal(t, os.FileMode(0o777), info.Mode().Perm())
		}
		return succeeded(nil), nil
	}))

	result, _, err := ae.processTask(ctx, "my-wf-1", pluginTemplate(wfv1.Artifacts{
		{Name: "plain", Path: "/in/plain.txt", ArtifactLocation: s3Key("inputs/plain.txt")},
		{Name: "tarred", Path: "/in/tarred.txt", ArtifactLocation: s3Key("inputs/tarred.tgz")},
		{Name: "optional", Path: "/in/optional.txt", Optional: true},
	}, nil))
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, result.Phase, result.Message)
	assert.Equal(t, []string{"plain", "tarred"}, read)
	assert.NoDirExists(t, filepath.Join(ae.workspace, "my-wf-1"))
}

func TestAgentPluginRequiredInputArtifactMissing(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	called := false
	ae := newArtifactAgent(t, &bucketDriver{dir: t.TempDir()}, pluginFunc(func(*wfv1.Template) (executorplugins.ExecuteTemplateReply, error) {
		called = true
		return succeeded(nil), nil
	}))

	result, _, err := ae.processTask(ctx, "my-wf-1", pluginTemplate(wfv1.Artifacts{
		{Name: "unsupplied", Path: "/in/unsupplied.txt"},
	}, nil))
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeError, result.Phase)
	assert.Contains(t, result.Message, "required artifact 'unsupplied' not supplied")
	assert.False(t, called, "the plugin is not offered a task whose inputs failed to load")
	assert.NoDirExists(t, filepath.Join(ae.workspace, "my-wf-1"))
}

func TestAgentPluginOutputArtifactMissing(t *testing.T) {
	outputs := wfv1.Artifacts{
		{Name: "required", Path: "/out/required.txt"},
		{Name: "optional", Path: "/out/optional.txt", Optional: true},
	}
	for _, tc := range []struct {
		name          string
		phase         wfv1.NodePhase
		expectPhase   wfv1.NodePhase
		expectMessage string
	}{
		{name: "fails a task that succeeded", phase: wfv1.NodeSucceeded, expectPhase: wfv1.NodeError, expectMessage: "failed to save output artifacts: required artifact 'required' not found at /out/required.txt"},
		{name: "leaves a failed task be", phase: wfv1.NodeFailed, expectPhase: wfv1.NodeFailed, expectMessage: "plugin message"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			driver := &bucketDriver{dir: t.TempDir()}
			ae := newArtifactAgent(t, driver, pluginFunc(func(*wfv1.Template) (executorplugins.ExecuteTemplateReply, error) {
				return executorplugins.ExecuteTemplateReply{Node: &wfv1.NodeResult{Phase: tc.phase, Message: "plugin message"}}, nil
			}))

			result, _, err := ae.processTask(ctx, "my-wf-1", pluginTemplate(nil, outputs))
			require.NoError(t, err)
			assert.Equal(t, tc.expectPhase, result.Phase)
			assert.Contains(t, result.Message, tc.expectMessage)
			assert.Nil(t, result.Outputs)
			assert.Zero(t, driver.saves)
		})
	}
}

func TestAgentPluginFailedTaskSavesOutputArtifacts(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	driver := &bucketDriver{dir: t.TempDir()}
	ae := newArtifactAgent(t, driver, pluginFunc(func(tmpl *wfv1.Template) (executorplugins.ExecuteTemplateReply, error) {
		if _, err := writeOutputs(tmpl); err != nil {
			return executorplugins.ExecuteTemplateReply{}, err
		}
		return executorplugins.ExecuteTemplateReply{Node: &wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: "plugin message"}}, nil
	}))

	result, _, err := ae.processTask(ctx, "my-wf-1", pluginTemplate(nil, wfv1.Artifacts{
		{Name: "debug", Path: "/out/debug.txt", Archive: archiveNone},
	}))
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeFailed, result.Phase)
	assert.Equal(t, "plugin message", result.Message)
	require.NotNil(t, result.Outputs)
	require.Len(t, result.Outputs.Artifacts, 1)
	assert.FileExists(t, filepath.Join(driver.dir, testArchiveKey, "debug.txt"))
}

func TestAgentPluginReportedArtifacts(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	driver := &bucketDriver{dir: t.TempDir()}
	reported := wfv1.Artifacts{
		{Name: "declared", ArtifactLocation: s3Key("stored/by/plugin")},
		{Name: "undeclared", ArtifactLocation: s3Key("referenced/by/plugin")},
	}
	ae := newArtifactAgent(t, driver, pluginFunc(func(*wfv1.Template) (executorplugins.ExecuteTemplateReply, error) {
		return succeeded(&wfv1.Outputs{
			Parameters: []wfv1.Parameter{{Name: "p", Value: wfv1.AnyStringPtr("v")}},
			Artifacts:  reported,
		}), nil
	}))

	result, _, err := ae.processTask(ctx, "my-wf-1", pluginTemplate(nil, wfv1.Artifacts{
		{Name: "declared", Path: "/out/declared.txt"},
	}))
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, result.Phase, result.Message)
	require.NotNil(t, result.Outputs)
	assert.Equal(t, reported, result.Outputs.Artifacts)
	assert.Len(t, result.Outputs.Parameters, 1)
	assert.Zero(t, driver.saves)
}

func TestAgentPluginRequeuedTask(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	driver := &bucketDriver{dir: t.TempDir()}
	require.NoError(t, os.WriteFile(filepath.Join(driver.dir, "input.txt"), []byte("input"), 0o600))
	calls := 0
	ae := newArtifactAgent(t, driver, pluginFunc(func(tmpl *wfv1.Template) (executorplugins.ExecuteTemplateReply, error) {
		calls++
		if calls == 1 {
			return executorplugins.ExecuteTemplateReply{
				Node:    &wfv1.NodeResult{Phase: wfv1.NodeRunning},
				Requeue: &metav1.Duration{Duration: 1},
			}, nil
		}
		return writeOutputs(tmpl)
	}))
	tmpl := pluginTemplate(
		wfv1.Artifacts{{Name: "input", Path: "/input.txt", ArtifactLocation: s3Key("input.txt")}},
		wfv1.Artifacts{{Name: "output", Path: "/output.txt"}},
	)
	taskDir := filepath.Join(ae.workspace, "my-wf-1")

	result, requeue, err := ae.processTask(ctx, "my-wf-1", tmpl)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, result.Phase)
	assert.NotZero(t, requeue)
	assert.FileExists(t, filepath.Join(taskDir, "input.txt"), "a running task keeps its directory")

	result, _, err = ae.processTask(ctx, "my-wf-1", tmpl)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, result.Phase, result.Message)
	assert.Equal(t, 1, driver.loads, "input artifacts are loaded once")
	require.NotNil(t, result.Outputs)
	assert.Len(t, result.Outputs.Artifacts, 1)
	assert.NoDirExists(t, taskDir)
}

func TestAgentPluginArtifactsStayInTaskDirectory(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "secret")
	for _, tc := range []struct {
		name     string
		art      wfv1.Artifact
		plugin   func(t *testing.T, workspace, path string)
		symlinks bool
		expect   string
	}{
		{
			name: "relative path",
			art:  wfv1.Artifact{Name: "out", Path: "../secret", Archive: archiveNone},
			plugin: func(t *testing.T, workspace, _ string) {
				require.NoError(t, os.WriteFile(filepath.Join(workspace, "secret"), []byte("secret"), 0o644))
			},
			expect: "required artifact 'out' not found at ../secret",
		},
		{
			name: "symlink",
			art:  wfv1.Artifact{Name: "out", Path: "/out.txt"},
			plugin: func(t *testing.T, _, path string) {
				require.NoError(t, os.Symlink(outside, path))
			},
			symlinks: true,
			expect:   "required artifact 'out' not found at /out.txt",
		},
		{
			name: "symlink in a directory uploaded with no archive",
			art:  wfv1.Artifact{Name: "out", Path: "/out", Archive: archiveNone},
			plugin: func(t *testing.T, _, path string) {
				require.NoError(t, os.MkdirAll(path, 0o755))
				require.NoError(t, os.Symlink(outside, filepath.Join(path, "link")))
			},
			symlinks: true,
			expect:   "is a symlink",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.symlinks && runtime.GOOS == "windows" {
				t.Skip("creating symlinks needs a privilege in windows, where the agent does not run")
			}
			ctx := logging.TestContext(t.Context())
			require.NoError(t, os.WriteFile(outside, []byte("secret"), 0o600))
			driver := &bucketDriver{dir: t.TempDir()}
			var ae *AgentExecutor
			ae = newArtifactAgent(t, driver, pluginFunc(func(tmpl *wfv1.Template) (executorplugins.ExecuteTemplateReply, error) {
				tc.plugin(t, ae.workspace, tmpl.Outputs.Artifacts[0].Path)
				return succeeded(nil), nil
			}))

			result, _, err := ae.processTask(ctx, "my-wf-1", pluginTemplate(nil, wfv1.Artifacts{tc.art}))
			require.NoError(t, err)
			assert.Equal(t, wfv1.NodeError, result.Phase)
			assert.Contains(t, result.Message, tc.expect)
			assert.Zero(t, driver.saves)
		})
	}
}

func TestAgentPluginArtifactPluginsUnsupported(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	ae := newArtifactAgent(t, &bucketDriver{dir: t.TempDir()}, pluginFunc(writeOutputs))
	tmpl := pluginTemplate(nil, wfv1.Artifacts{{Name: "out", Path: "/out.txt"}})
	tmpl.ArchiveLocation = &wfv1.ArtifactLocation{Plugin: &wfv1.PluginArtifact{Name: "my-artifact-plugin", Key: testArchiveKey}}

	result, _, err := ae.processTask(ctx, "my-wf-1", tmpl)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeError, result.Phase)
	assert.Contains(t, result.Message, "artifact plugin my-artifact-plugin")
}

func TestAgentResources(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	secretDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(secretDir, "mounted"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(secretDir, "mounted", "key"), []byte("from file"), 0o600))
	r := agentResources{
		clientSet: fake.NewClientset(
			&apiv1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "unmounted", Namespace: "my-ns"}, Data: map[string][]byte{"key": []byte("from api")}},
			&apiv1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "my-cm", Namespace: "my-ns"}, Data: map[string]string{"key": "value"}},
		),
		namespace: "my-ns",
		secretDir: secretDir,
	}

	val, err := r.GetSecret(ctx, "mounted", "key")
	require.NoError(t, err)
	assert.Equal(t, "from file", val)

	val, err = r.GetSecret(ctx, "unmounted", "key")
	require.NoError(t, err)
	assert.Equal(t, "from api", val)

	_, err = r.GetSecret(ctx, "missing", "key")
	require.ErrorContains(t, err, `secret "missing" is not mounted in the agent pod`)

	_, err = r.GetSecret(ctx, "..", "mounted/key")
	require.ErrorContains(t, err, "invalid secret name")

	val, err = r.GetConfigMapKey(ctx, "my-cm", "key")
	require.NoError(t, err)
	assert.Equal(t, "value", val)
}
