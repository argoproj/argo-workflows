package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

type plugin struct {
	Kind     pluginKind        `json:"kind"`
	Metadata metav1.ObjectMeta `json:"metadata"`
	Spec     struct {
		Description string          `json:"description"`
		Address     string          `json:"address"`
		Container   apiv1.Container `json:"container"`
	} `json:"spec"`
}

type pluginKind string

func (k pluginKind) short() interface{} {
	switch k {
	case "ExecutorPlugin":
		return "executor"
	default:
		return "controller"
	}
}

type loadResult struct {
	plugin          *plugin
	configMap       *apiv1.ConfigMap
	controllerPatch *appsv1.Deployment
}

func loadPlugin(pluginDir string) (*loadResult, error) {
	manifest, err := os.ReadFile(filepath.Join(pluginDir, "plugin.yaml"))
	if err != nil {
		return nil, err
	}
	plug := &plugin{}
	err = yaml.UnmarshalStrict(manifest, plug)
	if err != nil {
		return nil, err
	}
	name := plug.Metadata.Name
	files, err := filepath.Glob(filepath.Join(pluginDir, "server.*"))
	if err != nil {
		return nil, err
	}
	if len(files) < 1 {
		panic(fmt.Sprintf("plugin %s is missing a server.* file", name))
	}
	code, err := os.ReadFile(files[0])
	if err != nil {
		return nil, err
	}
	plug.Spec.Container.Args = []string{string(code)}

	// Match default security settings for easier patch.
	runAsNonRoot := true
	runAsUser := int64(1000)
	plug.Spec.Container.SecurityContext = &apiv1.SecurityContext{RunAsNonRoot: &runAsNonRoot, RunAsUser: &runAsUser}
	cm := &apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s-plugin", name, plug.Kind.short()),
			Labels: map[string]string{
				common.LabelKeyConfigMapType: string(plug.Kind),
			},
			Annotations: map[string]string{
				common.AnnotationKeyPluginName:  plug.Metadata.Name,
				common.AnnotationKeyDescription: plug.Spec.Description,
				common.AnnotationKeyVersion:     ">= v3.3",
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
			return nil, err
		}
		cm.Data["container"] = string(data)
	}
	var controllerPatch *appsv1.Deployment
	switch plug.Kind {
	case "ControllerPlugin":
		controllerPatch = &appsv1.Deployment{
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
		}
	}
	return &loadResult{
		plugin:          plug,
		configMap:       cm,
		controllerPatch: controllerPatch,
	}, nil
}
