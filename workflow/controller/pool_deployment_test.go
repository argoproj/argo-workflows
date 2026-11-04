package controller

import (
	"cmp"
	"context"
	"flag"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

var updatePoolGolden = flag.Bool("update-pool-golden", false, "rewrite testdata/pooldeployment from buildPoolDeployment")

const poolTestExecutorImage = "quay.io/argoproj/argoexec:v4.0.0"

// poolTestPluginEntrypoints are the image entrypoints of the plugins the tests
// configure, as dispatch would look them up into spec.pool.
var poolTestPluginEntrypoints = map[wfv1.ArtifactPluginName][]string{
	"in-plugin":   {"/bin/in-plugin"},
	"out-plugin":  {"/bin/out-plugin"},
	"both-plugin": {"/plugin-server"},
}

// mustPoolDeploymentSpec builds spec.pool and its hash as dispatch does: from
// the inputs, with each plugin's Command set to its looked-up entrypoint.
func mustPoolDeploymentSpec(ctx context.Context, t *testing.T, in *podBuilderInputs, x poolSpecExtras) (*wfv1.PoolSpec, string) {
	t.Helper()
	spec, err := buildPoolSpec(ctx, in, x)
	require.NoError(t, err)
	for i := range spec.ArtifactPlugins {
		spec.ArtifactPlugins[i].Command = poolTestPluginEntrypoints[spec.ArtifactPlugins[i].Name]
	}
	hash, err := poolHash(spec)
	require.NoError(t, err)
	return spec, hash
}

// poolDeploymentTestInputs is the pod builder snapshot and extras of a pooled
// container task with a little of everything a Deployment is built from:
// template and workflow volumes, env of each kind, metadata (including
// attempts to set reserved labels), artifacts using the repository and its
// secrets, and executor and main container config.
func poolDeploymentTestInputs() (*podBuilderInputs, poolSpecExtras) {
	in := &podBuilderInputs{
		tmpl: &wfv1.Template{
			Name: "build",
			Pool: &wfv1.Pool{Name: "builder", InitialReplicas: new(intstr.FromInt32(2))},
			Metadata: wfv1.Metadata{
				Labels:      map[string]string{"team": "a", common.LabelKeyPoolHash: "spoofed"},
				Annotations: map[string]string{"note": "from the template"},
			},
			Container: &apiv1.Container{
				Image:      "golang:1.25",
				Command:    []string{"go"},
				Args:       []string{"build", "./..."},
				WorkingDir: "/work",
				Env: []apiv1.EnvVar{
					{Name: "PLAIN", Value: "per task"},
					{Name: "TOKEN", ValueFrom: &apiv1.EnvVarSource{SecretKeyRef: secretRef("api", "token")}},
				},
				EnvFrom:      []apiv1.EnvFromSource{{ConfigMapRef: &apiv1.ConfigMapEnvSource{LocalObjectReference: apiv1.LocalObjectReference{Name: "settings"}}}},
				Resources:    apiv1.ResourceRequirements{Requests: apiv1.ResourceList{apiv1.ResourceCPU: resource.MustParse("1")}},
				VolumeMounts: []apiv1.VolumeMount{{Name: "cache", MountPath: "/cache"}, {Name: "work", MountPath: "/work"}},
			},
			Volumes: []apiv1.Volume{{Name: "cache", VolumeSource: apiv1.VolumeSource{EmptyDir: &apiv1.EmptyDirVolumeSource{}}}},
			Inputs: wfv1.Inputs{Artifacts: []wfv1.Artifact{
				{Name: "src", Path: "/work/src", ArtifactLocation: wfv1.ArtifactLocation{Git: &wfv1.GitArtifact{Repo: "https://example.com/r.git", PasswordSecret: secretRef("git", "password"), UsernameSecret: secretRef("git", "username")}}},
			}},
			Outputs: wfv1.Outputs{Artifacts: []wfv1.Artifact{{Name: "bin", Path: "/work/bin"}}},
		},
		execWfSpec: &wfv1.WorkflowSpec{
			ServiceAccountName: "runner",
			ImagePullSecrets:   []apiv1.LocalObjectReference{{Name: "regcred"}},
			PodMetadata: &wfv1.Metadata{
				Labels: map[string]string{"wf-label": "b", common.LabelKeyWorkflow: "spoofed", common.LabelKeyControllerInstanceID: "spoofed"},
			},
			Volumes: []apiv1.Volume{{Name: "work", VolumeSource: apiv1.VolumeSource{PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{ClaimName: "work"}}}},
		},
		volumes: []apiv1.Volume{{Name: "work", VolumeSource: apiv1.VolumeSource{PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{ClaimName: "work"}}}},
		artifactRepository: &wfv1.ArtifactRepository{S3: &wfv1.S3ArtifactRepository{
			S3Bucket:  wfv1.S3Bucket{Bucket: "bucket", AccessKeySecret: secretRef("repo-creds", "access"), SecretKeySecret: secretRef("repo-creds", "secret")},
			KeyFormat: wfv1.DefaultArchivePattern,
		}},
		config: &config.Config{
			Executor: &apiv1.Container{
				Args:      []string{"--executor-flag"},
				Env:       []apiv1.EnvVar{{Name: "EXECUTOR_SETTING", Value: "1"}},
				Resources: apiv1.ResourceRequirements{Requests: apiv1.ResourceList{apiv1.ResourceMemory: resource.MustParse("64Mi")}},
			},
		},
		mainContainerDefaults: &apiv1.Container{
			Env:       []apiv1.EnvVar{{Name: "MAIN_DEFAULT", Value: "all pods"}},
			Resources: apiv1.ResourceRequirements{Limits: apiv1.ResourceList{apiv1.ResourceMemory: resource.MustParse("1Gi")}},
		},
		execCfg: execContainerConfig{
			image:           poolTestExecutorImage,
			imagePullPolicy: apiv1.PullIfNotPresent,
		},
	}
	return in, poolSpecExtras{
		executorLogArgs:   []string{"--loglevel", "info", "--log-format", "text", "--gloglevel", "0"},
		controllerVersion: "v4.0.0",
	}
}

func withPoolTestInitless(in *podBuilderInputs, _ *poolSpecExtras) {
	in.config.InitlessPod = &config.InitlessPodConfig{Enabled: true}
}

// withPoolTestPlugins turns the task into a script whose input and output
// artifacts use one plugin each.
func withPoolTestPlugins(in *podBuilderInputs, _ *poolSpecExtras) {
	in.config.ArtifactDrivers = []config.ArtifactDriver{
		{Name: "in-plugin", Image: "in-plugin:v1"},
		{Name: "out-plugin", Image: "out-plugin:v1"},
	}
	in.tmpl.Script = &wfv1.ScriptTemplate{Container: *in.tmpl.Container, Source: "go build ./..."}
	in.tmpl.Container = nil
	in.tmpl.Inputs.Artifacts = append(in.tmpl.Inputs.Artifacts, wfv1.Artifact{Name: "data", Path: "/data", ArtifactLocation: wfv1.ArtifactLocation{Plugin: &wfv1.PluginArtifact{Name: "in-plugin", Configuration: "{}", Key: "k"}}})
	in.tmpl.Outputs.Artifacts = append(in.tmpl.Outputs.Artifacts, wfv1.Artifact{Name: "report", Path: "/work/report", ArtifactLocation: wfv1.ArtifactLocation{Plugin: &wfv1.PluginArtifact{Name: "out-plugin", Configuration: "{}", Key: "k"}}})
}

// withPoolTestExecutorAccount runs the executor as its own service account,
// with a controller kubeconfig and instance ID.
func withPoolTestExecutorAccount(in *podBuilderInputs, x *poolSpecExtras) {
	in.tmpl.Executor = &wfv1.ExecutorConfig{ServiceAccountName: "executor"}
	x.executorTokenSecretName = "executor-token-abcde"
	in.execCfg.kubeConfig = &config.KubeConfig{SecretName: "kubeconfig", SecretKey: "admin.conf"}
	in.execCfg.instanceID = "prod"
}

func withPoolTestNoAutomount(in *podBuilderInputs, x *poolSpecExtras) {
	in.tmpl.AutomountServiceAccountToken = new(false)
	in.tmpl.Executor = &wfv1.ExecutorConfig{ServiceAccountName: "executor"}
	x.executorTokenSecretName = "executor-token-abcde"
}

// poolDeploymentCases are the golden Deployments, each the base inputs
// with its options applied.
var poolDeploymentCases = []struct {
	name    string
	options []func(*podBuilderInputs, *poolSpecExtras)
}{
	{name: "container"},
	{name: "container-initless", options: []func(*podBuilderInputs, *poolSpecExtras){withPoolTestInitless}},
	{name: "script-plugins", options: []func(*podBuilderInputs, *poolSpecExtras){withPoolTestPlugins}},
	{name: "script-plugins-initless", options: []func(*podBuilderInputs, *poolSpecExtras){withPoolTestPlugins, withPoolTestInitless}},
	{name: "executor-service-account", options: []func(*podBuilderInputs, *poolSpecExtras){withPoolTestExecutorAccount}},
	{name: "executor-service-account-initless", options: []func(*podBuilderInputs, *poolSpecExtras){withPoolTestExecutorAccount, withPoolTestInitless}},
	{name: "no-automount-initless", options: []func(*podBuilderInputs, *poolSpecExtras){withPoolTestNoAutomount, withPoolTestInitless}},
}

// mustPoolDeployment builds spec.pool from the base inputs with options
// applied, then the pool's Deployment.
func mustPoolDeployment(ctx context.Context, t *testing.T, options ...func(*podBuilderInputs, *poolSpecExtras)) (*wfv1.PoolSpec, *appsv1.Deployment) {
	t.Helper()
	in, x := poolDeploymentTestInputs()
	for _, o := range options {
		o(in, &x)
	}
	spec, hash := mustPoolDeploymentSpec(ctx, t, in, x)
	d, err := buildPoolDeployment("ns", hash, spec)
	require.NoError(t, err)
	return spec, d
}

// allPodContainers lists a pod spec's init and regular containers.
func allPodContainers(spec *apiv1.PodSpec) []apiv1.Container {
	return slices.Concat(spec.InitContainers, spec.Containers)
}

// TestPoolDeploymentGolden checks the Deployments built for poolDeploymentCases
// against their golden files. Regenerate the files with
// `go test ./workflow/controller/ -run TestPoolDeploymentGolden -update-pool-golden`
// and review the diff.
func TestPoolDeploymentGolden(t *testing.T) {
	for _, tt := range poolDeploymentCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			_, d := mustPoolDeployment(ctx, t, tt.options...)
			got, err := yaml.Marshal(d)
			require.NoError(t, err)
			path := filepath.Join("testdata", "pooldeployment", tt.name+".yaml")
			if *updatePoolGolden {
				require.NoError(t, os.WriteFile(path, got, 0o644))
			}
			want, err := os.ReadFile(path)
			require.NoError(t, err, "run with -update-pool-golden to create it")
			assert.Equal(t, string(want), string(got))
		})
	}
}

// TestPoolDeploymentIsPure checks two builds of one spec.pool are equal and
// leave it unchanged.
func TestPoolDeploymentIsPure(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	in, x := poolDeploymentTestInputs()
	withPoolTestPlugins(in, &x)
	withPoolTestExecutorAccount(in, &x)
	spec, hash := mustPoolDeploymentSpec(ctx, t, in, x)
	before := spec.DeepCopy()
	first, err := buildPoolDeployment("ns", hash, spec)
	require.NoError(t, err)
	second, err := buildPoolDeployment("ns", hash, spec)
	require.NoError(t, err)
	assert.Equal(t, first, second)
	assert.Equal(t, before, spec)
}

// TestPoolDeploymentCredentialsOnlyInWorker checks that artifact secrets, the
// executor token and the kubeconfig, every Secret volume of a pool pod, are
// mounted by worker and no other container.
func TestPoolDeploymentCredentialsOnlyInWorker(t *testing.T) {
	for _, tt := range poolDeploymentCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			spec, d := mustPoolDeployment(ctx, t, tt.options...)
			podSpec := &d.Spec.Template.Spec
			var secretVolumes []string
			for _, v := range podSpec.Volumes {
				if v.Secret != nil {
					secretVolumes = append(secretVolumes, v.Name)
				}
			}
			require.NotEmpty(t, secretVolumes)
			if spec.ExecutorServiceAccountTokenSecretName != "" {
				assert.Contains(t, secretVolumes, common.ServiceAccountTokenVolumeName)
			}
			for _, c := range allPodContainers(podSpec) {
				var mounted []string
				for _, m := range c.VolumeMounts {
					if slices.Contains(secretVolumes, m.Name) {
						mounted = append(mounted, m.Name)
					}
				}
				if c.Name == common.WorkerContainerName {
					assert.ElementsMatch(t, secretVolumes, mounted, "worker mounts every Secret volume")
				} else {
					assert.Empty(t, mounted, "container %s", c.Name)
				}
			}
		})
	}
}

// TestPoolWorkerMountsMatchSupervisor checks that worker mounts
// what a pod's supervisor, its init-less counterpart, mounts for the same
// template, in either pool layout, and each plugin sidecar what the pod's
// does. The supervisor is the reference rather than the legacy wait
// container because a pool always stages inputs as init-less pods do.
func TestPoolWorkerMountsMatchSupervisor(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tmpl := &wfv1.Template{
		Name: "t",
		Script: &wfv1.ScriptTemplate{
			Container: apiv1.Container{Image: "python:3", Command: []string{"python"}, VolumeMounts: []apiv1.VolumeMount{{Name: "cache", MountPath: "/cache"}}},
			Source:    "print(1)",
		},
		Volumes: []apiv1.Volume{{Name: "cache", VolumeSource: apiv1.VolumeSource{EmptyDir: &apiv1.EmptyDirVolumeSource{}}}},
		Inputs: wfv1.Inputs{Artifacts: []wfv1.Artifact{
			{Name: "src", Path: "/src", ArtifactLocation: wfv1.ArtifactLocation{Git: &wfv1.GitArtifact{Repo: "https://example.com/r.git", SSHPrivateKeySecret: secretRef("git", "ssh")}}},
			{Name: "data", Path: "/data", ArtifactLocation: wfv1.ArtifactLocation{Plugin: &wfv1.PluginArtifact{Name: "both-plugin", Configuration: "{}", Key: "in"}}},
		}},
		Outputs: wfv1.Outputs{Artifacts: []wfv1.Artifact{
			{Name: "out", Path: "/out"},
			{Name: "report", Path: "/report", ArtifactLocation: wfv1.ArtifactLocation{Plugin: &wfv1.PluginArtifact{Name: "both-plugin", Configuration: "{}", Key: "out"}}},
		}},
	}
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "wf", Namespace: "default", UID: "u"},
		Spec:       wfv1.WorkflowSpec{Entrypoint: "t", Templates: []wfv1.Template{*tmpl}},
	}
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	controller.Config.InitlessPod = &config.InitlessPodConfig{Enabled: true}
	controller.Config.ArtifactDrivers = []config.ArtifactDriver{{Name: "both-plugin", Image: "plugin:v1"}}
	// The entrypoint index holds this map, so add to it rather than replace it.
	controller.Config.Images["plugin:v1"] = config.Image{Entrypoint: []string{"/plugin-server"}}
	// The key format has no workflow variables, which this woc hasn't set.
	controller.Config.ArtifactRepository = wfv1.ArtifactRepository{S3: &wfv1.S3ArtifactRepository{
		S3Bucket:  wfv1.S3Bucket{Bucket: "bucket", AccessKeySecret: secretRef("repo-creds", "access"), SecretKeySecret: secretRef("repo-creds", "secret")},
		KeyFormat: "{{pod.name}}",
	}}
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.artifactRepository = &controller.Config.ArtifactRepository

	podTmpl := tmpl.DeepCopy()
	pod, err := woc.createWorkflowPod(ctx, tmpl.Name, []apiv1.Container{podTmpl.Script.Container}, podTmpl, &createWorkflowPodOpts{})
	require.NoError(t, err)
	require.NotNil(t, pod)
	supervisor := containerByName(pod.Spec.Containers, common.SupervisorContainerName)
	podSidecar := containerByName(pod.Spec.Containers, common.ArtifactPluginSidecarPrefix+"both-plugin")
	require.NotNil(t, supervisor)
	require.NotNil(t, podSidecar)

	pooledTmpl := tmpl.DeepCopy()
	pooledTmpl.Pool = &wfv1.Pool{}
	for _, initless := range []bool{true, false} {
		controller.Config.InitlessPod = &config.InitlessPodConfig{Enabled: initless}
		pb := woc.newPodBuilder(ctx, tmpl.Name, []apiv1.Container{pooledTmpl.Script.Container}, pooledTmpl, &createWorkflowPodOpts{})
		spec, hash := mustPoolDeploymentSpec(ctx, t, &pb.in, poolSpecExtras{executorLogArgs: woc.getExecutorLogOpts(ctx)})
		d, err := buildPoolDeployment("default", hash, spec)
		require.NoError(t, err)
		worker := containerByName(d.Spec.Template.Spec.Containers, common.WorkerContainerName)
		require.NotNil(t, worker)
		assert.Equal(t, sortedMounts(supervisor.VolumeMounts), sortedMounts(worker.VolumeMounts), "initless=%v", initless)
		if initless {
			poolSidecar := containerByName(d.Spec.Template.Spec.Containers, podSidecar.Name)
			require.NotNil(t, poolSidecar)
			assert.Equal(t, sortedMounts(podSidecar.VolumeMounts), sortedMounts(poolSidecar.VolumeMounts))
		}
	}
}

func sortedMounts(mounts []apiv1.VolumeMount) []apiv1.VolumeMount {
	return slices.SortedFunc(slices.Values(mounts), func(a, b apiv1.VolumeMount) int {
		return cmp.Or(cmp.Compare(a.MountPath, b.MountPath), cmp.Compare(a.Name, b.Name))
	})
}

// TestPoolDeploymentMountsTaskDirs checks that worker and main mount, in every
// pool, each shared directory a task's own directories go beneath: worker
// creates all of them for every task, a container's as well as a script's.
func TestPoolDeploymentMountsTaskDirs(t *testing.T) {
	for _, tt := range poolDeploymentCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			_, d := mustPoolDeployment(ctx, t, tt.options...)
			for _, name := range []string{common.WorkerContainerName, common.MainContainerName} {
				c := containerByName(d.Spec.Template.Spec.Containers, name)
				require.NotNil(t, c)
				var paths []string
				for _, m := range c.VolumeMounts {
					paths = append(paths, m.MountPath)
				}
				assert.Subset(t, paths, []string{common.VarRunArgoPath, common.ExecutorArtifactBaseDir, common.ExecutorStagingEmptyDir}, "container %s", name)
			}
		})
	}
}

// TestPoolDeploymentNoTaskEnv checks that no per-workflow or per-task env is in
// the Deployment. Every plain value is one of an allowed set, and the
// template's valueFrom and envFrom entries are on main only.
func TestPoolDeploymentNoTaskEnv(t *testing.T) {
	allowed := []string{
		common.EnvVarContainerName,
		common.EnvVarInstanceID,
		common.EnvVarReleaseReserveSeconds,
		common.EnvVarTerminationGracePeriodSeconds,
		"EXECUTOR_SETTING", // Config.Executor.Env, on every argoexec container
		"MAIN_DEFAULT",     // Config.MainContainer.Env, on main
	}
	for _, tt := range poolDeploymentCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			_, d := mustPoolDeployment(ctx, t, tt.options...)
			for _, c := range allPodContainers(&d.Spec.Template.Spec) {
				for _, e := range c.Env {
					if e.ValueFrom == nil {
						assert.Contains(t, allowed, e.Name, "container %s", c.Name)
					}
				}
				hasToken := slices.ContainsFunc(c.Env, func(e apiv1.EnvVar) bool { return e.Name == "TOKEN" })
				if c.Name == common.MainContainerName {
					assert.True(t, hasToken, "main has the template's valueFrom env")
					assert.Len(t, c.EnvFrom, 1, "main has the template's envFrom")
					assert.Nil(t, c.Args)
				} else {
					assert.False(t, hasToken, "container %s", c.Name)
					assert.Empty(t, c.EnvFrom, "container %s", c.Name)
				}
			}
		})
	}
}

// TestPoolDeploymentLabels checks the pool's labels, reserved ones included:
// the pool and instance ID labels are on the Deployment, its selector and its
// pods, which the selector and the pool pod informer select; user metadata
// can't override them or add the workflow label.
func TestPoolDeploymentLabels(t *testing.T) {
	for _, instanceID := range []string{"", "prod"} {
		t.Run("instanceID="+instanceID, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			spec, d := mustPoolDeployment(ctx, t, func(in *podBuilderInputs, _ *poolSpecExtras) { in.execCfg.instanceID = instanceID })
			hash := d.Labels[common.LabelKeyPoolHash]
			want := map[string]string{common.LabelKeyPoolName: "builder", common.LabelKeyPoolHash: hash}
			if instanceID != "" {
				want[common.LabelKeyControllerInstanceID] = instanceID
			}
			assert.Equal(t, want, d.Labels)
			assert.Equal(t, want, d.Spec.Selector.MatchLabels)
			assert.Equal(t, "builder-"+hash[:10], d.Name)
			assert.Equal(t, spec.InitialReplicas, *d.Spec.Replicas)

			podLabels := d.Spec.Template.Labels
			for k, v := range want {
				assert.Equal(t, v, podLabels[k])
			}
			assert.NotContains(t, podLabels, common.LabelKeyWorkflow)
			if instanceID == "" {
				assert.NotContains(t, podLabels, common.LabelKeyControllerInstanceID)
			}
			assert.Equal(t, "a", podLabels["team"])
			assert.Equal(t, "b", podLabels["wf-label"])

			selector, err := metav1.LabelSelectorAsSelector(d.Spec.Selector)
			require.NoError(t, err)
			assert.True(t, selector.Matches(labels.Set(podLabels)))
			wfc := &WorkflowController{Config: config.Config{InstanceID: instanceID}}
			informerSelector, err := labels.Parse(wfc.poolPodSelector())
			require.NoError(t, err)
			assert.True(t, informerSelector.Matches(labels.Set(podLabels)), "the pool pod informer sees the pool's pods")
			other := &WorkflowController{Config: config.Config{InstanceID: "other"}}
			otherSelector, err := labels.Parse(other.poolPodSelector())
			require.NoError(t, err)
			assert.False(t, otherSelector.Matches(labels.Set(podLabels)), "another instance's pool pod informer doesn't")
		})
	}
}

// TestPoolDeploymentPodFieldsMatchPod checks that a pool pod's pod-level fields
// are those of a pod built from the same snapshot, including the boundary
// template's scheduling constraints.
func TestPoolDeploymentPodFieldsMatchPod(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tmpl := simpleContainerTemplate()
	tmpl.Pool = &wfv1.Pool{}
	tmpl.SchedulerName = "tmpl-scheduler"
	tmpl.Tolerations = []apiv1.Toleration{{Key: "tmpl", Operator: apiv1.TolerationOpExists}}
	tmpl.HostAliases = []apiv1.HostAlias{{IP: "10.0.0.1", Hostnames: []string{"tmpl"}}}
	tmpl.PodResources = &apiv1.ResourceRequirements{Limits: apiv1.ResourceList{apiv1.ResourceCPU: resource.MustParse("2")}}
	tmpl.ServiceAccountName = "tmpl-sa"
	pb := newTestPodBuilder(tmpl, []apiv1.Container{*tmpl.Container}, false, newStubDeps())
	pb.in.boundaryTemplate = &wfv1.Template{
		NodeSelector: map[string]string{"boundary": "yes"},
		Affinity:     &apiv1.Affinity{NodeAffinity: &apiv1.NodeAffinity{}},
	}
	wfSpec := pb.in.execWfSpec
	wfSpec.NodeSelector = map[string]string{"wf": "yes"}
	wfSpec.PodPriorityClassName = "wf-priority"
	wfSpec.HostAliases = []apiv1.HostAlias{{IP: "10.0.0.2", Hostnames: []string{"wf"}}}
	wfSpec.SecurityContext = &apiv1.PodSecurityContext{RunAsNonRoot: new(true)}
	wfSpec.DNSPolicy = new(apiv1.DNSNone)
	wfSpec.DNSConfig = &apiv1.PodDNSConfig{Nameservers: []string{"1.1.1.1"}}
	wfSpec.HostNetwork = new(true)
	wfSpec.ImagePullSecrets = []apiv1.LocalObjectReference{{Name: "regcred"}}
	wfSpec.ResourceClaims = []apiv1.PodResourceClaim{{Name: "gpu"}}
	wfSpec.PodMetadata = &wfv1.Metadata{Labels: map[string]string{"wf-label": "b"}, Annotations: map[string]string{"wf-note": "c"}}

	result, err := pb.build(ctx)
	require.NoError(t, err)
	pod := result.Pod
	spec, hash := mustPoolDeploymentSpec(ctx, t, &pb.in, poolSpecExtras{})
	d, err := buildPoolDeployment("ns", hash, spec)
	require.NoError(t, err)
	got := d.Spec.Template.Spec

	assert.Equal(t, pod.Spec.ServiceAccountName, got.ServiceAccountName)
	assert.Equal(t, pod.Spec.AutomountServiceAccountToken, got.AutomountServiceAccountToken)
	assert.Equal(t, pod.Spec.NodeSelector, got.NodeSelector)
	assert.Equal(t, pod.Spec.Affinity, got.Affinity)
	assert.Equal(t, pod.Spec.Tolerations, got.Tolerations)
	assert.Equal(t, pod.Spec.SchedulerName, got.SchedulerName)
	assert.Equal(t, pod.Spec.PriorityClassName, got.PriorityClassName)
	assert.Equal(t, pod.Spec.HostAliases, got.HostAliases)
	assert.Equal(t, pod.Spec.SecurityContext, got.SecurityContext)
	assert.Equal(t, pod.Spec.DNSPolicy, got.DNSPolicy)
	assert.Equal(t, pod.Spec.DNSConfig, got.DNSConfig)
	assert.Equal(t, pod.Spec.HostNetwork, got.HostNetwork)
	assert.Equal(t, pod.Spec.ImagePullSecrets, got.ImagePullSecrets)
	assert.Equal(t, pod.Spec.Resources, got.Resources)
	assert.Equal(t, pod.Spec.ResourceClaims, got.ResourceClaims)
	assert.Equal(t, "b", d.Spec.Template.Labels["wf-label"])
	assert.Equal(t, "c", d.Spec.Template.Annotations["wf-note"])
	assert.Equal(t, apiv1.RestartPolicyAlways, got.RestartPolicy)
	assert.Equal(t, new(int64(30)), got.TerminationGracePeriodSeconds)
	assert.Nil(t, got.ActiveDeadlineSeconds)
}

// TestBuildPoolDeploymentRejects checks a spec.pool the builder can't build
// fails rather than giving another Deployment.
func TestBuildPoolDeploymentRejects(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	in, x := poolDeploymentTestInputs()
	withPoolTestPlugins(in, &x)
	spec, hash := mustPoolDeploymentSpec(ctx, t, in, x)
	for _, tt := range []struct {
		name    string
		hash    string
		set     func(spec *wfv1.PoolSpec)
		wantErr string
	}{
		{name: "no spec", hash: hash, set: nil, wantErr: "no pool spec"},
		{name: "short hash", hash: "abc", set: func(*wfv1.PoolSpec) {}, wantErr: "invalid pool hash"},
		{name: "no template", hash: hash, set: func(s *wfv1.PoolSpec) { s.Template = nil }, wantErr: "no container or script template"},
		{name: "no executor image", hash: hash, set: func(s *wfv1.PoolSpec) { s.ExecutorImage = "" }, wantErr: "no executor image"},
		{name: "bad name", hash: hash, set: func(s *wfv1.PoolSpec) { s.Name = "Bad_Name" }, wantErr: "invalid pool name"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var s *wfv1.PoolSpec
			if tt.set != nil {
				s = spec.DeepCopy()
				tt.set(s)
			}
			_, err := buildPoolDeployment("ns", tt.hash, s)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
