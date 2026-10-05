//go:build !windows

package commands

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/maindriver"
)

type mainDriverFunc func(context.Context, maindriver.Task, maindriver.ResultSink) (int, error)

func (f mainDriverFunc) Run(ctx context.Context, task maindriver.Task, sink maindriver.ResultSink) (int, error) {
	return f(ctx, task, sink)
}

func TestEmissary_OutputsSurviveRetryStartupFailure(t *testing.T) {
	for _, scenario := range []string{"cancelled", "missing-executable"} {
		t.Run(scenario, func(t *testing.T) {
			useTempVarRunArgo(t)
			t.Setenv(common.EnvVarWaitForReady, "false")
			dir := t.TempDir()
			param := filepath.Join(dir, "param")
			artifact := filepath.Join(dir, "artifact")
			executable := filepath.Join(dir, "command")
			require.NoError(t, os.WriteFile(executable, []byte("#!/bin/sh\nprintf parameter > \"$1\"\nprintf artifact > \"$2\"\necho attempt-output\nexit 7\n"), 0o755))
			command := []string{executable, param, artifact}
			tmpl := wfv1.Template{
				ContainerSet: &wfv1.ContainerSetTemplate{
					Containers:    []wfv1.ContainerNode{{Container: corev1.Container{Name: "main", Image: "alpine", Command: command}}},
					RetryStrategy: &wfv1.ContainerSetRetryStrategy{Retries: new(intstr.FromInt(2))},
				},
				Outputs: wfv1.Outputs{
					Parameters: []wfv1.Parameter{{Name: "p", ValueFrom: &wfv1.ValueFrom{Path: param}}},
					Artifacts:  []wfv1.Artifact{{Name: "a", Path: artifact}},
				},
			}
			body, err := json.Marshal(tmpl)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(varRunArgo, "template"), body, 0o600))
			ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
			defer cancel()
			attempts := 0
			driver := mainDriverFunc(func(ctx context.Context, task maindriver.Task, sink maindriver.ResultSink) (int, error) {
				attempts++
				code, runErr := (maindriver.Container{}).Run(ctx, task, sink)
				if attempts == 1 {
					require.Equal(t, 7, code)
					require.NoFileExists(t, varRunArgo+"/outputs/parameters/"+param, "stage only after retries")
					// Make the next attempt fail before it starts, after the
					// first command has finished producing its outputs.
					if scenario == "cancelled" {
						cancel()
					} else {
						require.NoError(t, os.Remove(executable))
					}
				}
				return code, runErr
			})

			err = runEmissary(ctx, "main", newPodSource("main", true, command), driver)
			require.Error(t, err)
			if scenario == "cancelled" {
				require.ErrorIs(t, err, context.Canceled)
			} else {
				require.ErrorIs(t, err, os.ErrNotExist)
			}
			require.Equal(t, 2, attempts)
			data, err := os.ReadFile(varRunArgo + "/outputs/parameters/" + param)
			require.NoError(t, err)
			assert.Equal(t, "parameter", string(data))
			data, err = os.ReadFile(filepath.Join(varRunArgo, "ctr", "main", "stdout"))
			require.NoError(t, err)
			assert.Equal(t, "attempt-output\n", string(data))
			archive, err := os.Open(filepath.Join(varRunArgo, "outputs", "artifacts", artifact+".tgz"))
			require.NoError(t, err)
			defer archive.Close()
			gz, err := gzip.NewReader(archive)
			require.NoError(t, err)
			defer gz.Close()
			tr := tar.NewReader(gz)
			_, err = tr.Next()
			require.NoError(t, err)
			data, err = io.ReadAll(tr)
			require.NoError(t, err)
			assert.Equal(t, "artifact", string(data))
		})
	}
}
