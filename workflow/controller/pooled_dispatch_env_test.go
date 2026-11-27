package controller

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// pooledTaskEnv classifies every ARGO_* variable a task's pod gives its
// containers by how a pooled task gets it instead, so none is missing:
// as a constant of its pool, from a spec.dispatch field, or from the TaskSet.
var pooledTaskEnv = map[string]pooledEnvSource{
	common.EnvVarPodName:                       poolEnv("the worker pod's own, through the downward API"),
	common.EnvVarPodUID:                        poolEnv("the worker pod's own, through the downward API"),
	common.EnvVarContainerName:                 poolEnv("the pool pod's container names"),
	common.EnvVarInstanceID:                    poolEnv("spec.pool instanceID"),
	common.EnvVarInitlessPod:                   poolEnv("spec.pool initlessPod"),
	common.EnvVarWaitForReady:                  poolEnv("the pod layout, from spec.pool initlessPod"),
	common.EnvVarTerminationGracePeriodSeconds: poolEnv("spec.pool terminationGracePeriodSeconds"),
	common.EnvVarWorkflowName:                  taskSetEnv("its workflow label"),
	common.EnvVarWorkflowUID:                   taskSetEnv("its owner reference"),
	common.EnvVarNodeID:                        taskSetEnv("its spec.tasks key"),
	common.EnvVarTemplate:                      taskSetEnv("its spec.tasks template"),
	common.EnvVarContainerArgsFile:             taskSetEnv("its spec.tasks template carries the args, however long"),
	common.EnvVarProgressFile:                  taskSetEnv("a fixed path, used when spec.dispatch has both progress ticks"),
	common.EnvVarDeadline:                      dispatchEnv("Deadline"),
	common.EnvVarIncludeScriptOutput:           dispatchEnv("IncludeScriptOutput"),
	common.EnvVarProgressPatchTickDuration:     dispatchEnv("ProgressPatchTickDuration"),
	common.EnvVarProgressFileTickDuration:      dispatchEnv("ProgressFileTickDuration"),
	common.EnvVarArtifactPluginNames:           poolEnv("none: it names the plugin sidecars to kill once a pod's task ends, and a pool pod's sidecars, from spec.pool artifactPlugins, outlive its tasks"),
	common.EnvVarInputArtifactPluginNames:      dispatchEnv("InputArtifactPlugins"),
}

type pooledEnvSource struct {
	// dispatchField is the spec.dispatch field it comes from, if any.
	dispatchField string
	why           string
}

func poolEnv(why string) pooledEnvSource    { return pooledEnvSource{why: why} }
func taskSetEnv(why string) pooledEnvSource { return pooledEnvSource{why: why} }
func dispatchEnv(field string) pooledEnvSource {
	return pooledEnvSource{dispatchField: field, why: "spec.dispatch"}
}

// TestPooledTaskEnvClassified checks that every ARGO_* variable the pods built
// for a corpus of templates and configurations give any container is
// classified in pooledTaskEnv, and the spec.dispatch fields named exist.
func TestPooledTaskEnvClassified(t *testing.T) {
	dispatchType := reflect.TypeFor[wfv1.TaskDispatch]()
	for name, source := range pooledTaskEnv {
		if source.dispatchField != "" {
			_, ok := dispatchType.FieldByName(source.dispatchField)
			assert.True(t, ok, "%s: spec.dispatch has no field %s", name, source.dispatchField)
		}
	}

	plugin := func(name string) wfv1.ArtifactLocation {
		return wfv1.ArtifactLocation{Plugin: &wfv1.PluginArtifact{Name: wfv1.ArtifactPluginName(name), Configuration: "{}", Key: "k"}}
	}
	withArtifacts := func(tmpl *wfv1.Template) *wfv1.Template {
		tmpl.Inputs.Artifacts = []wfv1.Artifact{{Name: "in", Path: "/in", ArtifactLocation: plugin("in-plugin")}}
		tmpl.Outputs.Artifacts = []wfv1.Artifact{{Name: "out", Path: "/out", ArtifactLocation: plugin("out-plugin")}}
		return tmpl
	}
	container := func() *wfv1.Template {
		return &wfv1.Template{Name: "t", Container: &apiv1.Container{Image: "my-image", Args: []string{strings.Repeat("x", common.MaxEnvVarLen)}}}
	}
	script := func() *wfv1.Template {
		return &wfv1.Template{Name: "t", Script: &wfv1.ScriptTemplate{Container: apiv1.Container{Image: "my-image", Command: []string{"sh"}}, Source: strings.Repeat("x", common.MaxEnvVarLen)}}
	}
	seen := map[string]bool{}
	for _, initless := range []bool{false, true} {
		for _, tmpl := range []*wfv1.Template{container(), withArtifacts(container()), script(), withArtifacts(script())} {
			tmpl.PodSpecPatch = `{"terminationGracePeriodSeconds": 5}`
			ctx := logging.TestContext(t.Context())
			cancel, controller := newController(ctx, func(wfc *WorkflowController) {
				wfc.Config.InstanceID = "instance"
				wfc.Config.InitlessPod = &config.InitlessPodConfig{Enabled: initless}
				wfc.Config.ArtifactDrivers = []config.ArtifactDriver{{Name: "in-plugin", Image: "plugin-image"}, {Name: "out-plugin", Image: "plugin-image"}}
				wfc.Config.Images["plugin-image"] = config.Image{Entrypoint: []string{"/plugin"}}
			})
			wf := &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{Name: "env", Namespace: "default", UID: "wf-uid"},
				Spec:       wfv1.WorkflowSpec{Entrypoint: "t", Templates: []wfv1.Template{*tmpl}},
			}
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			mainCtr := tmpl.Container
			if mainCtr == nil {
				mainCtr = &tmpl.Script.Container
			}
			pod, err := woc.createWorkflowPod(ctx, "env", []apiv1.Container{*mainCtr}, tmpl, &createWorkflowPodOpts{includeScriptOutput: true, executionDeadline: time.Now().Add(time.Hour)})
			require.NoError(t, err)
			require.NotNil(t, pod)
			for _, c := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
				for _, env := range c.Env {
					if strings.HasPrefix(env.Name, "ARGO_") {
						seen[env.Name] = true
					}
				}
			}
			cancel()
		}
	}
	for name := range seen {
		_, ok := pooledTaskEnv[name]
		assert.True(t, ok, "a task's pod gets %s: classify it in pooledTaskEnv (a pool constant, a spec.dispatch field, or from the TaskSet)", name)
	}
	// The corpus must keep reaching what it was built to reach.
	for _, name := range []string{common.EnvVarArtifactPluginNames, common.EnvVarInputArtifactPluginNames, common.EnvVarContainerArgsFile, common.EnvVarTerminationGracePeriodSeconds, common.EnvVarInitlessPod, common.EnvVarProgressFile} {
		assert.True(t, seen[name], "the corpus no longer gives a pod %s", name)
	}
}
