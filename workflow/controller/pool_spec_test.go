package controller

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func secretRef(name, key string) *apiv1.SecretKeySelector {
	return &apiv1.SecretKeySelector{LocalObjectReference: apiv1.LocalObjectReference{Name: name}, Key: key}
}

// poolTestInputs is a pooled container task's pod builder snapshot and extras,
// with only what buildPoolSpec reads set.
func poolTestInputs() (*podBuilderInputs, poolSpecExtras) {
	in := &podBuilderInputs{
		tmpl: &wfv1.Template{
			Name:      "tmpl",
			Pool:      &wfv1.Pool{Name: "pool"},
			Container: &apiv1.Container{Image: "base"},
			Volumes:   []apiv1.Volume{{Name: "base-vol", VolumeSource: apiv1.VolumeSource{EmptyDir: &apiv1.EmptyDirVolumeSource{}}}},
		},
		execWfSpec: &wfv1.WorkflowSpec{},
		artifactRepository: &wfv1.ArtifactRepository{S3: &wfv1.S3ArtifactRepository{
			S3Bucket:  wfv1.S3Bucket{Bucket: "bucket", AccessKeySecret: secretRef("repo-creds", "access"), SecretKeySecret: secretRef("repo-creds", "secret")},
			KeyFormat: wfv1.DefaultArchivePattern,
		}},
		config: &config.Config{},
		execCfg: execContainerConfig{
			image:           "argoexec:base",
			imagePullPolicy: apiv1.PullIfNotPresent,
		},
	}
	return in, poolSpecExtras{executorLogArgs: []string{"--loglevel", "info"}, controllerVersion: "v4.0.0"}
}

func mustPoolHash(ctx context.Context, t *testing.T, in *podBuilderInputs, x poolSpecExtras) (*wfv1.PoolSpec, string) {
	t.Helper()
	spec, err := buildPoolSpec(ctx, in, x)
	require.NoError(t, err)
	hash, err := poolHash(spec)
	require.NoError(t, err)
	return spec, hash
}

// TestPoolHashFormat checks the hash fits the pool-hash label and its first
// 10 characters fit a Deployment name.
func TestPoolHashFormat(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	in, x := poolTestInputs()
	_, hash := mustPoolHash(ctx, t, in, x)
	assert.Regexp(t, `^[a-z2-7]{52}$`, hash)
	assert.Empty(t, validation.IsValidLabelValue(hash))
	assert.Empty(t, validation.IsDNS1123Label("pool-"+hash[:10]))
}

// TestPoolHashDeterministic checks equal inputs give one hash, however the
// template lists its artifacts, and that the spec is built from a copy.
func TestPoolHashDeterministic(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	artifacts := []wfv1.Artifact{
		{Name: "a", Path: "/a", ArtifactLocation: wfv1.ArtifactLocation{S3: &wfv1.S3Artifact{S3Bucket: wfv1.S3Bucket{AccessKeySecret: secretRef("s1", "k"), SecretKeySecret: secretRef("s1", "j")}}}},
		{Name: "b", Path: "/b", ArtifactLocation: wfv1.ArtifactLocation{Git: &wfv1.GitArtifact{Repo: "r", UsernameSecret: secretRef("s2", "u"), PasswordSecret: secretRef("s2", "p")}}},
		{Name: "c", Path: "/c", ArtifactLocation: wfv1.ArtifactLocation{GCS: &wfv1.GCSArtifact{GCSBucket: wfv1.GCSBucket{ServiceAccountKeySecret: secretRef("s0", "k")}}}},
	}
	in, x := poolTestInputs()
	in.tmpl.Inputs.Artifacts = artifacts
	spec, want := mustPoolHash(ctx, t, in, x)
	assert.Equal(t, []apiv1.SecretKeySelector{
		*secretRef("repo-creds", "access"), *secretRef("repo-creds", "secret"),
		*secretRef("s0", "k"), *secretRef("s1", "j"), *secretRef("s1", "k"), *secretRef("s2", "p"), *secretRef("s2", "u"),
	}, spec.ArtifactSecrets)

	for range 20 {
		shuffled, sx := poolTestInputs()
		shuffled.tmpl.Inputs.Artifacts = []wfv1.Artifact{artifacts[2], artifacts[0], artifacts[1]}
		_, hash := mustPoolHash(ctx, t, shuffled, sx)
		require.Equal(t, want, hash)
	}

	spec.Template.Container.Image = "changed"
	assert.Equal(t, "base", in.tmpl.Container.Image)
}

// TestPoolSpecTaskArtifactSecretsAndPlugins checks that a task whose artifacts
// use a secret or plugin the repository doesn't gets a pool spec containing it,
// and a different hash.
func TestPoolSpecTaskArtifactSecretsAndPlugins(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	plugin := func(name string) *wfv1.PluginArtifact {
		return &wfv1.PluginArtifact{Name: wfv1.ArtifactPluginName(name), Configuration: "{}", Key: "k"}
	}
	in, x := poolTestInputs()
	in.config.ArtifactDrivers = []config.ArtifactDriver{
		{Name: "in-plugin", Image: "in-plugin:v1", ConnectionTimeoutSeconds: 7},
		{Name: "out-plugin", Image: "out-plugin:v1"},
	}
	_, base := mustPoolHash(ctx, t, in, x)

	for _, tt := range []struct {
		name        string
		set         func(tmpl *wfv1.Template)
		wantSecret  *apiv1.SecretKeySelector
		wantPlugins []wfv1.PoolArtifactPlugin
	}{
		{
			name: "input artifact secret",
			set: func(tmpl *wfv1.Template) {
				tmpl.Inputs.Artifacts = []wfv1.Artifact{{Name: "in", Path: "/in", ArtifactLocation: wfv1.ArtifactLocation{Git: &wfv1.GitArtifact{Repo: "r", SSHPrivateKeySecret: secretRef("git-key", "ssh")}}}}
			},
			wantSecret: secretRef("git-key", "ssh"),
		},
		{
			name: "output artifact secret",
			set: func(tmpl *wfv1.Template) {
				tmpl.Outputs.Artifacts = []wfv1.Artifact{{Name: "out", Path: "/out", ArtifactLocation: wfv1.ArtifactLocation{Azure: &wfv1.AzureArtifact{AzureBlobContainer: wfv1.AzureBlobContainer{AccountKeySecret: secretRef("azure", "key")}}}}}
			},
			wantSecret: secretRef("azure", "key"),
		},
		{
			name: "archive location secret",
			set: func(tmpl *wfv1.Template) {
				tmpl.ArchiveLocation = &wfv1.ArtifactLocation{OSS: &wfv1.OSSArtifact{OSSBucket: wfv1.OSSBucket{AccessKeySecret: secretRef("oss", "a")}}}
			},
			wantSecret: secretRef("oss", "a"),
		},
		{
			name: "input artifact plugin",
			set: func(tmpl *wfv1.Template) {
				tmpl.Inputs.Artifacts = []wfv1.Artifact{{Name: "in", Path: "/in", ArtifactLocation: wfv1.ArtifactLocation{Plugin: plugin("in-plugin")}}}
			},
			wantPlugins: []wfv1.PoolArtifactPlugin{{Name: "in-plugin", Image: "in-plugin:v1", ConnectionTimeoutSeconds: 7}},
		},
		{
			name: "output artifact plugin",
			set: func(tmpl *wfv1.Template) {
				tmpl.Outputs.Artifacts = []wfv1.Artifact{{Name: "out", Path: "/out", ArtifactLocation: wfv1.ArtifactLocation{Plugin: plugin("out-plugin")}}}
			},
			wantPlugins: []wfv1.PoolArtifactPlugin{{Name: "out-plugin", Image: "out-plugin:v1"}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			in, x := poolTestInputs()
			in.config.ArtifactDrivers = []config.ArtifactDriver{
				{Name: "in-plugin", Image: "in-plugin:v1", ConnectionTimeoutSeconds: 7},
				{Name: "out-plugin", Image: "out-plugin:v1"},
			}
			tt.set(in.tmpl)
			spec, hash := mustPoolHash(ctx, t, in, x)
			if tt.wantSecret != nil {
				assert.Contains(t, spec.ArtifactSecrets, *tt.wantSecret)
				assert.Contains(t, spec.ArtifactSecrets, *secretRef("repo-creds", "access"), "the repository's secrets stay")
			}
			assert.Equal(t, tt.wantPlugins, spec.ArtifactPlugins)
			assert.NotEqual(t, base, hash)
		})
	}

	t.Run("unconfigured plugin", func(t *testing.T) {
		in, x := poolTestInputs()
		in.tmpl.Inputs.Artifacts = []wfv1.Artifact{{Name: "in", Path: "/in", ArtifactLocation: wfv1.ArtifactLocation{Plugin: plugin("missing")}}}
		_, err := buildPoolSpec(logging.TestContext(t.Context()), in, x)
		assert.ErrorContains(t, err, "artifact driver missing not found")
	})

	t.Run("plugin repository", func(t *testing.T) {
		in, x := poolTestInputs()
		in.config.ArtifactDrivers = []config.ArtifactDriver{{Name: "repo-plugin", Image: "repo-plugin:v1"}}
		in.artifactRepository = &wfv1.ArtifactRepository{Plugin: &wfv1.PluginArtifactRepository{Name: "repo-plugin", Configuration: "{}"}}
		spec, _ := mustPoolHash(ctx, t, in, x)
		assert.Equal(t, []wfv1.PoolArtifactPlugin{{Name: "repo-plugin", Image: "repo-plugin:v1"}}, spec.ArtifactPlugins,
			"a task may archive its logs or outputs to the repository")
		assert.Empty(t, spec.ArtifactSecrets)
	})
}

// TestPoolSpecNumbers checks pool.initialReplicas and pool.retentionSeconds
// are read once substituted, as numbers or strings, and that one that isn't
// a whole number fails the build rather than defaulting.
func TestPoolSpecNumbers(t *testing.T) {
	for _, tt := range []struct {
		name     string
		pool     wfv1.Pool
		replicas int32
		want     string
	}{
		{name: "a number", pool: wfv1.Pool{InitialReplicas: new(intstr.FromInt32(4))}, replicas: 4},
		{name: "a substituted string", pool: wfv1.Pool{InitialReplicas: new(intstr.FromString("5"))}, replicas: 5},
		{name: "unset", pool: wfv1.Pool{}, replicas: 1},
		{name: "not substituted", pool: wfv1.Pool{InitialReplicas: new(intstr.FromString("{{inputs.parameters.replicas}}"))},
			want: "pool.initialReplicas '{{inputs.parameters.replicas}}' is not a whole number"},
		{name: "negative retention", pool: wfv1.Pool{RetentionSeconds: new(intstr.FromInt32(-1))}, want: "pool.retentionSeconds must not be negative"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			in, x := poolTestInputs()
			in.tmpl.Pool = &tt.pool
			spec, err := buildPoolSpec(ctx, in, x)
			if tt.want != "" {
				require.ErrorContains(t, err, tt.want)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.replicas, spec.InitialReplicas)
		})
	}
}

// TestPoolSpecContents checks the resolved and controller fields of spec.pool,
// and that the repository's key format stays as written.
func TestPoolSpecContents(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	in, x := poolTestInputs()
	in.tmpl.Container.Env = []apiv1.EnvVar{{Name: "PLAIN", Value: "v"}}
	in.tmpl.Container.Args = []string{"per-task"}
	in.tmpl.Pool = &wfv1.Pool{Name: "pool", InitialReplicas: new(intstr.FromInt32(3)), RetentionSeconds: new(intstr.FromString("60"))}
	in.tmpl.ServiceAccountName = "tmpl-sa"
	in.execWfSpec.ServiceAccountName = "wf-sa"
	in.execWfSpec.Executor = &wfv1.ExecutorConfig{ServiceAccountName: "exec-sa"}
	in.execWfSpec.AutomountServiceAccountToken = new(false)
	in.execWfSpec.Entrypoint = "main"
	in.execCfg.instanceID = "instance"
	in.config.InitlessPod = &config.InitlessPodConfig{Enabled: true}
	in.config.HotWorkers = &config.HotWorkersConfig{TerminationGracePeriodSeconds: new(int64(40)), ReleaseReserveSeconds: new(int64(15))}
	in.execCfg.kubeConfig = &config.KubeConfig{SecretName: "kc", SecretKey: "config"}
	x.executorTokenSecretName = "exec-sa-token"

	spec, _ := mustPoolHash(ctx, t, in, x)
	assert.Equal(t, "pool", spec.Name)
	assert.Equal(t, int32(3), spec.InitialReplicas)
	assert.Equal(t, int64(60), spec.RetentionSeconds)
	assert.Equal(t, &wfv1.Template{
		Container: &apiv1.Container{Image: "base"},
		Volumes:   in.tmpl.Volumes,
	}, spec.Template, "per-task and resolved fields are cleared")
	assert.Nil(t, spec.WorkflowSpec)
	assert.Equal(t, "tmpl-sa", spec.ServiceAccountName)
	assert.Equal(t, new(false), spec.AutomountServiceAccountToken)
	assert.Equal(t, "exec-sa-token", spec.ExecutorServiceAccountTokenSecretName)
	assert.Equal(t, "{{workflow.name}}/{{node.taskName}}", spec.ArtifactRepository.S3.KeyFormat)
	assert.Equal(t, "instance", spec.InstanceID)
	assert.Equal(t, "v4.0.0", spec.ControllerVersion)
	assert.Nil(t, spec.Executor)
	assert.Nil(t, spec.MainContainer)
	assert.Equal(t, "argoexec:base", spec.ExecutorImage)
	assert.Equal(t, apiv1.PullIfNotPresent, spec.ExecutorImagePullPolicy)
	assert.Equal(t, []string{"--loglevel", "info"}, spec.ExecutorLogArgs)
	assert.Equal(t, &wfv1.PoolKubeConfig{SecretName: "kc", SecretKey: "config", VolumeName: "kubeconfig", MountPath: "/kube/config"}, spec.KubeConfig)
	assert.True(t, spec.InitlessPod)
	assert.Equal(t, int64(40), spec.TerminationGracePeriodSeconds)
	assert.Equal(t, int64(15), spec.ReleaseReserveSeconds)

	t.Run("defaults", func(t *testing.T) {
		in, x := poolTestInputs()
		in.tmpl.Pool = &wfv1.Pool{}
		spec, _ := mustPoolHash(ctx, t, in, x)
		assert.Equal(t, "tmpl", spec.Name)
		assert.Equal(t, int32(1), spec.InitialReplicas)
		assert.Equal(t, int64(600), spec.RetentionSeconds)
		assert.Equal(t, int64(30), spec.TerminationGracePeriodSeconds)
		assert.Equal(t, int64(10), spec.ReleaseReserveSeconds)
	})

	t.Run("script", func(t *testing.T) {
		in, x := poolTestInputs()
		in.tmpl.Script = &wfv1.ScriptTemplate{Container: apiv1.Container{Image: "python", Args: []string{"/argo/staging/script"}}, Source: "print(1)"}
		in.tmpl.Container = nil
		spec, _ := mustPoolHash(ctx, t, in, x)
		assert.Equal(t, &wfv1.ScriptTemplate{Container: apiv1.Container{Image: "python"}}, spec.Template.Script)
	})
}

func TestBuildPoolSpecErrors(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	for _, tt := range []struct {
		name string
		set  func(in *podBuilderInputs, x *poolSpecExtras)
		want string
	}{
		{"not pooled", func(in *podBuilderInputs, _ *poolSpecExtras) { in.tmpl.Pool = nil }, `template "tmpl" is not pooled`},
		{"no type", func(in *podBuilderInputs, _ *poolSpecExtras) { in.tmpl.Container = nil }, "only container and script templates can be pooled"},
		{"executor account without token", func(in *podBuilderInputs, _ *poolSpecExtras) {
			in.tmpl.Executor = &wfv1.ExecutorConfig{ServiceAccountName: "exec"}
		}, `no token Secret for executor service account "exec"`},
		{"automount off without executor account", func(in *podBuilderInputs, _ *poolSpecExtras) {
			in.tmpl.AutomountServiceAccountToken = new(false)
		}, "executor.serviceAccountName must not be empty"},
		{"unknown volume", func(in *podBuilderInputs, _ *poolSpecExtras) {
			in.tmpl.Container.VolumeMounts = []apiv1.VolumeMount{{Name: "claim", MountPath: "/claim"}}
		}, "volume 'claim' is neither the template's nor in the workflow's volumes"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			in, x := poolTestInputs()
			tt.set(in, &x)
			_, err := buildPoolSpec(ctx, in, x)
			assert.ErrorContains(t, err, tt.want)
		})
	}
}

// TestPoolSpecExecutorImageFromCLIFlag checks the executor image and pull
// policy reach spec.pool from the controller's command-line flags, ahead of
// Config.Executor, through the pod builder's snapshot.
func TestPoolSpecExecutorImageFromCLIFlag(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	controller.Config.Executor = &apiv1.Container{Image: "argoexec:config", ImagePullPolicy: apiv1.PullNever}
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "wf", UID: "u"},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "t",
			Templates:  []wfv1.Template{{Name: "t", Pool: &wfv1.Pool{}, Container: &apiv1.Container{Image: "busybox", Command: []string{"true"}}}},
		},
	}
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	build := func() (*wfv1.PoolSpec, string) {
		pb := woc.newPodBuilder(ctx, "wf", nil, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
		return mustPoolHash(ctx, t, &pb.in, poolSpecExtras{})
	}

	spec, configHash := build()
	assert.Equal(t, "argoexec:config", spec.ExecutorImage)
	assert.Equal(t, apiv1.PullNever, spec.ExecutorImagePullPolicy)

	controller.cliExecutorImage = "argoexec:flag"
	spec, flagHash := build()
	assert.Equal(t, "argoexec:flag", spec.ExecutorImage)
	assert.NotEqual(t, configHash, flagHash)

	controller.cliExecutorImagePullPolicy = string(apiv1.PullAlways)
	spec, pullHash := build()
	assert.Equal(t, apiv1.PullAlways, spec.ExecutorImagePullPolicy)
	assert.NotEqual(t, flagHash, pullHash)
}
