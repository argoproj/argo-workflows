//go:build !windows

package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/maindriver"
)

func TestEmissary_DebugPauseAfter(t *testing.T) {
	for _, scenario := range []string{"released", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			useTempVarRunArgo(t)
			t.Setenv(common.EnvVarWaitForReady, "false")
			t.Setenv("ARGO_DEBUG_PAUSE_AFTER", "true")
			require.NoError(t, os.WriteFile(filepath.Join(varRunArgo, "template"), []byte(`{}`), 0o600))
			ctx := logging.TestContext(t.Context())
			runCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			command := []string{"sh", "-c", "exit 0"}
			driver := mainDriverFunc(func(ctx context.Context, task maindriver.Task, sink maindriver.ResultSink) (int, error) {
				code, err := (maindriver.Container{}).Run(ctx, task, sink)
				require.NoError(t, err)
				require.Zero(t, code)
				// Interrupt only after the real command succeeds. Then the error
				// comes from the debug pause, not from command startup.
				if scenario == "cancelled" {
					cancel()
				} else {
					require.NoError(t, os.WriteFile(filepath.Join(varRunArgo, "ctr", "producer", "after"), nil, 0o600))
				}
				return code, err
			})

			err := runEmissary(runCtx, "producer", newPodSource("producer", false, command), driver)
			marker, readErr := os.ReadFile(filepath.Join(varRunArgo, "ctr", "producer", "exitcode"))
			require.NoError(t, readErr)
			// When only the producer gets an interrupt, the context of a sibling
			// stays active. Use that context to check the real dependency gate.
			dependencyErr := waitForDependency(ctx, "producer")
			if scenario == "cancelled" {
				require.ErrorIs(t, err, context.Canceled)
				require.ErrorContains(t, err, "failed waiting for debug-pause-after marker")
				assert.Equal(t, "64", string(marker))
				assert.ErrorContains(t, dependencyErr, `dependency "producer" exited with non-zero code: 64`)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "0", string(marker))
				require.NoError(t, dependencyErr)
			}
		})
	}
}
