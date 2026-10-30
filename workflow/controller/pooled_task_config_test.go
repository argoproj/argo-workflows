package controller

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	argoexec "github.com/argoproj/argo-workflows/v4/cmd/argoexec/executor"
	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/pooled"
)

// TestPooledTaskConfigMatchesPod builds a pod and a pooled task for each of a
// corpus of templates, then the executor's TaskConfig from each: from the env
// of the init-less pod's supervisor, which prepares and collects as a pool's
// worker does, and from the TaskSet as a worker claims it. They must be the
// same, apart from the resolved command, which a pod's main gets in its
// container spec, the span their trace continues, and the emptyDir volumes a
// pool pod's worker empties between tasks, which a pod's go with it; and
// main's per-task env must be what the pod gives main.
func TestPooledTaskConfigMatchesPod(t *testing.T) {
	plugin := func(name string) wfv1.ArtifactLocation {
		return wfv1.ArtifactLocation{Plugin: &wfv1.PluginArtifact{Name: wfv1.ArtifactPluginName(name), Configuration: "{}", Key: "k"}}
	}
	withArtifacts := func(tmpl *wfv1.Template) *wfv1.Template {
		tmpl.Inputs.Artifacts = []wfv1.Artifact{{Name: "in", Path: "/in", ArtifactLocation: plugin("in-plugin")}}
		tmpl.Outputs.Artifacts = []wfv1.Artifact{{Name: "out", Path: "/out", ArtifactLocation: plugin("out-plugin")}}
		return tmpl
	}
	withParameters := func(tmpl *wfv1.Template) *wfv1.Template {
		tmpl.Inputs.Parameters = []wfv1.Parameter{{Name: "p", Value: wfv1.AnyStringPtr("v")}}
		tmpl.Outputs.Parameters = []wfv1.Parameter{{Name: "o", ValueFrom: &wfv1.ValueFrom{Path: "/o"}}}
		return tmpl
	}
	container := func() *wfv1.Template {
		return &wfv1.Template{Name: "t", Container: &apiv1.Container{Image: "my-image", Command: []string{"sh", "-c"}, Args: []string{"echo hi"}, Env: []apiv1.EnvVar{{Name: "X", Value: "x"}}}}
	}
	entrypoint := func() *wfv1.Template {
		return &wfv1.Template{Name: "t", Container: &apiv1.Container{Image: "my-image"}}
	}
	withVolumes := func(tmpl *wfv1.Template) *wfv1.Template {
		tmpl.Volumes = []apiv1.Volume{
			{Name: "scratch", VolumeSource: apiv1.VolumeSource{EmptyDir: &apiv1.EmptyDirVolumeSource{}}},
			{Name: "cache", VolumeSource: apiv1.VolumeSource{HostPath: &apiv1.HostPathVolumeSource{Path: "/cache"}}},
		}
		tmpl.Container.VolumeMounts = []apiv1.VolumeMount{{Name: "scratch", MountPath: "/scratch"}, {Name: "cache", MountPath: "/cache"}}
		return tmpl
	}
	script := func() *wfv1.Template {
		return &wfv1.Template{Name: "t", Script: &wfv1.ScriptTemplate{Container: apiv1.Container{Image: "my-image", Command: []string{"sh"}}, Source: "echo hi"}}
	}
	for name, tmpl := range map[string]*wfv1.Template{
		"container":           container(),
		"entrypoint":          entrypoint(),
		"script":              script(),
		"plugin artifacts":    withArtifacts(container()),
		"volumes":             withVolumes(container()),
		"parameters":          withParameters(script()),
		"entrypoint with env": withParameters(withArtifacts(entrypoint())),
	} {
		t.Run(name, func(t *testing.T) {
			// A remote parent, so both carry the same trace context.
			ctx := trace.ContextWithRemoteSpanContext(logging.TestContext(t.Context()), trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    trace.TraceID{1},
				SpanID:     trace.SpanID{2},
				TraceFlags: trace.FlagsSampled,
			}))
			cancel, controller := newController(ctx, func(wfc *WorkflowController) {
				wfc.Config.InstanceID = "instance"
				wfc.Config.InitlessPod = &config.InitlessPodConfig{Enabled: true}
				wfc.Config.ArtifactDrivers = []config.ArtifactDriver{{Name: "in-plugin", Image: "plugin-image"}, {Name: "out-plugin", Image: "plugin-image"}}
				wfc.Config.Images["plugin-image"] = config.Image{Entrypoint: []string{"/plugin"}}
			})
			defer cancel()
			wf := &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{Name: "env", Namespace: "default", UID: "wf-uid"},
				Spec:       wfv1.WorkflowSpec{Entrypoint: "t", Templates: []wfv1.Template{*tmpl}},
			}
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			woc.artifactRepository = &controller.Config.ArtifactRepository
			opts := &createWorkflowPodOpts{includeScriptOutput: true, executionDeadline: time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)}
			mainCtr := func(tmpl *wfv1.Template) apiv1.Container {
				if tmpl.Container != nil {
					return *tmpl.Container
				}
				return tmpl.Script.Container
			}

			// Both run the same template, pool and all; only how it is run differs.
			tmpl.Pool = &wfv1.Pool{}
			podTmpl := tmpl.DeepCopy()
			pod, err := woc.createWorkflowPod(ctx, "env", []apiv1.Container{mainCtr(podTmpl)}, podTmpl, opts)
			require.NoError(t, err)
			require.NotNil(t, pod)
			supervisor := containerByName(pod.Spec.Containers, common.SupervisorContainerName)
			main := containerByName(pod.Spec.Containers, common.MainContainerName)
			require.NotNil(t, supervisor)
			require.NotNil(t, main)
			podCfg, err := argoexec.TaskConfigFromLookup(envLookup(supervisor.Env), t.TempDir())
			require.NoError(t, err)

			pooledTmpl := tmpl.DeepCopy()
			pb := woc.newPodBuilder(ctx, "env", []apiv1.Container{mainCtr(pooledTmpl)}, pooledTmpl, opts)
			task, err := pb.buildPooledTask(ctx, "")
			require.NoError(t, err)
			require.NotNil(t, task)
			// The worker's view of the TaskSet, which reaches it through the API.
			f := dispatch.NewFake()
			enqueued, err := f.Dispatcher().Enqueue(ctx, *task)
			require.NoError(t, err)
			data, err := json.Marshal(enqueued.TaskSet)
			require.NoError(t, err)
			var taskSet wfv1.WorkflowTaskSet
			require.NoError(t, json.Unmarshal(data, &taskSet))
			candidate, ok := dispatch.CandidateFromTaskSet(&taskSet)
			require.True(t, ok)
			claimed, err := f.Consumer(dispatch.Worker{Namespace: "default", PoolHash: task.PoolHash, PodName: "worker", PodUID: "worker-uid"}).Claim(ctx, candidate)
			require.NoError(t, err)
			pooledCfg, err := pooled.TaskConfig(&taskSet, claimed.Claim())
			require.NoError(t, err)

			// The same trace; a pod's spans continue the span that created the
			// pod, a pooled task's the span of the reconcile that enqueued it.
			require.NotEmpty(t, podCfg.TraceContext, "the corpus has a trace context")
			assert.Equal(t, traceID(t, podCfg.TraceContext), traceID(t, pooledCfg.TraceContext))
			podCfg.TraceContext, pooledCfg.TraceContext = nil, nil
			podMainEnv := slices.DeleteFunc(mainTaskEnv(main), func(e string) bool { return strings.HasPrefix(e, "TRACEPARENT=") })
			pooledMainEnv := slices.DeleteFunc(pooledCfg.MainEnv(), func(e string) bool { return strings.HasPrefix(e, "TRACEPARENT=") })
			// A pod's main container spec carries the command it runs: the
			// image's entrypoint and cmd, looked up as the controller does.
			if len(mainCtr(tmpl).Command) == 0 {
				assert.Equal(t, append(slices.Clone(entrypointOf(t, main)), main.Args...), append(slices.Clone(pooledCfg.Command), mainCtr(tmpl).Args...))
				pooledCfg.Command = nil
			} else {
				assert.Nil(t, pooledCfg.Command)
			}
			if len(tmpl.Volumes) > 0 {
				assert.Equal(t, []string{"/scratch"}, pooledCfg.EmptyDirs)
				pooledCfg.EmptyDirs = nil
			}
			assert.Equal(t, podCfg, pooledCfg)
			assert.Equal(t, podMainEnv, pooledMainEnv)
		})
	}
}

func envLookup(env []apiv1.EnvVar) func(string) (string, bool) {
	return func(key string) (string, bool) {
		for _, e := range env {
			if e.Name == key && e.ValueFrom == nil {
				return e.Value, true
			}
		}
		return "", false
	}
}

// entrypointOf is what a pod's main runs before its args: the command the
// emissary runs, after its own arguments.
func entrypointOf(t *testing.T, main *apiv1.Container) []string {
	t.Helper()
	i := slices.Index(main.Command, "--")
	require.GreaterOrEqual(t, i, 0, "main runs through the emissary: %v", main.Command)
	return main.Command[i+1:]
}

// poolMainEnv are the ARGO_* variables a pool's main gets as constants of
// its own container, or not at all: its name, and the init-less layout's
// wait for the supervisor, which a pool's main replaces with waiting for
// tasks. ARGO_TEMPLATE goes to main in its task's files instead.
var poolMainEnv = []string{common.EnvVarContainerName, common.EnvVarWaitForReady, common.EnvVarTemplate}

func traceID(t *testing.T, carrier map[string]string) trace.TraceID {
	t.Helper()
	sc := trace.SpanContextFromContext(propagation.TraceContext{}.Extract(t.Context(), propagation.MapCarrier(carrier)))
	require.True(t, sc.IsValid())
	return sc.TraceID()
}

// mainTaskEnv is the per-task env a pod gives its main container: the ARGO_*
// and trace variables after the template's own, except those a pool's main
// has as constants of its own container.
func mainTaskEnv(main *apiv1.Container) []string {
	var env []string
	for _, e := range main.Env {
		isTask := strings.HasPrefix(e.Name, "ARGO_") || e.Name == "TRACEPARENT" || e.Name == "TRACESTATE"
		if !isTask || e.ValueFrom != nil || slices.Contains(poolMainEnv, e.Name) {
			continue
		}
		env = append(env, e.Name+"="+e.Value)
	}
	return env
}
