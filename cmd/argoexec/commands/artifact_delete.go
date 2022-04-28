package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/spf13/cobra"
)

func NewArtifactDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "delete",
		SilenceUsage: true,

		RunE: func(cmd *cobra.Command, args []string) error {
			a := &wfv1.Artifact{}
			if err := json.Unmarshal([]byte(os.Getenv(common.EnvVarArtifact)), a); err != nil {
				return fmt.Errorf("failed to unmarshal artifact: %w", err)
			}

			drv, err := artifacts.NewDriver(cmd.Context(), a, &x{})
			if err != nil {
				return fmt.Errorf("failed to create driver: %w", err)
			}

			if err := drv.Delete(a); err != nil {
				return fmt.Errorf("failed to delete artifact: %w", err)
			}
			return nil
		},
	}
}

type x struct{}

func (x x) GetSecret(ctx context.Context, name, key string) (string, error) {
	file, err := os.ReadFile(filepath.Join(common.SecretVolMountPath, name, key))
	return string(file), err

}

func (x x) GetConfigMapKey(ctx context.Context, name, key string) (string, error) {
	panic("implement me")
}
