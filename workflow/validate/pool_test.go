package validate

import (
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// pooledWorkflow is a workflow whose steps entrypoint runs a pooled container
// template, Spec.Templates[1], changed by change.
func pooledWorkflow(change func(wf *wfv1.Workflow, tmpl *wfv1.Template)) *wfv1.Workflow {
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "pooled-"},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "main",
			Templates: []wfv1.Template{
				{Name: "main", Steps: []wfv1.ParallelSteps{{Steps: []wfv1.WorkflowStep{{Name: "run", Template: "pooled"}}}}},
				{Name: "pooled", Pool: &wfv1.Pool{}, Container: &apiv1.Container{Image: "busybox", Command: []string{"true"}}},
			},
		},
	}
	if change != nil {
		change(wf, &wf.Spec.Templates[1])
	}
	return wf
}

// asType makes tmpl a template of another type, from YAML.
func asType(src string) func(*wfv1.Workflow, *wfv1.Template) {
	return func(_ *wfv1.Workflow, tmpl *wfv1.Template) {
		var other wfv1.Template
		if err := yaml.Unmarshal([]byte(src), &other); err != nil {
			panic(err)
		}
		other.Name, other.Pool = tmpl.Name, tmpl.Pool
		*tmpl = other
	}
}

func mountAt(path string) func(*wfv1.Workflow, *wfv1.Template) {
	return func(_ *wfv1.Workflow, tmpl *wfv1.Template) {
		tmpl.Volumes = []apiv1.Volume{{Name: "data", VolumeSource: apiv1.VolumeSource{EmptyDir: &apiv1.EmptyDirVolumeSource{}}}}
		tmpl.Container.VolumeMounts = []apiv1.VolumeMount{{Name: "data", MountPath: path}}
	}
}

func asScript(_ *wfv1.Workflow, tmpl *wfv1.Template) {
	tmpl.Script = &wfv1.ScriptTemplate{Container: *tmpl.Container, Source: "echo hi"}
	tmpl.Container = nil
}

var probe = &apiv1.Probe{ProbeHandler: apiv1.ProbeHandler{Exec: &apiv1.ExecAction{Command: []string{"true"}}}}

// TestPoolableRejectsUnpoolableFields checks that each field a pooled template can't have
// is rejected at submit with a message naming it.
func TestPoolableRejectsUnpoolableFields(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(wf *wfv1.Workflow, tmpl *wfv1.Template)
		want   string
	}{
		{"steps", asType(`{steps: [[{name: s, template: leaf}]]}`), "templates.pooled.pool: only container and script templates can be pooled, not Steps"},
		{"dag", asType(`{dag: {tasks: [{name: t, template: leaf}]}}`), "not DAG"},
		{"containerSet", asType(`{containerSet: {containers: [{name: main, image: busybox}]}}`), "not ContainerSet"},
		{"resource", asType(`{resource: {action: get, manifest: "kind: ConfigMap"}}`), "not Resource"},
		{"suspend", asType(`{suspend: {}}`), "not Suspend"},
		{"data", asType(`{data: {source: {artifactPaths: {name: a, s3: {key: k}}}, transformation: [{expression: "true"}]}}`), "not Data"},
		{"http", asType(`{http: {url: "http://example.com"}}`), "not HTTP"},
		{"plugin", asType(`{plugin: {hello: {}}}`), "not Plugin"},
		{"daemon", func(_ *wfv1.Workflow, tmpl *wfv1.Template) { tmpl.Daemon = new(true) }, "templates.pooled.daemon can't be set on a pooled template"},
		{"pod.name", func(_ *wfv1.Workflow, tmpl *wfv1.Template) { tmpl.Container.Args = []string{"{{pod.name}}"} },
			"templates.pooled: {{pod.name}} can't be used in a pooled template, which has no pod of its own; use {{node.taskName}}"},
		{"pod.name with spaces", func(_ *wfv1.Workflow, tmpl *wfv1.Template) { tmpl.Container.Args = []string{"{{ pod.name }}"} }, "{{pod.name}} can't be used"},
		{"pod.name in a script", func(wf *wfv1.Workflow, tmpl *wfv1.Template) {
			asScript(wf, tmpl)
			tmpl.Script.Source = "echo {{pod.name}}"
		}, "use {{node.taskName}}"},
		{"initContainers", func(_ *wfv1.Workflow, tmpl *wfv1.Template) {
			tmpl.InitContainers = []wfv1.UserContainer{{Container: apiv1.Container{Name: "init", Image: "busybox"}}}
		}, "templates.pooled.initContainers can't be set on a pooled template"},
		{"sidecars", func(_ *wfv1.Workflow, tmpl *wfv1.Template) {
			tmpl.Sidecars = []wfv1.UserContainer{{Container: apiv1.Container{Name: "side", Image: "busybox"}}}
		}, "templates.pooled.sidecars can't be set"},
		{"podSpecPatch", func(_ *wfv1.Workflow, tmpl *wfv1.Template) { tmpl.PodSpecPatch = `{"hostname": "x"}` }, "templates.pooled.podSpecPatch can't be set"},
		{"volumeClaimTemplates mount", func(wf *wfv1.Workflow, tmpl *wfv1.Template) {
			wf.Spec.VolumeClaimTemplates = []apiv1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: "claim"}}}
			tmpl.Container.VolumeMounts = []apiv1.VolumeMount{{Name: "claim", MountPath: "/claim"}}
		}, "templates.pooled: volume 'claim' is one of spec.volumeClaimTemplates, which a pooled template can't mount"},
		{"retryStrategy.affinity", func(_ *wfv1.Workflow, tmpl *wfv1.Template) {
			tmpl.RetryStrategy = &wfv1.RetryStrategy{Affinity: &wfv1.RetryAffinity{NodeAntiAffinity: &wfv1.RetryNodeAntiAffinity{}}}
		}, "templates.pooled.retryStrategy.affinity can't be set"},
		{"input path ancestor of a volume mount", func(wf *wfv1.Workflow, tmpl *wfv1.Template) {
			mountAt("/data/shared")(wf, tmpl)
			tmpl.Inputs.Artifacts = []wfv1.Artifact{{Name: "in", Path: "/data", ArtifactLocation: wfv1.ArtifactLocation{Raw: &wfv1.RawArtifact{Data: "x"}}}}
		}, "templates.pooled.inputs.artifacts.in.path '/data' is an ancestor of volume mount 'data' (/data/shared), which a pooled template can't have"},
		{"input path is the workingDir", func(_ *wfv1.Workflow, tmpl *wfv1.Template) {
			tmpl.Container.WorkingDir = "/src"
			tmpl.Inputs.Artifacts = []wfv1.Artifact{{Name: "in", Path: "/src", ArtifactLocation: wfv1.ArtifactLocation{Raw: &wfv1.RawArtifact{Data: "x"}}}}
		}, "templates.pooled.inputs.artifacts.in.path '/src' is or holds the workingDir '/src', which a pooled template can't have"},
		{"input path holds a script's workingDir", func(wf *wfv1.Workflow, tmpl *wfv1.Template) {
			asScript(wf, tmpl)
			tmpl.Script.WorkingDir = "/src/app"
			tmpl.Inputs.Artifacts = []wfv1.Artifact{{Name: "in", Path: "/src", ArtifactLocation: wfv1.ArtifactLocation{Raw: &wfv1.RawArtifact{Data: "x"}}}}
		}, "is or holds the workingDir '/src/app'"},
		{"debug pause before", func(_ *wfv1.Workflow, tmpl *wfv1.Template) {
			tmpl.Container.Env = []apiv1.EnvVar{{Name: "ARGO_DEBUG_PAUSE_BEFORE", Value: "true"}}
		}, "templates.pooled: ARGO_DEBUG_PAUSE_BEFORE can't be set on a pooled template"},
		{"debug pause after", func(_ *wfv1.Workflow, tmpl *wfv1.Template) {
			tmpl.Container.Env = []apiv1.EnvVar{{Name: "ARGO_DEBUG_PAUSE_AFTER", Value: "true"}}
		}, "ARGO_DEBUG_PAUSE_AFTER can't be set on a pooled template"},
		{"restartPolicy", func(_ *wfv1.Workflow, tmpl *wfv1.Template) {
			tmpl.Container.RestartPolicy = new(apiv1.ContainerRestartPolicyAlways)
		}, "templates.pooled.container.restartPolicy can't be set"},
		{"restartPolicyRules", func(_ *wfv1.Workflow, tmpl *wfv1.Template) {
			tmpl.Container.RestartPolicyRules = []apiv1.ContainerRestartRule{{}}
		}, "templates.pooled.container.restartPolicyRules can't be set"},
		{"livenessProbe", func(_ *wfv1.Workflow, tmpl *wfv1.Template) { tmpl.Container.LivenessProbe = probe }, "templates.pooled.container.livenessProbe can't be set"},
		{"readinessProbe", func(_ *wfv1.Workflow, tmpl *wfv1.Template) { tmpl.Container.ReadinessProbe = probe }, "templates.pooled.container.readinessProbe can't be set"},
		{"startupProbe", func(_ *wfv1.Workflow, tmpl *wfv1.Template) { tmpl.Container.StartupProbe = probe }, "templates.pooled.container.startupProbe can't be set"},
		{"lifecycle", func(_ *wfv1.Workflow, tmpl *wfv1.Template) {
			tmpl.Container.Lifecycle = &apiv1.Lifecycle{PreStop: &apiv1.LifecycleHandler{Exec: &apiv1.ExecAction{Command: []string{"true"}}}}
		}, "templates.pooled.container.lifecycle can't be set"},
		{"script lifecycle", func(wf *wfv1.Workflow, tmpl *wfv1.Template) {
			asScript(wf, tmpl)
			tmpl.Script.Lifecycle = &apiv1.Lifecycle{PreStop: &apiv1.LifecycleHandler{Exec: &apiv1.ExecAction{Command: []string{"true"}}}}
		}, "templates.pooled.script.lifecycle can't be set"},
		{"workflow retryStrategy.affinity", func(wf *wfv1.Workflow, _ *wfv1.Template) {
			wf.Spec.RetryStrategy = &wfv1.RetryStrategy{Affinity: &wfv1.RetryAffinity{NodeAntiAffinity: &wfv1.RetryNodeAntiAffinity{}}}
		}, "templates.pooled: spec.retryStrategy.affinity can't be set on a workflow that runs a pooled template"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			err := Workflow(ctx, wftmplGetter, cwftmplGetter, pooledWorkflow(tt.change), nil, Opts{Submit: true})
			assert.ErrorContains(t, err, tt.want)
		})
	}
}

// TestPoolableAllows checks what a pooled template may do: variables in
// hashed fields, {{node.taskName}}, scripts, and the cases next to the
// rejected ones.
func TestPoolableAllows(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(wf *wfv1.Workflow, tmpl *wfv1.Template)
	}{
		{"plain", nil},
		{"script", asScript},
		{"workflow.name and node.taskName in hashed fields", func(_ *wfv1.Workflow, tmpl *wfv1.Template) {
			tmpl.Container.Image = "busybox:{{workflow.name}}"
			tmpl.Container.WorkingDir = "/work/{{node.taskName}}"
			tmpl.Metadata.Labels = map[string]string{"task": "{{node.taskName}}", "wf": "{{workflow.name}}"}
			tmpl.Container.Args = []string{"{{node.taskName}}", "{{=node.taskName}}"}
		}},
		{"workflow.name in the pool name", func(_ *wfv1.Workflow, tmpl *wfv1.Template) { tmpl.Pool.Name = "{{workflow.name}}" }},
		{"daemon false", func(_ *wfv1.Workflow, tmpl *wfv1.Template) { tmpl.Daemon = new(false) }},
		{"retryStrategy without affinity", func(_ *wfv1.Workflow, tmpl *wfv1.Template) {
			tmpl.RetryStrategy = &wfv1.RetryStrategy{RetryPolicy: wfv1.RetryPolicyAlways}
		}},
		{"input path inside a volume mount", func(wf *wfv1.Workflow, tmpl *wfv1.Template) {
			mountAt("/data")(wf, tmpl)
			tmpl.Inputs.Artifacts = []wfv1.Artifact{{Name: "in", Path: "/data/in", ArtifactLocation: wfv1.ArtifactLocation{Raw: &wfv1.RawArtifact{Data: "x"}}}}
		}},
		{"input path beneath the workingDir", func(_ *wfv1.Workflow, tmpl *wfv1.Template) {
			tmpl.Container.WorkingDir = "/src"
			tmpl.Inputs.Artifacts = []wfv1.Artifact{{Name: "in", Path: "/src/in", ArtifactLocation: wfv1.ArtifactLocation{Raw: &wfv1.RawArtifact{Data: "x"}}}}
		}},
		{"workflow volume named like a claim", func(wf *wfv1.Workflow, tmpl *wfv1.Template) {
			wf.Spec.Volumes = []apiv1.Volume{{Name: "claim", VolumeSource: apiv1.VolumeSource{EmptyDir: &apiv1.EmptyDirVolumeSource{}}}}
			wf.Spec.VolumeClaimTemplates = []apiv1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: "claim"}}}
			tmpl.Container.VolumeMounts = []apiv1.VolumeMount{{Name: "claim", MountPath: "/claim"}}
		}},
		{"claims the pooled template doesn't mount", func(wf *wfv1.Workflow, _ *wfv1.Template) {
			wf.Spec.VolumeClaimTemplates = []apiv1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: "claim"}}}
		}},
		// Validation checks variable tags, not expressions, for every
		// template; a pooled task's expression has no pod.name to read when
		// it is substituted.
		{"pod.name in an expression", func(_ *wfv1.Workflow, tmpl *wfv1.Template) {
			tmpl.Container.Env = []apiv1.EnvVar{{Name: "POD", Value: "{{=pod.name ?? 'none'}}"}}
		}},
		{"pod.name in a template that isn't pooled", func(wf *wfv1.Workflow, _ *wfv1.Template) {
			wf.Spec.Templates = append(wf.Spec.Templates, wfv1.Template{Name: "pod", Container: &apiv1.Container{Image: "busybox", Args: []string{"{{pod.name}}"}}})
		}},
		{"podSpecPatch without a pooled template", func(wf *wfv1.Workflow, tmpl *wfv1.Template) {
			wf.Spec.PodSpecPatch = `{"hostname": "x"}`
			tmpl.Pool = nil
		}},
		{"templateDefaults for another template type", func(wf *wfv1.Workflow, _ *wfv1.Template) {
			wf.Spec.TemplateDefaults = &wfv1.Template{Script: &wfv1.ScriptTemplate{Container: apiv1.Container{LivenessProbe: probe}}}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			require.NoError(t, Workflow(ctx, wftmplGetter, cwftmplGetter, pooledWorkflow(tt.change), nil, Opts{Submit: true}))
		})
	}
}

// TestPoolableChecksPool checks the pool name and initialReplicas rules, and
// that retentionSeconds is at least 0.
func TestPoolableChecksPool(t *testing.T) {
	for _, tt := range []struct {
		name string
		pool wfv1.Pool
		want string
	}{
		{"default name", wfv1.Pool{}, ""},
		{"35 characters", wfv1.Pool{Name: strings.Repeat("a", 35)}, ""},
		{"36 characters", wfv1.Pool{Name: strings.Repeat("a", 36)}, "templates.pooled.pool.name '" + strings.Repeat("a", 36) + "' is not a valid pool name: must be no more than 35 characters"},
		{"not a DNS label", wfv1.Pool{Name: "My_Pool"}, "templates.pooled.pool.name 'My_Pool' is not a valid pool name: a lowercase RFC 1123 label"},
		{"trailing dash", wfv1.Pool{Name: "pool-"}, "is not a valid pool name"},
		{"one replica", wfv1.Pool{InitialReplicas: new(intstr.FromInt32(1))}, ""},
		{"no replicas", wfv1.Pool{InitialReplicas: new(intstr.FromInt32(0))}, "templates.pooled.pool.initialReplicas must be at least 1"},
		{"replicas as a string", wfv1.Pool{InitialReplicas: new(intstr.FromString("3"))}, ""},
		{"replicas not a number", wfv1.Pool{InitialReplicas: new(intstr.FromString("three"))}, "templates.pooled.pool.initialReplicas 'three' is not a whole number"},
		{"replicas from a variable", wfv1.Pool{InitialReplicas: new(intstr.FromString("{{workflow.parameters.replicas}}"))}, ""},
		{"too many replicas", wfv1.Pool{InitialReplicas: new(intstr.FromString("2147483648"))}, "templates.pooled.pool.initialReplicas must be at most 2147483647"},
		{"zero retention", wfv1.Pool{RetentionSeconds: new(intstr.FromInt32(0))}, ""},
		{"negative retention", wfv1.Pool{RetentionSeconds: new(intstr.FromInt32(-1))}, "templates.pooled.pool.retentionSeconds must not be negative"},
		{"retention from a variable", wfv1.Pool{RetentionSeconds: new(intstr.FromString("{{workflow.parameters.retention}}"))}, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf := pooledWorkflow(func(wf *wfv1.Workflow, tmpl *wfv1.Template) {
				tmpl.Pool = &tt.pool
				wf.Spec.Arguments.Parameters = []wfv1.Parameter{
					{Name: "replicas", Value: wfv1.AnyStringPtr("2")},
					{Name: "retention", Value: wfv1.AnyStringPtr("60")},
				}
			})
			err := Workflow(ctx, wftmplGetter, cwftmplGetter, wf, nil, Opts{Submit: true})
			if tt.want == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.want)
			}
		})
	}
}

// TestPoolableWorkflowLevel checks that workflow-level fields that reach a
// pooled template are rejected at submit where they are visible: the
// workflow's own, its workflowTemplateRef's, and the workflow defaults.
func TestPoolableWorkflowLevel(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	patch := `{"hostname": "x"}`
	sidecars := []wfv1.UserContainer{{Container: apiv1.Container{Name: "side", Image: "busybox"}}}

	for _, tt := range []struct {
		name       string
		change     func(wf *wfv1.Workflow, tmpl *wfv1.Template)
		wfDefaults *wfv1.Workflow
		want       string
	}{
		{name: "workflow podSpecPatch", change: func(wf *wfv1.Workflow, _ *wfv1.Template) { wf.Spec.PodSpecPatch = patch },
			want: "templates.pooled: spec.podSpecPatch can't be set on a workflow that runs a pooled template"},
		{name: "workflow defaults podSpecPatch", wfDefaults: &wfv1.Workflow{Spec: wfv1.WorkflowSpec{PodSpecPatch: patch}},
			want: "spec.podSpecPatch can't be set"},
		{name: "templateDefaults sidecars", change: func(wf *wfv1.Workflow, _ *wfv1.Template) {
			wf.Spec.TemplateDefaults = &wfv1.Template{Sidecars: sidecars}
		}, want: "templates.pooled.sidecars can't be set on a pooled template (with spec.templateDefaults merged in)"},
		{name: "templateDefaults container lifecycle", change: func(wf *wfv1.Workflow, _ *wfv1.Template) {
			wf.Spec.TemplateDefaults = &wfv1.Template{Container: &apiv1.Container{Lifecycle: &apiv1.Lifecycle{PreStop: &apiv1.LifecycleHandler{Exec: &apiv1.ExecAction{Command: []string{"true"}}}}}}
		}, want: "templates.pooled.container.lifecycle can't be set on a pooled template (with spec.templateDefaults merged in)"},
		{name: "templateDefaults pod.name", change: func(wf *wfv1.Workflow, _ *wfv1.Template) {
			wf.Spec.TemplateDefaults = &wfv1.Template{Container: &apiv1.Container{Env: []apiv1.EnvVar{{Name: "POD", Value: "{{pod.name}}"}}}}
		}, want: "use {{node.taskName}} (with spec.templateDefaults merged in)"},
		{name: "templateDefaults claim mount", change: func(wf *wfv1.Workflow, _ *wfv1.Template) {
			wf.Spec.VolumeClaimTemplates = []apiv1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: "claim"}}}
			wf.Spec.TemplateDefaults = &wfv1.Template{Container: &apiv1.Container{VolumeMounts: []apiv1.VolumeMount{{Name: "claim", MountPath: "/claim"}}}}
		}, want: "volume 'claim' is one of spec.volumeClaimTemplates"},
		{name: "workflow defaults templateDefaults", wfDefaults: &wfv1.Workflow{Spec: wfv1.WorkflowSpec{TemplateDefaults: &wfv1.Template{Sidecars: sidecars}}},
			want: "templates.pooled.sidecars can't be set"},
		{name: "templateDefaults pool, which the steps template ignores", change: func(wf *wfv1.Workflow, tmpl *wfv1.Template) {
			tmpl.Pool = nil
			wf.Spec.TemplateDefaults = &wfv1.Template{Pool: &wfv1.Pool{}}
		}},
		{name: "templateDefaults pool and sidecars", change: func(wf *wfv1.Workflow, tmpl *wfv1.Template) {
			tmpl.Pool = nil
			wf.Spec.TemplateDefaults = &wfv1.Template{Pool: &wfv1.Pool{}, Sidecars: sidecars}
		}, want: "templates.pooled.sidecars can't be set on a pooled template (with spec.templateDefaults merged in)"},
		{name: "workflow defaults templateDefaults pool and sidecars", change: func(_ *wfv1.Workflow, tmpl *wfv1.Template) {
			tmpl.Pool = nil
		}, wfDefaults: &wfv1.Workflow{Spec: wfv1.WorkflowSpec{TemplateDefaults: &wfv1.Template{Pool: &wfv1.Pool{}, Sidecars: sidecars}}},
			want: "templates.pooled.sidecars can't be set on a pooled template (with spec.templateDefaults merged in)"},
		{name: "workflow defaults claim mounted by the template", change: func(_ *wfv1.Workflow, tmpl *wfv1.Template) {
			tmpl.Container.VolumeMounts = []apiv1.VolumeMount{{Name: "claim", MountPath: "/claim"}}
		}, wfDefaults: &wfv1.Workflow{Spec: wfv1.WorkflowSpec{VolumeClaimTemplates: []apiv1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: "claim"}}}}},
			want: "volume 'claim' is one of spec.volumeClaimTemplates"},
		{name: "the template's own claims win", change: func(_ *wfv1.Workflow, tmpl *wfv1.Template) {
			tmpl.Container.VolumeMounts = []apiv1.VolumeMount{{Name: "claim", MountPath: "/claim"}}
			tmpl.Volumes = []apiv1.Volume{{Name: "claim", VolumeSource: apiv1.VolumeSource{EmptyDir: &apiv1.EmptyDirVolumeSource{}}}}
		}, wfDefaults: &wfv1.Workflow{Spec: wfv1.WorkflowSpec{VolumeClaimTemplates: []apiv1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: "claim"}}}}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := Workflow(ctx, wftmplGetter, cwftmplGetter, pooledWorkflow(tt.change), tt.wfDefaults, Opts{Submit: true})
			if tt.want == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.want)
			}
		})
	}

	t.Run("workflowTemplateRef podSpecPatch", func(t *testing.T) {
		wftmpl := &wfv1.WorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: "pooled-patched"}, Spec: pooledWorkflow(nil).Spec}
		wftmpl.Spec.PodSpecPatch = patch
		require.NoError(t, createWorkflowTemplate(ctx, wftmpl))
		defer func() { _ = deleteWorkflowTemplate(ctx, wftmpl.Name) }()
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{GenerateName: "pooled-"},
			Spec:       wfv1.WorkflowSpec{WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: wftmpl.Name}},
		}
		err := Workflow(ctx, wftmplGetter, cwftmplGetter, wf, nil, Opts{Submit: true})
		assert.ErrorContains(t, err, "spec.podSpecPatch can't be set on a workflow that runs a pooled template")
	})

	t.Run("workflowTemplateRef with the workflow's podSpecPatch", func(t *testing.T) {
		wftmpl := &wfv1.WorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: "pooled-unpatched"}, Spec: pooledWorkflow(nil).Spec}
		require.NoError(t, createWorkflowTemplate(ctx, wftmpl))
		defer func() { _ = deleteWorkflowTemplate(ctx, wftmpl.Name) }()
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{GenerateName: "pooled-"},
			Spec:       wfv1.WorkflowSpec{WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: wftmpl.Name}},
		}
		require.NoError(t, Workflow(ctx, wftmplGetter, cwftmplGetter, wf, nil, Opts{Submit: true}))
		wf.Spec.PodSpecPatch = patch
		err := Workflow(ctx, wftmplGetter, cwftmplGetter, wf, nil, Opts{Submit: true})
		assert.ErrorContains(t, err, "spec.podSpecPatch can't be set")
	})
}

// TestPoolableEverywhere checks that rejection works through templateRef and in
// WorkflowTemplate, ClusterWorkflowTemplate and CronWorkflow lint.
func TestPoolableEverywhere(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	withSidecar := func(_ *wfv1.Workflow, tmpl *wfv1.Template) {
		tmpl.Sidecars = []wfv1.UserContainer{{Container: apiv1.Container{Name: "side", Image: "busybox"}}}
	}
	const want = "templates.pooled.sidecars can't be set on a pooled template"

	t.Run("templateRef", func(t *testing.T) {
		wftmpl := &wfv1.WorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: "pooled-library"}, Spec: pooledWorkflow(withSidecar).Spec}
		wftmpl.Spec.Entrypoint = ""
		require.NoError(t, createWorkflowTemplate(ctx, wftmpl))
		defer func() { _ = deleteWorkflowTemplate(ctx, wftmpl.Name) }()
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{GenerateName: "pooled-"},
			Spec: wfv1.WorkflowSpec{
				Entrypoint: "main",
				Templates: []wfv1.Template{{Name: "main", Steps: []wfv1.ParallelSteps{{Steps: []wfv1.WorkflowStep{
					{Name: "run", TemplateRef: &wfv1.TemplateRef{Name: wftmpl.Name, Template: "pooled"}},
				}}}}},
			},
		}
		require.ErrorContains(t, Workflow(ctx, wftmplGetter, cwftmplGetter, wf, nil, Opts{Submit: true}), want)

		// The workflow's own spec reaches the referenced template.
		clean := &wfv1.WorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: "pooled-library-clean"}, Spec: pooledWorkflow(nil).Spec}
		require.NoError(t, createWorkflowTemplate(ctx, clean))
		defer func() { _ = deleteWorkflowTemplate(ctx, clean.Name) }()
		wf.Spec.Templates[0].Steps[0].Steps[0].TemplateRef.Name = clean.Name
		require.NoError(t, Workflow(ctx, wftmplGetter, cwftmplGetter, wf, nil, Opts{Submit: true}))
		wf.Spec.PodSpecPatch = `{"hostname": "x"}`
		assert.ErrorContains(t, Workflow(ctx, wftmplGetter, cwftmplGetter, wf, nil, Opts{Submit: true}), "spec.podSpecPatch can't be set")
	})

	t.Run("WorkflowTemplate lint", func(t *testing.T) {
		wftmpl := &wfv1.WorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: "pooled-lint"}, Spec: pooledWorkflow(withSidecar).Spec}
		require.ErrorContains(t, WorkflowTemplate(ctx, wftmplGetter, cwftmplGetter, wftmpl, nil, Opts{Lint: true}), want)
		wftmpl.Spec.Entrypoint = ""
		assert.ErrorContains(t, WorkflowTemplate(ctx, wftmplGetter, cwftmplGetter, wftmpl, nil, Opts{Lint: true}), want)
	})

	t.Run("ClusterWorkflowTemplate lint", func(t *testing.T) {
		cwftmpl := &wfv1.ClusterWorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: "pooled-lint"}, Spec: pooledWorkflow(withSidecar).Spec}
		assert.ErrorContains(t, ClusterWorkflowTemplate(ctx, wftmplGetter, cwftmplGetter, cwftmpl, nil, Opts{Lint: true}), want)
	})

	t.Run("CronWorkflow lint", func(t *testing.T) {
		cronWf := &wfv1.CronWorkflow{
			ObjectMeta: metav1.ObjectMeta{Name: "pooled-cron"},
			Spec:       wfv1.CronWorkflowSpec{Schedules: []string{"* * * * *"}, WorkflowSpec: pooledWorkflow(withSidecar).Spec},
		}
		require.ErrorContains(t, CronWorkflow(ctx, wftmplGetter, cwftmplGetter, cronWf, nil), want)
		cronWf.Spec.WorkflowSpec = pooledWorkflow(nil).Spec
		require.NoError(t, CronWorkflow(ctx, wftmplGetter, cwftmplGetter, cronWf, nil))
	})
}

// TestPoolableMergedTemplate checks Poolable on a merged template and
// execution spec, as the controller calls it at dispatch to catch what
// only merging shows.
func TestPoolableMergedTemplate(t *testing.T) {
	tmpl := &pooledWorkflow(nil).Spec.Templates[1]
	require.NoError(t, Poolable(tmpl, nil))
	require.NoError(t, Poolable(tmpl, &wfv1.WorkflowSpec{}))
	require.NoError(t, Poolable(&wfv1.Template{Name: "not-pooled", Sidecars: []wfv1.UserContainer{{}}}, &wfv1.WorkflowSpec{PodSpecPatch: "{}"}))

	require.ErrorContains(t, Poolable(tmpl, &wfv1.WorkflowSpec{PodSpecPatch: "{}"}), "templates.pooled: spec.podSpecPatch can't be set")

	tmpl.Pool.Name = "{{workflow.name}}"
	require.NoError(t, Poolable(tmpl, nil), "checked once substituted")
	tmpl.Pool.Name = "Substituted_Name"
	assert.ErrorContains(t, Poolable(tmpl, nil), "is not a valid pool name")
}
