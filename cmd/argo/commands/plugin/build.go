package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

func addHeader(x []byte, h string) []byte {
	return []byte(fmt.Sprintf("%s\n%s", h, string(x)))
}

func addCodegenHeader(x []byte) []byte {
	return addHeader(x, "# This is an auto-generated file. DO NOT EDIT")
}

func NewBuildCommand() *cobra.Command {
	return &cobra.Command{
		Use:           "build DIR",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			pluginDir := args[0]
			defn, err := loadPlugin(pluginDir)
			if err != nil {
				return err
			}
			data, err := yaml.Marshal(defn.configMap)
			if err != nil {
				return err
			}
			name := defn.plugin.Metadata.Name
			kind := defn.plugin.Kind
			cmPath := filepath.Join(pluginDir, fmt.Sprintf("%s-%s-plugin-configmap.yaml", name, kind.short()))
			err = os.WriteFile(cmPath, addCodegenHeader(data), 0666)
			if err != nil {
				return err
			}
			fmt.Printf("%s created\n", cmPath)
			if defn.controllerPatch != nil {
				data, err := yaml.Marshal(defn.controllerPatch)
				if err != nil {
					return err
				}
				patchName := fmt.Sprintf("%s-%s-plugin-deployment-patch.yaml", name, kind.short())
				patchPath := filepath.Join(pluginDir, patchName)
				header := fmt.Sprintf("# This is a Kustomize patch that will add the plugin to your controller.\n# Example: kubectl -n argo patch deployment workflow-controller --patch-file %s", patchName)
				if err := os.WriteFile(patchPath, addCodegenHeader(addHeader(data, header)), 0666); err != nil {
					return err
				}
				fmt.Printf("%s created\n", patchPath)
			}
			return nil
		},
	}
}
