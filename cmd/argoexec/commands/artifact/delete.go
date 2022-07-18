package artifact

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func NewArtifactDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "delete",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			// get environment variables for:
			//		- WorkflowTaskSet name
			//		- ArtifactGC Strategy
			// set up a pool of workers
			// read WorkflowTaskSet:
			//   for each Template:
			// 	   for each Output Artifact:
			//		 if it matches ArtifactGC Strategy (determine from both Artifact and WorkflowSpec):
			//			pass it on the goroutine that all the workers read from
			//			each worker will send the results on a channel which is receiving the results
			//			to populate an in-memory WorkflowTaskResult
			// wait for the goroutines to finish, then write the WorkflowTaskResult

			/*a := &wfv1.Artifact{}
			if err := json.Unmarshal([]byte(os.Getenv(common.EnvVarArtifact)), a); err != nil {
				return fmt.Errorf("failed to unmarshal artifact: %w", err)
			}

			drv, err := executor.NewDriver(cmd.Context(), a, &resources{})
			if err != nil {
				return fmt.Errorf("failed to create driver: %w", err)
			}

			if err := drv.Delete(a); err != nil {
				return fmt.Errorf("failed to delete artifact: %w", err)
			}*/
			return nil
		},
	}
}

type resources struct{}

func (r resources) GetSecret(ctx context.Context, name, key string) (string, error) {
	// create a cache here (sync.Map)
	file, err := os.ReadFile(filepath.Join(common.SecretVolMountPath, name, key))
	return string(file), err

}

func (r resources) GetConfigMapKey(ctx context.Context, name, key string) (string, error) {
	return "", fmt.Errorf("not supported")
}
