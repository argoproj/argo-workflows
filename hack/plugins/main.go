package main

import (
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

type plugin struct {
	Kind string `json:"kind"`
	Spec struct {
		Description string          `json:"description"`
		Address     string          `json:"address"`
		Container   apiv1.Container `json:"container"`
	} `json:"spec"`
}

func (p plugin) ShortKind() interface{} {
	switch p.Kind {
	case "ExecutorPlugin":
		return "executor"
	default:
		return "controller"
	}
}

func marshalYAML(v interface{}) []byte {
	data, err := yaml.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

func addHeader(x []byte, h string) []byte {
	return []byte(fmt.Sprintf("%s\n%s", h, string(x)))
}

func addCodegenHeader(x []byte) []byte {
	return addHeader(x, "# This is an auto-generated file. DO NOT EDIT")
}

func main() {
	rootDir := os.Args[1]
	dir, err := os.ReadDir(rootDir)
	if err != nil {
		panic(err)
	}
	for _, f := range dir {
		if !f.IsDir() {
			continue
		}
		log.Printf("creating plugin %s...\n", f.Name())
		pluginDir := filepath.Join(rootDir, f.Name())
		manifest, err := os.ReadFile(filepath.Join(pluginDir, fmt.Sprintf("%s-plugin.yaml", f.Name())))
		if err != nil {
			panic(err)
		}
		plug := &plugin{}
		err = yaml.UnmarshalStrict(manifest, plug)
		if err != nil {
			panic(err)
		}
		files, err := filepath.Glob(filepath.Join(pluginDir, "server.*"))
		if err != nil {
			panic(err)
		}
		if len(files) < 1 {
			panic(fmt.Sprintf("plugin %s is missing a server.* file", f.Name()))
		}
		code, err := os.ReadFile(files[0])
		if err != nil {
			panic(err)
		}
		plug.Spec.Container.Args = []string{string(code)}

		if strings.Contains(rootDir, "controller") {
			// Match default security settings for easier patch.
			runAsNonRoot := true
			runAsUser := int64(1000)
			plug.Spec.Container.SecurityContext = &apiv1.SecurityContext{RunAsNonRoot: &runAsNonRoot, RunAsUser: &runAsUser}
		}

		cm := apiv1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s-%s-plugin", f.Name(), plug.ShortKind()),
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
			cm.Data["container"] = string(marshalYAML(plug.Spec.Container))
		}
		data, err := yaml.Marshal(cm)
		if err != nil {
			panic(err)
		}
		cmPath := filepath.Join(pluginDir, fmt.Sprintf("%s-%s-plugin-configmap.yaml", f.Name(), plug.ShortKind()))
		log.Printf("- %s\n", cmPath)
		err = os.WriteFile(cmPath, addCodegenHeader(data), 0666)
		if err != nil {
			panic(err)
		}
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
					Selector: &metav1.LabelSelector{},  // Selector can't be null for a valid patch.
					Template: apiv1.PodTemplateSpec{
						Spec: apiv1.PodSpec{
							Containers: []apiv1.Container{
								{
									Name:                     "workflow-controller",
									Env:                      []apiv1.EnvVar{
										{
											Name:      "ARGO_PLUGINS",
											Value:     "true",
										},
									},
								},
								plug.Spec.Container,
							},
						},
					},
				},
			})
			if err != nil {
				panic(err)
			}
			patchPath := filepath.Join(pluginDir, fmt.Sprintf("%s-%s-plugin-deployment-patch.yaml", f.Name(), plug.ShortKind()))
			log.Printf("- %s\n", patchPath)
			header := fmt.Sprintf("# This is a Kustomize patch that will add the plugin to your controller.\n# Example: kubectl patch -n argo deployment workflow-controller --patch-file %s", patchPath)
			err = os.WriteFile(patchPath, addCodegenHeader(addHeader(data, header)), 0666)
			if err != nil {
				panic(err)
			}
		}
	}
}
