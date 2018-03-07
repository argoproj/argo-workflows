package commands

import (
	"github.com/argoproj/argo/install"
	"github.com/argoproj/argo/workflow/common"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// These values may be overridden by the link flags during build
	// (e.g. imageTag will use the official release tag on tagged builds)
	imageNamespace = "argoproj"
	imageTag       = "latest"

	// These are the default image names which `argo install` uses during install
	DefaultControllerImage = imageNamespace + "/workflow-controller:" + imageTag
	DefaultExecutorImage   = imageNamespace + "/argoexec:" + imageTag
	DefaultUiImage         = imageNamespace + "/argoui:" + imageTag
)

func NewInstallCommand() *cobra.Command {
	var (
		installArgs install.InstallOptions
	)
	var command = &cobra.Command{
		Use:   "install",
		Short: "install Argo",
		Run: func(cmd *cobra.Command, args []string) {
			_ = initKubeClient()
			installer, err := install.NewInstaller(restConfig, installArgs)
			if err != nil {
				log.Fatal(err)
			}
			installer.Install()
		},
	}
	command.Flags().BoolVar(&installArgs.Upgrade, "upgrade", false, "upgrade controller/ui deployments and configmap if already installed")
	command.Flags().BoolVar(&installArgs.DryRun, "dry-run", false, "print the kubernetes manifests to stdout instead of installing")
	command.Flags().StringVar(&installArgs.Namespace, "install-namespace", common.DefaultControllerNamespace, "install into a specific Namespace")
	command.Flags().StringVar(&installArgs.InstanceID, "instanceid", "", "optional instance id to use for the controller (for multi-controller environments)")
	command.Flags().StringVar(&installArgs.ConfigMap, "configmap", common.DefaultConfigMapName(common.DefaultControllerDeploymentName), "install controller using preconfigured configmap")
	command.Flags().StringVar(&installArgs.ControllerImage, "controller-image", DefaultControllerImage, "use a specified controller image")
	command.Flags().StringVar(&installArgs.ServiceAccount, "service-account", "", "use a specified service account for the workflow-controller deployment")
	command.Flags().StringVar(&installArgs.ExecutorImage, "executor-image", DefaultExecutorImage, "use a specified executor image")
	command.Flags().StringVar(&installArgs.UIImage, "ui-image", DefaultUiImage, "use a specified ui image")
	command.Flags().StringVar(&installArgs.UIBaseHref, "ui-base-href", "/", "UI base url")
	command.Flags().StringVar(&installArgs.UIServiceAccount, "ui-service-account", "", "use a specified service account for the argo-ui deployment")
	command.Flags().BoolVar(&installArgs.EnableWebConsole, "enable-web-console", false, "allows exec access into running step container using Argo UI")
	command.Flags().StringVar(&installArgs.ImagePullPolicy, "image-pull-policy", "", "imagePullPolicy to use for deployments")
	return command
}
