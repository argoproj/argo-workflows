package artifact

import (
	"context"
	//"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	//wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	//executor "github.com/argoproj/argo-workflows/v3/workflow/artifacts"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func NewArtifactDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "delete",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// to be implemented by Dillen Padhiar
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
	file, err := os.ReadFile(filepath.Join(common.SecretVolMountPath, name, key))
	return string(file), err

}

func (r resources) GetConfigMapKey(ctx context.Context, name, key string) (string, error) {
	return "", fmt.Errorf("not supported")
}
