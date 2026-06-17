package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// TestBuildPluginSidecarsDedup verifies that a plugin referenced by both
// inputs and outputs produces exactly one sidecar in init-less mode.
func TestBuildPluginSidecarsDedup(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	controller.Config.InitlessPod = &config.InitlessPodConfig{Enabled: true}
	controller.Config.ArtifactDrivers = []config.ArtifactDriver{
		{Name: "shared-plugin", Image: "busybox"},
		{Name: "only-input", Image: "alpine"},
		{Name: "only-output", Image: "alpine"},
	}
	controller.Config.Images = map[string]config.Image{
		"busybox": {Entrypoint: []string{"/plugin-server"}},
		"alpine":  {Entrypoint: []string{"/plugin-server"}},
	}

	tmpl := &wfv1.Template{
		Name: "t",
		Container: &apiv1.Container{
			Image:   "hello-world",
			Command: []string{"echo", "hi"},
		},
		Inputs: wfv1.Inputs{
			Artifacts: []wfv1.Artifact{
				{Name: "in1", Path: "/tmp/in1", ArtifactLocation: wfv1.ArtifactLocation{
					Plugin: &wfv1.PluginArtifact{Name: "shared-plugin"},
				}},
				{Name: "in2", Path: "/tmp/in2", ArtifactLocation: wfv1.ArtifactLocation{
					Plugin: &wfv1.PluginArtifact{Name: "only-input"},
				}},
			},
		},
		Outputs: wfv1.Outputs{
			Artifacts: []wfv1.Artifact{
				{Name: "out1", Path: "/tmp/out1", ArtifactLocation: wfv1.ArtifactLocation{
					Plugin: &wfv1.PluginArtifact{Name: "shared-plugin"},
				}},
				{Name: "out2", Path: "/tmp/out2", ArtifactLocation: wfv1.ArtifactLocation{
					Plugin: &wfv1.PluginArtifact{Name: "only-output"},
				}},
			},
		},
	}

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "wf", UID: "u"},
		Spec:       wfv1.WorkflowSpec{Entrypoint: "t", Templates: []wfv1.Template{*tmpl}},
	}
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	sidecars, inputPlugins, err := woc.buildPluginSidecars(ctx, tmpl)
	require.NoError(t, err)

	// Expect 3 unique plugins: shared-plugin, only-input, only-output — not 4.
	require.Len(t, sidecars, 3, "plugin in both inputs and outputs must dedup to one sidecar")

	names := map[string]bool{}
	for _, ctr := range sidecars {
		names[ctr.Name] = true
	}
	assert.True(t, names[common.ArtifactPluginSidecarPrefix+"shared-plugin"])
	assert.True(t, names[common.ArtifactPluginSidecarPrefix+"only-input"])
	assert.True(t, names[common.ArtifactPluginSidecarPrefix+"only-output"])

	// Input plugins list (what supervisor will invoke Load on) includes shared + only-input.
	inputNames := map[wfv1.ArtifactPluginName]bool{}
	for _, n := range inputPlugins {
		inputNames[n] = true
	}
	assert.True(t, inputNames["shared-plugin"])
	assert.True(t, inputNames["only-input"])
	assert.False(t, inputNames["only-output"], "only-output must not appear in input plugin list")
}

// TestBuildPluginSidecarsEmpty handles the no-plugins case.
func TestBuildPluginSidecarsEmpty(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	controller.Config.InitlessPod = &config.InitlessPodConfig{Enabled: true}

	tmpl := &wfv1.Template{
		Name: "t",
		Container: &apiv1.Container{
			Image:   "hello-world",
			Command: []string{"echo", "hi"},
		},
	}
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "wf", UID: "u"},
		Spec:       wfv1.WorkflowSpec{Entrypoint: "t", Templates: []wfv1.Template{*tmpl}},
	}
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	sidecars, inputPlugins, err := woc.buildPluginSidecars(ctx, tmpl)
	require.NoError(t, err)
	assert.Empty(t, sidecars)
	assert.Empty(t, inputPlugins)
}

// TestNewSupervisorContainer verifies shape of the supervisor container.
func TestNewSupervisorContainer(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	tmpl := &wfv1.Template{
		Name: "t",
		Container: &apiv1.Container{
			Image:   "hello-world",
			Command: []string{"echo", "hi"},
		},
	}
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "wf", UID: "u"},
		Spec:       wfv1.WorkflowSpec{Entrypoint: "t", Templates: []wfv1.Template{*tmpl}},
	}
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	ctr := woc.newSupervisorContainer(ctx, tmpl)

	assert.Equal(t, common.SupervisorContainerName, ctr.Name)
	require.NotEmpty(t, ctr.Command)
	assert.Equal(t, "argoexec", ctr.Command[0])
	assert.Equal(t, "supervisor", ctr.Command[1])
}

// TestBuildArgoBinVolume verifies the image volume source uses the executor
// image and that the controller's executor pull-policy flows through to the
// ImageVolumeSource so users can pin pull behavior across the binary mount
// and the supervisor/wait runtime.
func TestBuildArgoBinVolume(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	controller.cliExecutorImage = "quay.io/argoproj/argoexec:testtag"
	controller.cliExecutorImagePullPolicy = string(apiv1.PullAlways)

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "wf", UID: "u"},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "t",
			Templates: []wfv1.Template{{
				Name: "t",
				Container: &apiv1.Container{
					Image:   "hello-world",
					Command: []string{"echo", "hi"},
				},
			}},
		},
	}
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	vol := woc.buildArgoBinVolume()
	assert.Equal(t, argoBinVolumeName, vol.Name)
	require.NotNil(t, vol.Image, "volume source must be Image (KEP-4639)")
	assert.Equal(t, "quay.io/argoproj/argoexec:testtag", vol.Image.Reference)
	assert.Equal(t, apiv1.PullAlways, vol.Image.PullPolicy,
		"pull policy must be threaded from controller config into ImageVolumeSource")
}

// TestCreateWorkflowPod_InitlessShape runs the full createWorkflowPod pipeline
// with initlessPod enabled and asserts the pod-level invariants the proposal
// promises: zero init containers, supervisor container with the right command,
// argoexec-bin image volume on the pod, /argo-bin mounted onto main, and
// ARGO_WAIT_FOR_READY=true set on main.
func TestCreateWorkflowPod_InitlessShape(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	controller.Config.InitlessPod = &config.InitlessPodConfig{Enabled: true}
	controller.cliExecutorImage = "quay.io/argoproj/argoexec:initless-test"

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	tmpl := &woc.execWf.Spec.Templates[0]
	mainCtr := tmpl.Container.DeepCopy()
	pod, err := woc.createWorkflowPod(ctx, tmpl.Name, []apiv1.Container{*mainCtr}, tmpl, &createWorkflowPodOpts{})
	require.NoError(t, err)
	require.NotNil(t, pod)

	// 1. Zero init containers.
	assert.Empty(t, pod.Spec.InitContainers, "init-less pods must have zero init containers")

	// 2. Supervisor container present with `argoexec supervisor` command.
	var supervisor, main *apiv1.Container
	for i, c := range pod.Spec.Containers {
		switch c.Name {
		case common.SupervisorContainerName:
			supervisor = &pod.Spec.Containers[i]
		case common.MainContainerName:
			main = &pod.Spec.Containers[i]
		}
	}
	require.NotNil(t, supervisor, "supervisor container must exist")
	require.NotNil(t, main, "main container must exist")
	require.GreaterOrEqual(t, len(supervisor.Command), 2)
	assert.Equal(t, "argoexec", supervisor.Command[0])
	assert.Equal(t, "supervisor", supervisor.Command[1])

	// 3. Legacy wait container is NOT present.
	for _, c := range pod.Spec.Containers {
		assert.NotEqual(t, common.WaitContainerName, c.Name, "wait container must not appear in init-less mode")
	}

	// 4. argoexec-bin image volume present with the executor image reference.
	var argoBinVolume *apiv1.Volume
	for i, v := range pod.Spec.Volumes {
		if v.Name == argoBinVolumeName {
			argoBinVolume = &pod.Spec.Volumes[i]
			break
		}
	}
	require.NotNil(t, argoBinVolume, "argoexec-bin image volume must be on the pod")
	require.NotNil(t, argoBinVolume.Image, "volume source must be Image (KEP-4639)")
	assert.Equal(t, "quay.io/argoproj/argoexec:initless-test", argoBinVolume.Image.Reference)

	// 5. /argo-bin mount on main, read-only.
	var binMount *apiv1.VolumeMount
	for i, vm := range main.VolumeMounts {
		if vm.Name == argoBinVolumeName {
			binMount = &main.VolumeMounts[i]
			break
		}
	}
	require.NotNil(t, binMount, "main must have /argo-bin mount in init-less mode")
	assert.Equal(t, argoBinMountPath, binMount.MountPath)
	assert.True(t, binMount.ReadOnly, "/argo-bin must be read-only on main")

	// 6. main's emissary command prepended with the /argo-bin path.
	require.NotEmpty(t, main.Command)
	assert.Equal(t, argoBinMountPath+"/bin/argoexec", main.Command[0], "main's emissary must exec from the image volume path (/bin/argoexec inside the argoexec image)")

	// 7. ARGO_WAIT_FOR_READY=true env on main.
	var waitReadyEnv *apiv1.EnvVar
	for i, e := range main.Env {
		if e.Name == common.EnvVarWaitForReady {
			waitReadyEnv = &main.Env[i]
			break
		}
	}
	require.NotNil(t, waitReadyEnv, "main must have ARGO_WAIT_FOR_READY env in init-less mode")
	assert.Equal(t, "true", waitReadyEnv.Value)

	// 8. ARGO_TEMPLATE set on supervisor so it can write /var/run/argo/template.
	var tmplEnv *apiv1.EnvVar
	for i, e := range supervisor.Env {
		if e.Name == common.EnvVarTemplate {
			tmplEnv = &supervisor.Env[i]
			break
		}
	}
	require.NotNil(t, tmplEnv, "supervisor must have ARGO_TEMPLATE env")
	assert.NotEmpty(t, tmplEnv.Value)
}

// TestCreateWorkflowPod_LegacyShape asserts that with initlessPod disabled,
// the legacy layout is unchanged: init container present, wait container
// present, no argoexec-bin volume, no ARGO_WAIT_FOR_READY on main.
func TestCreateWorkflowPod_LegacyShape(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	// InitlessPod not set → default off.

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	tmpl := &woc.execWf.Spec.Templates[0]
	mainCtr := tmpl.Container.DeepCopy()
	pod, err := woc.createWorkflowPod(ctx, tmpl.Name, []apiv1.Container{*mainCtr}, tmpl, &createWorkflowPodOpts{})
	require.NoError(t, err)
	require.NotNil(t, pod)

	// Init container present.
	require.NotEmpty(t, pod.Spec.InitContainers, "legacy pods must have an init container")
	assert.Equal(t, common.InitContainerName, pod.Spec.InitContainers[0].Name)

	// Wait container present, supervisor not.
	var hasWait, hasSupervisor bool
	for _, c := range pod.Spec.Containers {
		if c.Name == common.WaitContainerName {
			hasWait = true
		}
		if c.Name == common.SupervisorContainerName {
			hasSupervisor = true
		}
	}
	assert.True(t, hasWait, "legacy pod must have wait container")
	assert.False(t, hasSupervisor, "legacy pod must not have supervisor container")

	// No argoexec-bin image volume.
	for _, v := range pod.Spec.Volumes {
		assert.NotEqual(t, argoBinVolumeName, v.Name, "legacy pod must not have argoexec-bin image volume")
	}

	// main must NOT have ARGO_WAIT_FOR_READY.
	for _, c := range pod.Spec.Containers {
		if c.Name == common.MainContainerName {
			for _, e := range c.Env {
				assert.NotEqual(t, common.EnvVarWaitForReady, e.Name, "main must not have ARGO_WAIT_FOR_READY in legacy mode")
			}
		}
	}
}

// helloWorldWithInputArtifactWf exercises the init-less input-artifact path
// (whole-volume mount on main, mount on supervisor) — the legacy SubPath
// per-artifact bind-mount scheme is replaced by emissary symlinks post-ready.
var helloWorldWithInputArtifactWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-with-artifact
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    inputs:
      artifacts:
      - name: in1
        path: /tmp/in1
        raw:
          data: hello
    container:
      image: docker/whalesay:latest
      command: [cowsay]
`

// TestCreateWorkflowPod_InitlessInputArtifacts asserts the central invariants
// for input artifacts in init-less mode: main gets the whole-volume mount at
// ExecutorArtifactBaseDir (no per-artifact SubPath), and supervisor also
// mounts the same volume so it can populate it pre-ready.
func TestCreateWorkflowPod_InitlessInputArtifacts(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWithInputArtifactWf)
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	controller.Config.InitlessPod = &config.InitlessPodConfig{Enabled: true}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	tmpl := &woc.execWf.Spec.Templates[0]
	mainCtr := tmpl.Container.DeepCopy()
	pod, err := woc.createWorkflowPod(ctx, tmpl.Name, []apiv1.Container{*mainCtr}, tmpl, &createWorkflowPodOpts{})
	require.NoError(t, err)
	require.NotNil(t, pod)

	var main, supervisor *apiv1.Container
	for i, c := range pod.Spec.Containers {
		switch c.Name {
		case common.MainContainerName:
			main = &pod.Spec.Containers[i]
		case common.SupervisorContainerName:
			supervisor = &pod.Spec.Containers[i]
		}
	}
	require.NotNil(t, main)
	require.NotNil(t, supervisor)

	// main: whole input-artifacts volume mounted at ExecutorArtifactBaseDir,
	// and NO SubPath bind mount for the per-artifact path /tmp/in1.
	var mainArtMount *apiv1.VolumeMount
	for i, vm := range main.VolumeMounts {
		if vm.Name == "input-artifacts" {
			mainArtMount = &main.VolumeMounts[i]
		}
		assert.NotEqual(t, "/tmp/in1", vm.MountPath, "init-less mode must not use per-artifact SubPath mounts on main")
	}
	require.NotNil(t, mainArtMount, "main must have input-artifacts volume mounted")
	assert.Equal(t, common.ExecutorArtifactBaseDir, mainArtMount.MountPath)
	assert.False(t, mainArtMount.ReadOnly, "input-artifacts mount must be writable so user code can mutate files in place")

	// supervisor: same input-artifacts volume mounted at ExecutorArtifactBaseDir
	// so supervisor can populate it before signaling ready. (A second mirror
	// of main's mounts at /mainctrfs/argo/inputs/artifacts is also added by
	// addOutputArtifactsVolumes for the overlap case — we ignore that here
	// and verify the canonical mount exists.)
	hasSupArtMount := false
	for _, vm := range supervisor.VolumeMounts {
		if vm.Name == "input-artifacts" && vm.MountPath == common.ExecutorArtifactBaseDir {
			hasSupArtMount = true
			break
		}
	}
	assert.True(t, hasSupArtMount, "supervisor must have input-artifacts mount at %s in init-less mode", common.ExecutorArtifactBaseDir)
}

// resourceNoLogsInitlessWf is a Resource template without SaveLogsAsArtifact;
// in init-less mode this is the no-supervisor case where the template arrives
// at main via the ARGO_TEMPLATE env var instead of /var/run/argo/template.
var resourceNoLogsInitlessWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: resource-init-less
spec:
  archiveLogs: false
  entrypoint: get-pod
  templates:
  - name: get-pod
    resource:
      action: get
      manifest: |
        apiVersion: v1
        kind: Pod
        metadata:
          name: example
`

// TestCreateWorkflowPod_InitlessNoAuxContainer asserts that templates without
// an auxiliary container in init-less mode (resource templates without
// SaveLogsAsArtifact; data templates) carry ARGO_TEMPLATE directly on main —
// the emissary readTemplate fallback reads it from the env var rather than
// from the supervisor-written file.
func TestCreateWorkflowPod_InitlessNoAuxContainer(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(resourceNoLogsInitlessWf)
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	controller.Config.InitlessPod = &config.InitlessPodConfig{Enabled: true}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	tmpl := &woc.execWf.Spec.Templates[0]
	mainCtr := apiv1.Container{
		Name:    common.MainContainerName,
		Image:   "argoproj/argoexec:v3",
		Command: []string{"argoexec", "resource", "get"},
	}
	pod, err := woc.createWorkflowPod(ctx, tmpl.Name, []apiv1.Container{mainCtr}, tmpl, &createWorkflowPodOpts{})
	require.NoError(t, err)
	require.NotNil(t, pod)

	// No supervisor in this mode — there's no pre-main work and no post-main
	// outputs to capture.
	for _, c := range pod.Spec.Containers {
		assert.NotEqual(t, common.SupervisorContainerName, c.Name,
			"resource-without-logs templates must not schedule a supervisor")
	}

	// main carries ARGO_TEMPLATE inline so emissary readTemplate falls back to env.
	var main *apiv1.Container
	for i, c := range pod.Spec.Containers {
		if c.Name == common.MainContainerName {
			main = &pod.Spec.Containers[i]
			break
		}
	}
	require.NotNil(t, main)
	var tmplEnv *apiv1.EnvVar
	for i, e := range main.Env {
		if e.Name == common.EnvVarTemplate {
			tmplEnv = &main.Env[i]
			break
		}
	}
	require.NotNil(t, tmplEnv, "main must carry ARGO_TEMPLATE env var when there's no supervisor")
	assert.NotEmpty(t, tmplEnv.Value)
}

// resourceManifestFromInitlessWf is a Resource template that is NOT archiving
// logs but sources its manifest from an input artifact. `argoexec resource`
// reads that artifact from disk, so in init-less mode it needs a supervisor to
// download it during pre-main — even though, without manifestFrom, this same
// template would run with no aux container.
var resourceManifestFromInitlessWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: resource-manifestfrom-init-less
spec:
  archiveLogs: false
  entrypoint: create-pod
  templates:
  - name: create-pod
    inputs:
      artifacts:
      - name: manifest
        path: /tmp/manifest.yaml
        raw:
          data: |
            apiVersion: v1
            kind: Pod
            metadata:
              generateName: example-
            spec:
              containers:
              - name: c
                image: argoproj/argosay:v2
              restartPolicy: Never
    resource:
      action: create
      manifestFrom:
        artifact:
          name: manifest
`

// TestCreateWorkflowPod_InitlessResourceManifestFrom is the regression test for
// the gap where a manifestFrom resource without log archiving got no supervisor
// in init-less mode, leaving its manifest artifact undownloaded. It must now
// schedule a supervisor and make main wait for readiness before exec'ing
// argoexec resource.
func TestCreateWorkflowPod_InitlessResourceManifestFrom(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(resourceManifestFromInitlessWf)
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	controller.Config.InitlessPod = &config.InitlessPodConfig{Enabled: true}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	tmpl := &woc.execWf.Spec.Templates[0]
	mainCtr := apiv1.Container{
		Name:    common.MainContainerName,
		Image:   "argoproj/argoexec:v3",
		Command: []string{"argoexec", "resource", "create"},
	}
	pod, err := woc.createWorkflowPod(ctx, tmpl.Name, []apiv1.Container{mainCtr}, tmpl, &createWorkflowPodOpts{})
	require.NoError(t, err)
	require.NotNil(t, pod)

	var main, supervisor *apiv1.Container
	for i, c := range pod.Spec.Containers {
		switch c.Name {
		case common.SupervisorContainerName:
			supervisor = &pod.Spec.Containers[i]
		case common.MainContainerName:
			main = &pod.Spec.Containers[i]
		}
	}
	require.NotNil(t, supervisor, "manifestFrom resource must schedule a supervisor to stage the manifest artifact")
	require.NotNil(t, main)

	// main must block on supervisor readiness so the manifest is staged before
	// argoexec resource reads it.
	var waitReadyEnv *apiv1.EnvVar
	for i, e := range main.Env {
		if e.Name == common.EnvVarWaitForReady {
			waitReadyEnv = &main.Env[i]
			break
		}
	}
	require.NotNil(t, waitReadyEnv, "main must wait for supervisor readiness so the manifest artifact is staged first")
	assert.Equal(t, "true", waitReadyEnv.Value)

	// The supervisor downloads input artifacts, so the input-artifacts volume
	// must be mounted on it.
	hasInputMount := false
	for _, vm := range supervisor.VolumeMounts {
		if vm.Name == "input-artifacts" && vm.MountPath == common.ExecutorArtifactBaseDir {
			hasInputMount = true
			break
		}
	}
	assert.True(t, hasInputMount, "supervisor must mount the input-artifacts volume to download the manifest")
}

// TestCreateWorkflowPod_LegacyResourceManifestFrom guards that the manifestFrom
// supervisor trigger is init-less-only: in legacy mode the same template gets
// its manifest from the init container and must not gain a wait container.
func TestCreateWorkflowPod_LegacyResourceManifestFrom(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(resourceManifestFromInitlessWf)
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	// InitlessPod not set → legacy.

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	tmpl := &woc.execWf.Spec.Templates[0]
	mainCtr := apiv1.Container{
		Name:    common.MainContainerName,
		Image:   "argoproj/argoexec:v3",
		Command: []string{"argoexec", "resource", "create"},
	}
	pod, err := woc.createWorkflowPod(ctx, tmpl.Name, []apiv1.Container{mainCtr}, tmpl, &createWorkflowPodOpts{})
	require.NoError(t, err)
	require.NotNil(t, pod)

	// Init container stages the manifest artifact in legacy mode.
	require.NotEmpty(t, pod.Spec.InitContainers, "legacy mode must stage the manifest via an init container")
	// No aux container: resource-without-logs has none, and the manifestFrom
	// trigger must not fire in legacy mode.
	for _, c := range pod.Spec.Containers {
		assert.NotEqual(t, common.WaitContainerName, c.Name, "legacy manifestFrom resource without logs must not gain a wait container")
		assert.NotEqual(t, common.SupervisorContainerName, c.Name)
	}
}

// artifactPathAncestorOfMountWf has an input artifact whose path (/data) is an
// ancestor of a user volume mount (/data/shared). In init-less mode staging the
// artifact would os.RemoveAll(/data) and recurse into the mounted volume, so the
// controller must reject this configuration.
var artifactPathAncestorOfMountWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: artifact-ancestor-of-mount
spec:
  entrypoint: main
  volumes:
  - name: shared
    emptyDir: {}
  templates:
  - name: main
    inputs:
      artifacts:
      - name: in
        path: /data
        raw:
          data: hi
    container:
      image: argoproj/argosay:v2
      command: [sh, -c, "true"]
      volumeMounts:
      - name: shared
        mountPath: /data/shared
`

// TestCreateWorkflowPod_InitlessRejectsArtifactPathAncestorOfMount asserts the
// controller refuses to build a pod where an input artifact path is an ancestor
// of a mounted volume in init-less mode (data-loss guard).
func TestCreateWorkflowPod_InitlessRejectsArtifactPathAncestorOfMount(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(artifactPathAncestorOfMountWf)
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	controller.Config.InitlessPod = &config.InitlessPodConfig{Enabled: true}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	tmpl := &woc.execWf.Spec.Templates[0]
	mainCtr := tmpl.Container.DeepCopy()
	_, err := woc.createWorkflowPod(ctx, tmpl.Name, []apiv1.Container{*mainCtr}, tmpl, &createWorkflowPodOpts{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ancestor of volume mount")
}

// TestCreateWorkflowPod_LegacyAllowsArtifactPathAncestorOfMount guards that the
// rejection is init-less-only: legacy delivers via a bind mount and never
// deletes, so the same config must still build a pod.
func TestCreateWorkflowPod_LegacyAllowsArtifactPathAncestorOfMount(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(artifactPathAncestorOfMountWf)
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	// InitlessPod not set → legacy.

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	tmpl := &woc.execWf.Spec.Templates[0]
	mainCtr := tmpl.Container.DeepCopy()
	pod, err := woc.createWorkflowPod(ctx, tmpl.Name, []apiv1.Container{*mainCtr}, tmpl, &createWorkflowPodOpts{})
	require.NoError(t, err)
	require.NotNil(t, pod)
}

// TestAddArtifactPluginsInitless_SupervisorWiring drives the function
// directly to assert the supervisor receives socket-volume mounts copied
// from each plugin sidecar AND the input-artifacts mount, and that the
// ARGO_ARTIFACT_PLUGIN_NAMES + ARGO_INPUT_ARTIFACT_PLUGIN_NAMES env vars
// are set with the expected union and input-subset values.
func TestAddArtifactPluginsInitless_SupervisorWiring(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	controller.Config.InitlessPod = &config.InitlessPodConfig{Enabled: true}
	controller.Config.ArtifactDrivers = []config.ArtifactDriver{
		{Name: "in-driver", Image: "alpine"},
		{Name: "out-driver", Image: "alpine"},
	}
	controller.Config.Images = map[string]config.Image{
		"alpine": {Entrypoint: []string{"/plugin-server"}},
	}

	tmpl := &wfv1.Template{
		Name: "t",
		Container: &apiv1.Container{
			Image:   "hello-world",
			Command: []string{"echo", "hi"},
		},
		Inputs: wfv1.Inputs{
			Artifacts: []wfv1.Artifact{
				{Name: "in1", Path: "/tmp/in1", ArtifactLocation: wfv1.ArtifactLocation{
					Plugin: &wfv1.PluginArtifact{Name: "in-driver"},
				}},
			},
		},
		Outputs: wfv1.Outputs{
			Artifacts: []wfv1.Artifact{
				{Name: "out1", Path: "/tmp/out1", ArtifactLocation: wfv1.ArtifactLocation{
					Plugin: &wfv1.PluginArtifact{Name: "out-driver"},
				}},
			},
		},
	}
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "wf", UID: "u"},
		Spec:       wfv1.WorkflowSpec{Entrypoint: "t", Templates: []wfv1.Template{*tmpl}},
	}
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Seed a supervisor container so the wiring loop has a target.
	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns"},
		Spec: apiv1.PodSpec{
			Containers: []apiv1.Container{
				{Name: common.SupervisorContainerName, Image: "argoproj/argoexec:test"},
				{Name: common.MainContainerName, Image: "hello-world"},
			},
		},
	}

	require.NoError(t, woc.addArtifactPluginsInitless(ctx, pod, tmpl))

	// Two plugin sidecars appended (one in, one out — no dedup possible).
	var inSidecar, outSidecar, supervisor *apiv1.Container
	for i, c := range pod.Spec.Containers {
		switch c.Name {
		case common.ArtifactPluginSidecarPrefix + "in-driver":
			inSidecar = &pod.Spec.Containers[i]
		case common.ArtifactPluginSidecarPrefix + "out-driver":
			outSidecar = &pod.Spec.Containers[i]
		case common.SupervisorContainerName:
			supervisor = &pod.Spec.Containers[i]
		}
	}
	require.NotNil(t, inSidecar, "input plugin sidecar must be appended")
	require.NotNil(t, outSidecar, "output plugin sidecar must be appended")
	require.NotNil(t, supervisor)

	// Each plugin sidecar must carry the input-artifacts mount (otherwise
	// supervisor's Load call writes to a path the plugin can't access).
	for _, sidecar := range []*apiv1.Container{inSidecar, outSidecar} {
		hasInputMount := false
		for _, vm := range sidecar.VolumeMounts {
			if vm.Name == "input-artifacts" && vm.MountPath == common.ExecutorArtifactBaseDir {
				hasInputMount = true
				break
			}
		}
		assert.True(t, hasInputMount, "plugin sidecar %q must mount the input-artifacts volume in init-less mode", sidecar.Name)
	}

	// Supervisor must carry both env vars: ARGO_ARTIFACT_PLUGIN_NAMES (union
	// of all plugins, used at Save) and ARGO_INPUT_ARTIFACT_PLUGIN_NAMES
	// (input subset, used at Load).
	envByName := map[string]string{}
	for _, e := range supervisor.Env {
		envByName[e.Name] = e.Value
	}
	require.Contains(t, envByName, common.EnvVarArtifactPluginNames)
	require.Contains(t, envByName, common.EnvVarInputArtifactPluginNames)
	assert.Contains(t, envByName[common.EnvVarArtifactPluginNames], "in-driver")
	assert.Contains(t, envByName[common.EnvVarArtifactPluginNames], "out-driver")
	assert.Contains(t, envByName[common.EnvVarInputArtifactPluginNames], "in-driver")
	assert.NotContains(t, envByName[common.EnvVarInputArtifactPluginNames], "out-driver",
		"output-only plugins must not appear in the input plugin list")
}

// TestAddArtifactPluginsInitless_NoSupervisor exercises the early-return
// branch for a template that has plugin artifacts but no supervisor to drive
// them. The function must NOT append orphan plugin sidecars (nothing would
// invoke their Load/Save), so the pod must be left untouched.
func TestAddArtifactPluginsInitless_NoSupervisor(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	controller.Config.InitlessPod = &config.InitlessPodConfig{Enabled: true}
	controller.Config.ArtifactDrivers = []config.ArtifactDriver{
		{Name: "in-driver", Image: "alpine"},
	}
	controller.Config.Images = map[string]config.Image{
		"alpine": {Entrypoint: []string{"/plugin-server"}},
	}

	// Template references a plugin input artifact, so buildPluginSidecars would
	// produce a sidecar — but the pod has no supervisor (as for a data template).
	tmpl := &wfv1.Template{
		Name:      "t",
		Container: &apiv1.Container{Image: "hello-world"},
		Inputs: wfv1.Inputs{
			Artifacts: []wfv1.Artifact{
				{Name: "in1", Path: "/tmp/in1", ArtifactLocation: wfv1.ArtifactLocation{
					Plugin: &wfv1.PluginArtifact{Name: "in-driver"},
				}},
			},
		},
	}
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "wf", UID: "u"},
		Spec:       wfv1.WorkflowSpec{Entrypoint: "t", Templates: []wfv1.Template{*tmpl}},
	}
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns"},
		Spec: apiv1.PodSpec{
			Containers: []apiv1.Container{
				{Name: common.MainContainerName, Image: "hello-world"},
			},
		},
	}

	require.NoError(t, woc.addArtifactPluginsInitless(ctx, pod, tmpl))
	// No orphan sidecars added (no supervisor to drive them) and no panic.
	assert.Len(t, pod.Spec.Containers, 1)
	for _, c := range pod.Spec.Containers {
		assert.NotContains(t, c.Name, common.ArtifactPluginSidecarPrefix,
			"plugin sidecar must not be appended when there is no supervisor")
	}
}

// TestAddArtifactPluginsInitless_NoDuplicateSupervisorMounts runs both
// addInputArtifactsVolumes (which gives the supervisor the shared
// input-artifacts + mirrored user-volume mounts) and addArtifactPluginsInitless
// (which wires plugin sidecars and copies socket mounts onto the supervisor)
// and asserts the supervisor ends with no duplicate mountPaths. Kubernetes
// rejects duplicate mountPaths at admission, so a regression here would make
// every init-less pod with input artifacts + a plugin sidecar fail to schedule.
func TestAddArtifactPluginsInitless_NoDuplicateSupervisorMounts(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	controller.Config.InitlessPod = &config.InitlessPodConfig{Enabled: true}
	controller.Config.ArtifactDrivers = []config.ArtifactDriver{
		{Name: "in-driver", Image: "alpine"},
	}
	controller.Config.Images = map[string]config.Image{
		"alpine": {Entrypoint: []string{"/plugin-server"}},
	}

	tmpl := &wfv1.Template{
		Name: "t",
		Container: &apiv1.Container{
			Image:   "hello-world",
			Command: []string{"echo", "hi"},
		},
		Inputs: wfv1.Inputs{
			Artifacts: []wfv1.Artifact{
				{Name: "in1", Path: "/tmp/in1", ArtifactLocation: wfv1.ArtifactLocation{
					Plugin: &wfv1.PluginArtifact{Name: "in-driver"},
				}},
			},
		},
	}
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "wf", UID: "u"},
		Spec:       wfv1.WorkflowSpec{Entrypoint: "t", Templates: []wfv1.Template{*tmpl}},
	}
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns"},
		Spec: apiv1.PodSpec{
			Containers: []apiv1.Container{
				{Name: common.SupervisorContainerName, Image: "argoproj/argoexec:test"},
				{Name: common.MainContainerName, Image: "hello-world"},
			},
		},
	}

	require.NoError(t, woc.addInputArtifactsVolumes(ctx, pod, tmpl))
	require.NoError(t, woc.addArtifactPluginsInitless(ctx, pod, tmpl))

	var supervisor *apiv1.Container
	for i, c := range pod.Spec.Containers {
		if c.Name == common.SupervisorContainerName {
			supervisor = &pod.Spec.Containers[i]
			break
		}
	}
	require.NotNil(t, supervisor)

	seen := map[string]int{}
	for _, vm := range supervisor.VolumeMounts {
		seen[vm.MountPath]++
	}
	for path, count := range seen {
		assert.Equalf(t, 1, count, "supervisor must not have duplicate mountPath %q (count=%d)", path, count)
	}
}

// TestInitlessNoDuplicateMounts_UserVolumeAndInputArtifact reproduces the
// specific condition that produced duplicate supervisor mountPaths: a template
// with BOTH an input artifact and a user volumeMount. addInputArtifactsVolumes
// mirrors the user volume onto the supervisor at /mainctrfs/<path>, and
// addOutputArtifactsVolumes later mirrors main's same user volume onto every
// argo sidecar at the same /mainctrfs/<path>. Without de-duplication the
// supervisor ends up with two mounts at that path and Kubernetes rejects the
// pod at admission.
func TestInitlessNoDuplicateMounts_UserVolumeAndInputArtifact(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	controller.Config.InitlessPod = &config.InitlessPodConfig{Enabled: true}

	userMount := apiv1.VolumeMount{Name: "workdir", MountPath: "/work"}
	tmpl := &wfv1.Template{
		Name: "t",
		Container: &apiv1.Container{
			Image:        "hello-world",
			Command:      []string{"echo", "hi"},
			VolumeMounts: []apiv1.VolumeMount{userMount},
		},
		Inputs: wfv1.Inputs{
			Artifacts: []wfv1.Artifact{
				{Name: "in1", Path: "/tmp/in1"},
			},
		},
	}
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "wf", UID: "u"},
		Spec:       wfv1.WorkflowSpec{Entrypoint: "t", Templates: []wfv1.Template{*tmpl}},
	}
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns"},
		Spec: apiv1.PodSpec{
			Containers: []apiv1.Container{
				{Name: common.SupervisorContainerName, Image: "argoproj/argoexec:test"},
				// main already carries the user volume mount (as addVolumeReferences
				// would have set up) so addOutputArtifactsVolumes mirrors it.
				{Name: common.MainContainerName, Image: "hello-world", VolumeMounts: []apiv1.VolumeMount{userMount}},
			},
		},
	}

	require.NoError(t, woc.addInputArtifactsVolumes(ctx, pod, tmpl))
	addOutputArtifactsVolumes(pod, tmpl)

	var supervisor *apiv1.Container
	for i, c := range pod.Spec.Containers {
		if c.Name == common.SupervisorContainerName {
			supervisor = &pod.Spec.Containers[i]
			break
		}
	}
	require.NotNil(t, supervisor)

	byPath := map[string]int{}
	workdirMounts := 0
	for _, vm := range supervisor.VolumeMounts {
		byPath[vm.MountPath]++
		if vm.Name == userMount.Name {
			workdirMounts++
		}
	}
	assert.Equal(t, 1, workdirMounts, "supervisor must have exactly one mount for the user volume, not a duplicate")
	for path, count := range byPath {
		assert.Equalf(t, 1, count, "supervisor must not have duplicate mountPath %q (count=%d)", path, count)
	}
}

// TestIsInitlessPodEnabled_DefaultFalse guards that the legacy path remains the default.
func TestIsInitlessPodEnabled_DefaultFalse(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	assert.False(t, controller.isInitlessPodEnabled(), "must default to off")

	controller.Config.InitlessPod = &config.InitlessPodConfig{Enabled: false}
	assert.False(t, controller.isInitlessPodEnabled())

	controller.Config.InitlessPod = &config.InitlessPodConfig{Enabled: true}
	assert.True(t, controller.isInitlessPodEnabled())
}
