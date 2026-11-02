package controller

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/pool"
)

// poolInputClass is how a controller input of pod construction reaches a
// pooled task.
type poolInputClass int

const (
	// inputPool feeds spec.pool and so the hash.
	inputPool poolInputClass = iota + 1
	// inputPoolUnhashed feeds spec.pool but not the hash.
	inputPoolUnhashed
	// inputPerTask differs per task: it goes in spec.dispatch or the TaskSet,
	// never in spec.pool.
	inputPerTask
	// inputNotPool never reaches a pool's pods.
	inputNotPool
	// inputElsewhere is classified field by field elsewhere: by the pool
	// package's template and workflow spec tables, or by other entries here.
	inputElsewhere
)

type poolInput struct {
	class poolInputClass
	// via, for an inputPool entry, is the entry that carries it into
	// buildPoolSpec, when buildPoolSpec doesn't read it itself.
	via string
	why string
}

func poolInputVia(via string) poolInput { return poolInput{class: inputPool, via: via} }

func notPool(why string) poolInput { return poolInput{class: inputNotPool, why: why} }

func perTaskInput(why string) poolInput { return poolInput{class: inputPerTask, why: why} }

func elsewhere(why string) poolInput { return poolInput{class: inputElsewhere, why: why} }

// poolControllerInputs classifies every input of pod construction: the pod
// builder (its snapshot, options, dependencies and layout), buildPoolSpec's
// extras, and the controller config. Keys are type name and field or method.
var poolControllerInputs = map[string]poolInput{
	"podBuilder.in":     elsewhere("podBuilderInputs entries"),
	"podBuilder.deps":   elsewhere("podBuilderDeps entries"),
	"podBuilder.layout": elsewhere("chosen from Config.InitlessPod"),

	"podBuilderInputs.nodeName":                  perTaskInput("node identity"),
	"podBuilderInputs.nodeID":                    perTaskInput("node identity"),
	"podBuilderInputs.namespace":                 perTaskInput("the TaskSet's, which is the pool's; pools are identified by (namespace, hash), so it isn't hashed"),
	"podBuilderInputs.wfName":                    perTaskInput("ARGO_WORKFLOW_NAME and labels"),
	"podBuilderInputs.ownerRef":                  notPool("pools have no owner reference: they outlive the workflows they serve"),
	"podBuilderInputs.mainCtrs":                  elsewhere("the template's containers, in the pool package's tables"),
	"podBuilderInputs.tmpl":                      elsewhere("the pool package's field tables"),
	"podBuilderInputs.opts":                      elsewhere("createWorkflowPodOpts entries"),
	"podBuilderInputs.boundaryTemplate":          {class: inputPool, why: "its scheduling constraints are folded into spec.pool's template"},
	"podBuilderInputs.execWfSpec":                elsewhere("the pool package's field tables"),
	"podBuilderInputs.globalParams":              perTaskInput("already substituted into the template and execution spec"),
	"podBuilderInputs.volumes":                   {class: inputPool, why: "the workflow volumes the template mounts"},
	"podBuilderInputs.artifactRepository":        {class: inputPool, why: "the selected repository, key format unresolved"},
	"podBuilderInputs.configMapIndexer":          notPool("only substitutes podSpecPatch, which pooled templates can't set"),
	"podBuilderInputs.execCfg":                   elsewhere("execContainerConfig entries"),
	"podBuilderInputs.config":                    elsewhere("Config entries"),
	"podBuilderInputs.mainContainerDefaults":     {class: inputPool},
	"podBuilderInputs.progressPatchTickDuration": perTaskInput("spec.dispatch"),
	"podBuilderInputs.progressFileTickDuration":  perTaskInput("spec.dispatch"),
	"podBuilderInputs.workflowDeadline":          perTaskInput("spec.dispatch deadline"),
	"podBuilderInputs.computedWorkflowDeadline":  perTaskInput("spec.dispatch deadline"),
	"podBuilderInputs.now":                       perTaskInput("deadlines"),
	"podBuilderInputs.podStatusCaptureFinalizer": notPool("lets the controller see a task pod's final status; a pooled task reports through its result"),
	"podBuilderInputs.podNameVersion":            perTaskInput("the task name, {{node.taskName}}"),
	"podBuilderInputs.log":                       notPool("the controller's logger; its level reaches pools through getExecutorLogOpts"),

	"execContainerConfig.instanceID":                 {class: inputPool},
	"execContainerConfig.kubeConfig":                 {class: inputPool},
	"execContainerConfig.image":                      {class: inputPool, why: "the command-line flag first, then Config.Executor"},
	"execContainerConfig.imagePullPolicy":            {class: inputPool, why: "as image"},
	"execContainerConfig.workflowUID":                perTaskInput("ARGO_WORKFLOW_UID, kept out of the Deployment"),
	"execContainerConfig.executor":                   poolInputVia("Config.Executor"),
	"execContainerConfig.executorServiceAccountName": notPool("empty in the snapshot; the pool's executor token comes from poolSpecExtras.executorTokenSecretName"),

	"createWorkflowPodOpts.includeScriptOutput": perTaskInput("spec.dispatch"),
	"createWorkflowPodOpts.onExitPod":           perTaskInput("spec.dispatch deadline, and the TaskSet's on-exit label"),
	"createWorkflowPodOpts.executionDeadline":   perTaskInput("spec.dispatch deadline"),

	"podBuilderDeps.lookupImage":                     perTaskInput("spec.dispatch command"),
	"podBuilderDeps.getExecutorLogOpts":              poolInputVia("poolSpecExtras.executorLogArgs"),
	"podBuilderDeps.createArtifactVolumeMounts":      elsewhere("plugin volumes, from spec.pool artifactPlugins"),
	"podBuilderDeps.addInputArtifactsVolumes":        elsewhere("from the template and Config.InitlessPod"),
	"podBuilderDeps.newArtifactPluginInitContainers": elsewhere("from spec.pool artifactPlugins and the executor settings"),
	"podBuilderDeps.addArtifactPluginsLegacy":        elsewhere("as newArtifactPluginInitContainers"),
	"podBuilderDeps.addArtifactPluginsInitless":      elsewhere("as newArtifactPluginInitContainers"),
	"podBuilderDeps.getNodeByName":                   perTaskInput("the node's start and phase, for spec.dispatch deadlines"),
	"podBuilderDeps.findRetryNode":                   notPool("only for retryStrategy.affinity, which pooled templates can't set"),
	"podBuilderDeps.retryStrategyForTemplate":        notPool("as findRetryNode"),
	"podBuilderDeps.retryNodeTemplate":               notPool("as findRetryNode"),
	"podBuilderDeps.applyRetryOnDifferentHost":       notPool("as findRetryNode"),
	"podBuilderDeps.getServiceAccountTokenName":      poolInputVia("poolSpecExtras.executorTokenSecretName"),
	"podBuilderDeps.getPodGCDelay":                   notPool("pool pods aren't garbage collected per task"),
	"podBuilderDeps.persistentVolumeClaims":          notPool("pooled templates can't mount volumeClaimTemplates volumes"),

	"poolSpecExtras.executorLogArgs":         {class: inputPool},
	"poolSpecExtras.executorTokenSecretName": {class: inputPool},
	"poolSpecExtras.workflowTemplateName":    {class: inputPool, why: "the default pool name"},
	"poolSpecExtras.controllerVersion":       {class: inputPool},

	"Config.NodeEvents":               notPool("controller events"),
	"Config.WorkflowEvents":           notPool("controller events"),
	"Config.Executor":                 {class: inputPool, why: "whole; its image and pull policy also reach execContainerConfig image and imagePullPolicy"},
	"Config.MainContainer":            poolInputVia("podBuilderInputs.mainContainerDefaults"),
	"Config.KubeConfig":               poolInputVia("execContainerConfig.kubeConfig"),
	"Config.ArtifactRepository":       poolInputVia("podBuilderInputs.artifactRepository"),
	"Config.Namespace":                notPool("the controller's watch"),
	"Config.InstanceID":               poolInputVia("execContainerConfig.instanceID"),
	"Config.MetricsConfig":            notPool("controller metrics"),
	"Config.TelemetryConfig":          notPool("controller telemetry"),
	"Config.Parallelism":              notPool("controller only"),
	"Config.NamespaceParallelism":     notPool("controller only"),
	"Config.ResourceRateLimit":        notPool("the controller's create rate"),
	"Config.Persistence":              notPool("controller only"),
	"Config.Links":                    notPool("UI"),
	"Config.Columns":                  notPool("UI"),
	"Config.WorkflowDefaults":         elsewhere("merged into the workflow, so classified by the pool package's tables"),
	"Config.PodSpecLogStrategy":       notPool("controller logging"),
	"Config.PodGCGracePeriodSeconds":  notPool("pod GC, which doesn't delete pool pods"),
	"Config.PodGCDeleteDelayDuration": notPool("as podGCGracePeriodSeconds"),
	"Config.WorkflowRestrictions":     notPool("controller only"),
	"Config.InitialDelay":             notPool("controller only"),
	"Config.Images":                   perTaskInput("the image entrypoint lookup for spec.dispatch command"),
	"Config.RetentionPolicy":          notPool("controller only"),
	"Config.NavColor":                 notPool("UI"),
	"Config.SSO":                      notPool("argo-server"),
	"Config.Synchronization":          notPool("controller only"),
	"Config.ArtifactDrivers":          {class: inputPool, why: "plugin images and timeouts in spec.pool artifactPlugins"},
	"Config.FailedPodRestart":         notPool("recreates task pods that fail before running; a worker pod holds no task then"),
	"Config.DisableAgentPodCreation":  notPool("agent pods only"),
	"Config.InitlessPod":              {class: inputPool, why: "the pod layout"},
	"Config.HotWorkers":               elsewhere("HotWorkersConfig entries"),

	"HotWorkersConfig.Enabled":                       notPool("gates dispatch and the pool controller"),
	"HotWorkersConfig.LeaseSlackSeconds":             notPool("the controller's lease"),
	"HotWorkersConfig.DefaultRetentionSeconds":       {class: inputPoolUnhashed, why: "spec.pool retentionSeconds when the template sets none"},
	"HotWorkersConfig.TerminationGracePeriodSeconds": {class: inputPool},
	"HotWorkersConfig.ReleaseReserveSeconds":         {class: inputPool},
}

// poolInputTypes are the types whose every field (or, for an interface, every
// method) poolControllerInputs classifies, by key prefix.
var poolInputTypes = map[string]reflect.Type{
	"podBuilder":            reflect.TypeFor[podBuilder](),
	"podBuilderInputs":      reflect.TypeFor[podBuilderInputs](),
	"execContainerConfig":   reflect.TypeFor[execContainerConfig](),
	"createWorkflowPodOpts": reflect.TypeFor[createWorkflowPodOpts](),
	"podBuilderDeps":        reflect.TypeFor[podBuilderDeps](),
	"poolSpecExtras":        reflect.TypeFor[poolSpecExtras](),
	"Config":                reflect.TypeFor[config.Config](),
	"HotWorkersConfig":      reflect.TypeFor[config.HotWorkersConfig](),
}

func poolInputKeys(prefix string, typ reflect.Type) []string {
	var keys []string
	if typ.Kind() == reflect.Interface {
		for method := range typ.Methods() {
			keys = append(keys, prefix+"."+method.Name)
		}
		return keys
	}
	for field := range typ.Fields() {
		keys = append(keys, prefix+"."+field.Name)
	}
	return keys
}

// TestPoolControllerInputsClassified is the controller-input completeness
// test: every input of pod construction is classified, and every
// classification is of an input that exists.
func TestPoolControllerInputsClassified(t *testing.T) {
	var want []string
	for prefix, typ := range poolInputTypes {
		want = append(want, poolInputKeys(prefix, typ)...)
	}
	assert.ElementsMatch(t, want, slices.Collect(maps.Keys(poolControllerInputs)))
	for key, in := range poolControllerInputs {
		if in.via != "" {
			assert.Equal(t, inputPool, poolControllerInputs[in.via].class, "%s is carried by %s, which must feed the pool", key, in.via)
		}
	}
}

// poolSpecChange is a change to a pooled task's inputs, and what it must do
// to spec.pool and its hash.
type poolSpecChange struct {
	// key is the poolControllerInputs key, or the pool field table path, of
	// what changes.
	key   string
	name  string
	setup func(in *podBuilderInputs, x *poolSpecExtras)
	// change is applied after setup, to compare against setup alone.
	change func(in *podBuilderInputs, x *poolSpecExtras)
	// check, if set, checks the changed spec.
	check func(t *testing.T, spec *wfv1.PoolSpec)
	// unhashed changes spec.pool without changing the hash.
	unhashed bool
}

func withExecutorAccount(in *podBuilderInputs, x *poolSpecExtras) {
	in.tmpl.Executor = &wfv1.ExecutorConfig{ServiceAccountName: "exec"}
	x.executorTokenSecretName = "exec-token"
}

func withPluginInput(in *podBuilderInputs, _ *poolSpecExtras) {
	in.config.ArtifactDrivers = []config.ArtifactDriver{{Name: "plugin", Image: "plugin:v1"}}
	in.tmpl.Inputs.Artifacts = []wfv1.Artifact{{Name: "in", Path: "/in", ArtifactLocation: wfv1.ArtifactLocation{Plugin: &wfv1.PluginArtifact{Name: "plugin", Configuration: "{}", Key: "k"}}}}
}

func withWorkflowVolumeMount(in *podBuilderInputs, _ *poolSpecExtras) {
	in.tmpl.Container.VolumeMounts = []apiv1.VolumeMount{{Name: "wf-vol", MountPath: "/wf"}}
	in.volumes = []apiv1.Volume{{Name: "wf-vol", VolumeSource: apiv1.VolumeSource{EmptyDir: &apiv1.EmptyDirVolumeSource{}}}}
}

// poolSpecChanges covers every spec.pool input buildPoolSpec reads: the
// controller inputs classified inputPool or inputPoolUnhashed without a via,
// and the pool field tables' fields resolved into another spec.pool field.
var poolSpecChanges = []poolSpecChange{
	// Controller inputs.
	{key: "podBuilderInputs.boundaryTemplate", change: func(in *podBuilderInputs, _ *poolSpecExtras) {
		in.boundaryTemplate = &wfv1.Template{NodeSelector: map[string]string{"zone": "a"}}
	}, check: func(t *testing.T, spec *wfv1.PoolSpec) {
		assert.Equal(t, map[string]string{"zone": "a"}, spec.Template.NodeSelector)
	}},
	{key: "podBuilderInputs.volumes", setup: withWorkflowVolumeMount, change: func(in *podBuilderInputs, _ *poolSpecExtras) {
		in.volumes[0].VolumeSource = apiv1.VolumeSource{HostPath: &apiv1.HostPathVolumeSource{Path: "/data"}}
	}},
	{key: "podBuilderInputs.artifactRepository", name: "bucket", change: func(in *podBuilderInputs, _ *poolSpecExtras) {
		in.artifactRepository.S3.Bucket = "other"
	}},
	{key: "podBuilderInputs.artifactRepository", name: "key format", change: func(in *podBuilderInputs, _ *poolSpecExtras) {
		in.artifactRepository.S3.KeyFormat = "prefix/{{node.taskName}}"
	}},
	{key: "podBuilderInputs.artifactRepository", name: "secret", change: func(in *podBuilderInputs, _ *poolSpecExtras) {
		in.artifactRepository.S3.SecretKeySecret = secretRef("other-creds", "secret")
	}},
	{key: "execContainerConfig.instanceID", change: func(in *podBuilderInputs, _ *poolSpecExtras) { in.execCfg.instanceID = "other" }},
	{key: "podBuilderInputs.mainContainerDefaults", change: func(in *podBuilderInputs, _ *poolSpecExtras) {
		in.mainContainerDefaults = &apiv1.Container{Resources: apiv1.ResourceRequirements{Requests: apiv1.ResourceList{apiv1.ResourceCPU: resource.MustParse("100m")}}}
	}},
	{key: "execContainerConfig.kubeConfig", change: func(in *podBuilderInputs, _ *poolSpecExtras) {
		in.execCfg.kubeConfig = &config.KubeConfig{SecretName: "kc", SecretKey: "config"}
	}},
	{key: "execContainerConfig.image", name: "set only by the command-line flag", change: func(in *podBuilderInputs, _ *poolSpecExtras) {
		in.execCfg.image = "argoexec:other"
	}},
	{key: "execContainerConfig.imagePullPolicy", change: func(in *podBuilderInputs, _ *poolSpecExtras) {
		in.execCfg.imagePullPolicy = apiv1.PullAlways
	}},
	{key: "poolSpecExtras.executorLogArgs", change: func(_ *podBuilderInputs, x *poolSpecExtras) {
		x.executorLogArgs = []string{"--loglevel", "info", "--log-format", "json"}
	}},
	{key: "poolSpecExtras.executorTokenSecretName", setup: withExecutorAccount, change: func(_ *podBuilderInputs, x *poolSpecExtras) {
		x.executorTokenSecretName = "rotated-token"
	}},
	{key: "poolSpecExtras.workflowTemplateName", setup: func(in *podBuilderInputs, _ *poolSpecExtras) { in.tmpl.Pool.Name = "" },
		change: func(_ *podBuilderInputs, x *poolSpecExtras) { x.workflowTemplateName = "wft" },
		check:  func(t *testing.T, spec *wfv1.PoolSpec) { assert.Equal(t, "wft-tmpl", spec.Name) }},
	{key: "poolSpecExtras.controllerVersion", change: func(_ *podBuilderInputs, x *poolSpecExtras) { x.controllerVersion = "v4.0.1" }},
	{key: "Config.Executor", change: func(in *podBuilderInputs, _ *poolSpecExtras) {
		in.config.Executor = &apiv1.Container{Env: []apiv1.EnvVar{{Name: "X", Value: "y"}}}
	}},
	{key: "Config.ArtifactDrivers", setup: withPluginInput, change: func(in *podBuilderInputs, _ *poolSpecExtras) {
		in.config.ArtifactDrivers[0].Image = "plugin:v2"
	}},
	{key: "Config.InitlessPod", change: func(in *podBuilderInputs, _ *poolSpecExtras) {
		in.config.InitlessPod = &config.InitlessPodConfig{Enabled: true}
	}},
	{key: "HotWorkersConfig.TerminationGracePeriodSeconds", change: func(in *podBuilderInputs, _ *poolSpecExtras) {
		in.config.HotWorkers = &config.HotWorkersConfig{TerminationGracePeriodSeconds: new(int64(60))}
	}},
	{key: "HotWorkersConfig.ReleaseReserveSeconds", change: func(in *podBuilderInputs, _ *poolSpecExtras) {
		in.config.HotWorkers = &config.HotWorkersConfig{ReleaseReserveSeconds: new(int64(5))}
	}},
	{key: "HotWorkersConfig.DefaultRetentionSeconds", unhashed: true, change: func(in *podBuilderInputs, _ *poolSpecExtras) {
		in.tmpl.Pool.RetentionSeconds = nil
		in.config.HotWorkers = &config.HotWorkersConfig{DefaultRetentionSeconds: new(int64(60))}
	}, check: func(t *testing.T, spec *wfv1.PoolSpec) { assert.Equal(t, int64(60), spec.RetentionSeconds) }},

	// Template and workflow spec fields resolved into other spec.pool fields.
	{key: "Template.ServiceAccountName", change: func(in *podBuilderInputs, _ *poolSpecExtras) { in.tmpl.ServiceAccountName = "sa" },
		check: func(t *testing.T, spec *wfv1.PoolSpec) { assert.Equal(t, "sa", spec.ServiceAccountName) }},
	{key: "Template.AutomountServiceAccountToken", setup: withExecutorAccount, change: func(in *podBuilderInputs, _ *poolSpecExtras) {
		in.tmpl.AutomountServiceAccountToken = new(false)
	}, check: func(t *testing.T, spec *wfv1.PoolSpec) {
		assert.Equal(t, new(false), spec.AutomountServiceAccountToken)
	}},
	{key: "Template.Executor", change: withExecutorAccount,
		check: func(t *testing.T, spec *wfv1.PoolSpec) {
			assert.Equal(t, "exec-token", spec.ExecutorServiceAccountTokenSecretName)
		}},
	{key: "Template.Pool", name: "name", change: func(in *podBuilderInputs, _ *poolSpecExtras) { in.tmpl.Pool.Name = "other" },
		check: func(t *testing.T, spec *wfv1.PoolSpec) { assert.Equal(t, "other", spec.Name) }},
	{key: "Template.Pool", name: "initialReplicas", unhashed: true, change: func(in *podBuilderInputs, _ *poolSpecExtras) {
		in.tmpl.Pool.InitialReplicas = new(intstr.FromInt32(5))
	}, check: func(t *testing.T, spec *wfv1.PoolSpec) { assert.Equal(t, int32(5), spec.InitialReplicas) }},
	{key: "Template.Pool", name: "retentionSeconds", unhashed: true, change: func(in *podBuilderInputs, _ *poolSpecExtras) {
		in.tmpl.Pool.RetentionSeconds = new(intstr.FromInt32(5))
	}, check: func(t *testing.T, spec *wfv1.PoolSpec) { assert.Equal(t, int64(5), spec.RetentionSeconds) }},
	{key: "WorkflowSpec.ServiceAccountName", change: func(in *podBuilderInputs, _ *poolSpecExtras) { in.execWfSpec.ServiceAccountName = "sa" },
		check: func(t *testing.T, spec *wfv1.PoolSpec) { assert.Equal(t, "sa", spec.ServiceAccountName) }},
	{key: "WorkflowSpec.AutomountServiceAccountToken", setup: withExecutorAccount, change: func(in *podBuilderInputs, _ *poolSpecExtras) {
		in.execWfSpec.AutomountServiceAccountToken = new(false)
	}, check: func(t *testing.T, spec *wfv1.PoolSpec) {
		assert.Equal(t, new(false), spec.AutomountServiceAccountToken)
	}},
	{key: "WorkflowSpec.Executor", change: func(in *podBuilderInputs, x *poolSpecExtras) {
		in.execWfSpec.Executor = &wfv1.ExecutorConfig{ServiceAccountName: "exec"}
		x.executorTokenSecretName = "exec-token"
	}, check: func(t *testing.T, spec *wfv1.PoolSpec) {
		assert.Equal(t, "exec-token", spec.ExecutorServiceAccountTokenSecretName)
	}},
	{key: "WorkflowSpec.Volumes", name: "mounted", change: func(in *podBuilderInputs, x *poolSpecExtras) {
		withWorkflowVolumeMount(in, x)
		in.execWfSpec.Volumes = in.volumes
	}, check: func(t *testing.T, spec *wfv1.PoolSpec) {
		assert.Equal(t, []apiv1.Volume{volumeNamed("wf-vol")}, spec.WorkflowSpec.Volumes)
	}},
}

func volumeNamed(name string) apiv1.Volume {
	return apiv1.Volume{Name: name, VolumeSource: apiv1.VolumeSource{EmptyDir: &apiv1.EmptyDirVolumeSource{}}}
}

// TestPoolSpecChangesCoverInputs checks poolSpecChanges has a case for every
// controller input buildPoolSpec reads and every resolved field of the
// pool package's tables.
func TestPoolSpecChangesCoverInputs(t *testing.T) {
	var want []string
	for key, in := range poolControllerInputs {
		if (in.class == inputPool || in.class == inputPoolUnhashed) && in.via == "" {
			want = append(want, key)
		}
	}
	for _, leaf := range pool.Leaves() {
		if leaf.ResolvedInto != "" {
			want = append(want, leaf.String())
		}
	}
	var got []string
	for _, c := range poolSpecChanges {
		got = append(got, c.key)
	}
	assert.ElementsMatch(t, want, slices.Compact(slices.Sorted(slices.Values(got))))
}

// TestPoolSpecChangesChangeHash checks that any spec.pool input changes the
// hash, the same inputs give the same hash, and initialReplicas and
// retentionSeconds change only spec.pool.
func TestPoolSpecChangesChangeHash(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	for _, c := range poolSpecChanges {
		t.Run(strings.TrimSpace(fmt.Sprintf("%s %s", c.key, c.name)), func(t *testing.T) {
			build := func(change bool) (*wfv1.PoolSpec, string) {
				in, x := poolTestInputs()
				if c.setup != nil {
					c.setup(in, &x)
				}
				if change {
					c.change(in, &x)
				}
				return mustPoolHash(ctx, t, in, x)
			}
			before, beforeHash := build(false)
			_, againHash := build(false)
			require.Equal(t, beforeHash, againHash, "the same inputs give the same hash")
			after, afterHash := build(true)
			require.NotEqual(t, before, after, "the change reaches spec.pool")
			if c.unhashed {
				assert.Equal(t, beforeHash, afterHash)
			} else {
				assert.NotEqual(t, beforeHash, afterHash)
			}
			if c.check != nil {
				c.check(t, after)
			}
		})
	}

	t.Run("unmounted workflow volume", func(t *testing.T) {
		in, x := poolTestInputs()
		_, before := mustPoolHash(ctx, t, in, x)
		in.volumes = []apiv1.Volume{volumeNamed("unmounted")}
		in.execWfSpec.Volumes = in.volumes
		spec, after := mustPoolHash(ctx, t, in, x)
		assert.Nil(t, spec.WorkflowSpec)
		assert.Equal(t, before, after, "a pod for the template wouldn't get it either")
	})
}
