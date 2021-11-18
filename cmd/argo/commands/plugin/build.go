package plugin

import (
	"fmt"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
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
		Use: "build DIR",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			pluginDir := args[0]
			manifest, err := os.ReadFile(filepath.Join(pluginDir, "plugin.yaml"))
			if err != nil {
				return err
			}
			plug := &plugin{}
			err = yaml.UnmarshalStrict(manifest, plug)
			if err != nil {
				return err
			}
			name := plug.Metadata.Name
			files, err := filepath.Glob(filepath.Join(pluginDir, "server.*"))
			if err != nil {
				return err
			}
			if len(files) < 1 {
				panic(fmt.Sprintf("plugin %s is missing a server.* file", name))
			}
			code, err := os.ReadFile(files[0])
			if err != nil {
				return err
			}
			plug.Spec.Container.Args = []string{string(code)}

			// Match default security settings for easier patch.
			runAsNonRoot := true
			runAsUser := int64(1000)
			plug.Spec.Container.SecurityContext = &apiv1.SecurityContext{RunAsNonRoot: &runAsNonRoot, RunAsUser: &runAsUser}

			cm := apiv1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("%s-%s-plugin", name, plug.ShortKind()),
					Labels: map[string]string{
						"workflows.argoproj.io/configmap-type": plug.Kind,
					},
					Annotations: map[string]string{
						"workflows.argoproj.io/description": plug.Spec.Description,
						"workflows.argoproj.io/version":     ">= v3.3",
					},
				},
				Data: map[string]string{
					"address": plug.Spec.Address,
				},
			}
			switch plug.Kind {
			case "ExecutorPlugin":
				data, err := yaml.Marshal(plug.Spec.Container)
				if err != nil {
					return err
				}
				cm.Data["container"] = string(data)
			}
			data, err := yaml.Marshal(cm)
			if err != nil {
				return err
			}
			cmPath := filepath.Join(pluginDir, fmt.Sprintf("%s-%s-plugin-configmap.yaml", name, plug.ShortKind()))
			err = os.WriteFile(cmPath, addCodegenHeader(data), 0666)
			if err != nil {
				return err
			}
			fmt.Printf("created %s\n", cmPath)
			switch plug.Kind {
			case "ControllerPlugin":
				data, err := yaml.Marshal(appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "workflow-controller",
					},
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{}, // Selector can't be null for a valid patch.
						Template: apiv1.PodTemplateSpec{
							Spec: apiv1.PodSpec{
								Containers: []apiv1.Container{plug.Spec.Container},
							},
						},
					},
				})
				if err != nil {
					return err
				}
				patchPath := filepath.Join(pluginDir, fmt.Sprintf("%s-%s-plugin-deployment-patch.yaml", name, plug.ShortKind()))
				header := fmt.Sprintf("# This is a Kustomize patch that will add the plugin to your controller.\n# Example: kubectl patch -n argo deployment workflow-controller --patch-file %s", patchPath)
				if err := os.WriteFile(patchPath, addCodegenHeader(addHeader(data, header)), 0666); err != nil {
					return err
				}
				fmt.Printf("created %s\n", patchPath)
			}
			return nil
		},
	}
}
