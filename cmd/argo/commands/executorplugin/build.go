package executorplugin

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	plugin "github.com/argoproj/argo-workflows/v3/workflow/util/plugins"
)

func NewBuildCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "build DIR",
		Short:        "build an executor plugin",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			pluginDir := args[0]
			plug, err := loadPluginManifest(pluginDir)
			if err != nil {
				return err
			}
			cm, err := plugin.ToConfigMap(plug)
			if err != nil {
				return err
			}
			cmPath, err := saveConfigMap(cm, pluginDir)
			if err != nil {
				return err
			}
			fmt.Printf("%s created\n", cmPath)
			readmePath, err := saveReadme(pluginDir, plug)
			if err != nil {
				return err
			}
			fmt.Printf("%s created\n", readmePath)
			return nil
		},
	}
}
