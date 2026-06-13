package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func Test_teeContainerLogs(t *testing.T) {
	t.Run("creates combined file", func(t *testing.T) {
		for _, containerName := range []string{common.InitContainerName, common.WaitContainerName} {
			t.Run(containerName, func(t *testing.T) {
				varRunArgo := t.TempDir()
				ctx := logging.TestContext(t.Context())

				_, closer, err := teeContainerLogs(ctx, varRunArgo, containerName)
				require.NoError(t, err)
				defer closer()

				combinedPath := filepath.Join(varRunArgo, "ctr", containerName, "combined")
				_, err = os.Stat(combinedPath)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("writes to combined file", func(t *testing.T) {
		varRunArgo := t.TempDir()
		ctx := logging.TestContext(t.Context())

		newCtx, closer, err := teeContainerLogs(ctx, varRunArgo, common.InitContainerName)
		require.NoError(t, err)
		defer closer()

		logging.RequireLoggerFromContext(newCtx).Info(newCtx, "test log message")
		closer()

		combinedPath := filepath.Join(varRunArgo, "ctr", common.InitContainerName, "combined")
		content, err := os.ReadFile(combinedPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "test log message")
	})
}
