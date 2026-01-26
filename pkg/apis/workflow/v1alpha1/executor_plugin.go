package v1alpha1

import execplugin "github.com/argoproj/argo-workflows/v3/pkg/plugins/spec"

type ExecutorPlugin struct {
	Spec execplugin.PluginSpec `json:"spec"`
}
