package controller

import (
	"context"
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
	"github.com/argoproj/argo-workflows/v4/workflow/controller/entrypoint"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

// These tests exercise podBuilder.build() directly — build is PURE. build
// performs no external writes and reads no ambient process state: the wall
// clock and the
// EnvVarPodStatusCaptureFinalizer env are captured into the podBuilderInputs
// snapshot at newPodBuilder time (in.now / in.podStatusCaptureFinalizer), so a
// given snapshot fully determines the output. A podBuilder is constructed from a
// literal podBuilderInputs snapshot plus a hand-written stubPodBuilderDeps that
// implements podBuilderDeps WITHOUT any fake clientset, informer, or live
// cluster. lookupImage returns a canned entrypoint so build never touches
// Docker Hub (the live-lookup failures that plague the createWorkflowPod-based
// tests cannot happen here). build is called and we assert on the resulting
// pod spec for each layout.

const testStubExecutorImage = "argoproj/argoexec:build-unit-test"

// stubPodBuilderDeps is a tiny hand-written implementation of podBuilderDeps.
// It does the minimum work each method must do for build to produce a
// representative pod, with NO fake.NewSimpleClientset and NO informer/cluster.
// Where a dep mutates the pod (init/supervisor/exec containers, scheduling,
// metadata, DNS), it reproduces just enough of the real behaviour to let the
// assertions below distinguish the two layouts.
type stubPodBuilderDeps struct {
	executorImage     string
	archiveLogs       bool
	initContainerName string // name of the init container legacyLayout asks for

	// gotTimeoutNow records the now value build threads into checkTemplateTimeouts,
	// so a test can assert build uses the captured snapshot time rather than the
	// live wall-clock.
	gotTimeoutNow time.Time
}

func newStubDeps() *stubPodBuilderDeps {
	return &stubPodBuilderDeps{
		executorImage:     testStubExecutorImage,
		initContainerName: common.InitContainerName,
	}
}

// lookupImage returns a fixed entrypoint so build never reaches the network.
func (s *stubPodBuilderDeps) lookupImage(_ context.Context, _ string, _ entrypoint.Options) (*entrypoint.Image, error) {
	return &entrypoint.Image{Entrypoint: []string{"/stub-entrypoint"}, Cmd: []string{"stub-cmd"}}, nil
}

// newExecContainer mirrors the real helper's essentials: an argoexec-image
// container carrying the executor image (so build's "mount /tmp on the
// executor container" branch fires).
func (s *stubPodBuilderDeps) newExecContainer(name string, _ *wfv1.Template) *apiv1.Container {
	return &apiv1.Container{Name: name, Image: s.executorImage}
}

func (s *stubPodBuilderDeps) getExecutorLogOpts(_ context.Context) []string {
	return []string{"--loglevel", "info"}
}

func (s *stubPodBuilderDeps) addMetadata(_ *apiv1.Pod, _ *wfv1.Template) {}
func (s *stubPodBuilderDeps) addDNSConfig(_ *apiv1.Pod)                  {}
func (s *stubPodBuilderDeps) addSchedulingConstraints(_ context.Context, _ *apiv1.Pod, _ *wfv1.WorkflowSpec, _ *wfv1.Template, _ *wfv1.Template) {
}

func (s *stubPodBuilderDeps) IsArchiveLogs(_ *wfv1.Template) bool                { return s.archiveLogs }
func (s *stubPodBuilderDeps) IsArchiveSystemContainerLogs(_ *wfv1.Template) bool { return false }
func (s *stubPodBuilderDeps) executorServiceAccountName(_ *wfv1.Template) string { return "" }
func (s *stubPodBuilderDeps) createArtifactVolumeMounts(_ context.Context, _ *wfv1.Template) []apiv1.Volume {
	return nil
}
func (s *stubPodBuilderDeps) addInputArtifactsVolumes(_ context.Context, _ *apiv1.Pod, _ *wfv1.Template) error {
	return nil
}

// newInitContainers reproduces the legacy layout's single standard init
// container (argoexec init). This is what build copies the var-argo mount onto.
func (s *stubPodBuilderDeps) newInitContainers(_ context.Context, _ *wfv1.Template) ([]apiv1.Container, error) {
	return []apiv1.Container{{Name: s.initContainerName, Image: s.executorImage, Command: []string{"argoexec", "init"}}}, nil
}

// newSupervisorContainer reproduces the init-less supervisor: an argoexec-image
// container running `argoexec supervisor` and carrying ARGO_INITLESS_POD.
func (s *stubPodBuilderDeps) newSupervisorContainer(_ context.Context, _ *wfv1.Template) *apiv1.Container {
	return &apiv1.Container{
		Name:    common.SupervisorContainerName,
		Image:   s.executorImage,
		Command: []string{"argoexec", "supervisor"},
		Env:     []apiv1.EnvVar{{Name: common.EnvVarInitlessPod, Value: "true"}},
	}
}

// buildArgoBinVolume reproduces the argoexec-bin image volume.
func (s *stubPodBuilderDeps) buildArgoBinVolume() apiv1.Volume {
	return apiv1.Volume{
		Name: argoBinVolumeName,
		VolumeSource: apiv1.VolumeSource{
			Image: &apiv1.ImageVolumeSource{Reference: s.executorImage, PullPolicy: apiv1.PullIfNotPresent},
		},
	}
}

func (s *stubPodBuilderDeps) addArtifactPluginsLegacy(_ context.Context, _ *apiv1.Pod, _ *wfv1.Template, _ *config.Config) error {
	return nil
}
func (s *stubPodBuilderDeps) addArtifactPluginsInitless(_ context.Context, _ *apiv1.Pod, _ *wfv1.Template) error {
	return nil
}

func (s *stubPodBuilderDeps) getNodeByName(_ string) (*wfv1.NodeStatus, error) { return nil, nil }
func (s *stubPodBuilderDeps) findRetryNode(_ string) *wfv1.NodeStatus          { return nil }
func (s *stubPodBuilderDeps) retryStrategyForTemplate(_ *wfv1.Template) *wfv1.RetryStrategy {
	return nil
}
func (s *stubPodBuilderDeps) retryNodeTemplate(_ context.Context, _ *wfv1.NodeStatus, fallback *wfv1.Template) (*wfv1.Template, error) {
	return fallback, nil
}
func (s *stubPodBuilderDeps) applyRetryOnDifferentHost(_ string, _ wfv1.RetryStrategy, _ *apiv1.Pod) {
}
func (s *stubPodBuilderDeps) checkTemplateTimeouts(_ *wfv1.Template, _ *wfv1.NodeStatus, now time.Time) (deadline, pendingDeadline *time.Time, err error) {
	s.gotTimeoutNow = now
	return nil, nil, nil
}
func (s *stubPodBuilderDeps) getServiceAccountTokenName(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (s *stubPodBuilderDeps) getPodGCDelay(_ context.Context, _ *wfv1.PodGC) time.Duration {
	return 0
}
func (s *stubPodBuilderDeps) persistentVolumeClaims() []apiv1.Volume { return nil }

// TestBuild_ThreadsSnapshotNowIntoTimeout asserts build() drives the template
// timeout determination off the captured snapshot time (pb.in.now) rather than
// the live wall-clock — a live time.Now() read inside checkTemplateTimeouts
// would leak a clock into the otherwise-pure build via the deps boundary and
// break determinism for a given snapshot.
func TestBuild_ThreadsSnapshotNowIntoTimeout(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tmpl := simpleContainerTemplate()
	main := tmpl.Container.DeepCopy()

	deps := newStubDeps()
	pb := newTestPodBuilder(tmpl, []apiv1.Container{*main}, false, deps)
	// Pin the snapshot clock to a fixed instant well away from real time.Now().
	snapshotNow := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	pb.in.now = snapshotNow

	_, err := pb.build(ctx)
	require.NoError(t, err)
	assert.Equal(t, snapshotNow, deps.gotTimeoutNow,
		"build must pass the captured snapshot time into checkTemplateTimeouts, not the live wall-clock")
}

// newTestPodBuilder constructs a podBuilder entirely from literals — no woc, no
// controller, no clientset. initless selects the layout, exactly as
// newPodBuilder does in production.
func newTestPodBuilder(tmpl *wfv1.Template, mainCtrs []apiv1.Container, initless bool, deps podBuilderDeps) *podBuilder {
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf", Namespace: "ns", UID: "wf-uid"}}
	return &podBuilder{
		in: podBuilderInputs{
			nodeName:           "node",
			nodeID:             "node-id",
			namespace:          "ns",
			wfName:             "wf",
			ownerRef:           *metav1.NewControllerRef(wf, wfv1.SchemeGroupVersion.WithKind("Workflow")),
			mainCtrs:           mainCtrs,
			tmpl:               tmpl,
			opts:               &createWorkflowPodOpts{},
			execWfSpec:         &wfv1.WorkflowSpec{Entrypoint: tmpl.Name, Templates: []wfv1.Template{*tmpl}},
			globalParams:       common.Parameters{},
			artifactRepository: &wfv1.ArtifactRepository{},
			config:             &config.Config{},
			executorImage:      testStubExecutorImage,
			now:                time.Now().UTC(),
			podNameVersion:     util.PodNameV2,
			log:                logging.RequireLoggerFromContext(logging.TestContext(context.Background())),
		},
		deps:   deps,
		layout: newPodLayout(initless),
	}
}

// containerByName finds a container in the list by name.
func containerByName(ctrs []apiv1.Container, name string) *apiv1.Container {
	for i := range ctrs {
		if ctrs[i].Name == name {
			return &ctrs[i]
		}
	}
	return nil
}

func hasVolume(vols []apiv1.Volume, name string) bool {
	for _, v := range vols {
		if v.Name == name {
			return true
		}
	}
	return false
}

func envValue(c *apiv1.Container, name string) (string, bool) {
	for _, e := range c.Env {
		if e.Name == name {
			return e.Value, true
		}
	}
	return "", false
}

func hasMount(c *apiv1.Container, name string) bool {
	for _, vm := range c.VolumeMounts {
		if vm.Name == name {
			return true
		}
	}
	return false
}

func simpleContainerTemplate() *wfv1.Template {
	return &wfv1.Template{
		Name: "t",
		Container: &apiv1.Container{
			Image:   "my-image",
			Command: []string{"echo", "hi"},
		},
	}
}

// assertLegacyPodShape asserts the layout divergences that define the legacy
// (init + wait) pod shape. It is the single source of truth for these
// assertions, called both by the pure build() tests in this file and by the
// integration-style createWorkflowPod tests in workflowpod_initless_test.go, so
// the two test groups can never drift apart on what "legacy shape" means.
//
// Divergences asserted: exactly one standard init container, a wait (not
// supervisor) aux container, no argoexec-bin image volume, and main without the
// /argo-bin mount or ARGO_WAIT_FOR_READY env.
func assertLegacyPodShape(t *testing.T, pod *apiv1.Pod) {
	t.Helper()

	// Exactly one init container, the standard `init`.
	require.Len(t, pod.Spec.InitContainers, 1, "legacy layout schedules one init container")
	assert.Equal(t, common.InitContainerName, pod.Spec.InitContainers[0].Name)

	// Aux container is `wait`, NOT `supervisor`.
	require.NotNil(t, containerByName(pod.Spec.Containers, common.WaitContainerName), "legacy aux container must be wait")
	assert.Nil(t, containerByName(pod.Spec.Containers, common.SupervisorContainerName), "legacy layout has no supervisor")

	// No argoexec-bin image volume in the legacy layout.
	assert.False(t, hasVolume(pod.Spec.Volumes, argoBinVolumeName), "legacy layout must not add the argoexec-bin image volume")

	// Main does NOT get the /argo-bin mount nor ARGO_WAIT_FOR_READY.
	mainCtr := containerByName(pod.Spec.Containers, common.MainContainerName)
	require.NotNil(t, mainCtr)
	assert.False(t, hasMount(mainCtr, argoBinVolumeName), "legacy main must not mount argoexec-bin")
	_, hasWaitForReady := envValue(mainCtr, common.EnvVarWaitForReady)
	assert.False(t, hasWaitForReady, "legacy main must not carry ARGO_WAIT_FOR_READY")
}

// assertInitlessPodShape asserts the layout divergences that define the
// init-less (supervisor) pod shape. Like assertLegacyPodShape it is the single
// source of truth for these assertions across both test groups.
//
// Divergences asserted: zero init containers, a supervisor (not wait) aux
// container running `argoexec supervisor`, the argoexec-bin image volume on the
// pod, the /argo-bin mount on main, ARGO_WAIT_FOR_READY=true on main, and
// ARGO_TEMPLATE on the supervisor (which writes /var/run/argo/template).
func assertInitlessPodShape(t *testing.T, pod *apiv1.Pod) {
	t.Helper()

	// Zero init containers.
	assert.Empty(t, pod.Spec.InitContainers, "init-less layout schedules zero init containers")

	// Aux container is `supervisor`, NOT `wait`.
	supervisor := containerByName(pod.Spec.Containers, common.SupervisorContainerName)
	require.NotNil(t, supervisor, "init-less aux container must be supervisor")
	assert.Nil(t, containerByName(pod.Spec.Containers, common.WaitContainerName), "init-less layout has no wait container")
	require.GreaterOrEqual(t, len(supervisor.Command), 2)
	assert.Equal(t, "argoexec", supervisor.Command[0])
	assert.Equal(t, "supervisor", supervisor.Command[1])

	// argoexec-bin image volume present on the pod.
	assert.True(t, hasVolume(pod.Spec.Volumes, argoBinVolumeName), "init-less layout adds the argoexec-bin image volume")

	// Main mounts /argo-bin and carries ARGO_WAIT_FOR_READY=true.
	mainCtr := containerByName(pod.Spec.Containers, common.MainContainerName)
	require.NotNil(t, mainCtr)
	assert.True(t, hasMount(mainCtr, argoBinVolumeName), "init-less main must mount argoexec-bin")
	v, ok := envValue(mainCtr, common.EnvVarWaitForReady)
	require.True(t, ok, "init-less main must carry ARGO_WAIT_FOR_READY")
	assert.Equal(t, "true", v)

	// Supervisor carries ARGO_TEMPLATE (it writes /var/run/argo/template).
	_, ok = envValue(supervisor, common.EnvVarTemplate)
	assert.True(t, ok, "supervisor must carry ARGO_TEMPLATE")
}

// assertInitlessNoAuxContainerShape asserts the supervisor-less init-less case
// (e.g. a resource template without log archiving): no supervisor is scheduled
// and main carries ARGO_TEMPLATE inline (there is no supervisor to write the
// template file, so emissary readTemplate falls back to the env var). Single
// source of truth shared by both test groups.
func assertInitlessNoAuxContainerShape(t *testing.T, pod *apiv1.Pod) {
	t.Helper()

	// No supervisor (no pre/post-main work to do).
	assert.Nil(t, containerByName(pod.Spec.Containers, common.SupervisorContainerName),
		"supervisor-less template must not schedule a supervisor")

	// main carries ARGO_TEMPLATE inline so emissary readTemplate falls back to env.
	mainCtr := containerByName(pod.Spec.Containers, common.MainContainerName)
	require.NotNil(t, mainCtr)
	tv, ok := envValue(mainCtr, common.EnvVarTemplate)
	require.True(t, ok, "supervisor-less main must carry ARGO_TEMPLATE inline")
	assert.NotEmpty(t, tv)
}

// TestBuild_LegacyLayout_PureSpec drives build() for the legacy (init+wait)
// layout from a literal podBuilder and asserts the resulting pod spec.
func TestBuild_LegacyLayout_PureSpec(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tmpl := simpleContainerTemplate()
	main := tmpl.Container.DeepCopy()

	pb := newTestPodBuilder(tmpl, []apiv1.Container{*main}, false, newStubDeps())
	result, err := pb.build(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)
	pod := result.Pod
	require.NotNil(t, pod)

	// Shared layout-shape divergences (single source of truth, also asserted by
	// TestCreateWorkflowPod_LegacyShape against the production deps adapter).
	assertLegacyPodShape(t, pod)

	// Below: assertions UNIQUE to the pure build() path.

	// Main container emissary-wrapped via the LEGACY argoexec path.
	mainCtr := containerByName(pod.Spec.Containers, common.MainContainerName)
	require.NotNil(t, mainCtr)
	require.NotEmpty(t, mainCtr.Command)
	assert.Equal(t, legacyArgoexecBinaryPath, mainCtr.Command[0], "legacy main execs argoexec from /var/run/argo")

	// Standard pod volumes present.
	assert.True(t, hasVolume(pod.Spec.Volumes, volumeVarArgo.Name))
	assert.True(t, hasVolume(pod.Spec.Volumes, volumeTmpDir.Name))

	// build is pure: no ExtraObjects for a small template, restart policy Never.
	assert.Empty(t, result.ExtraObjects)
	assert.Equal(t, apiv1.RestartPolicyNever, pod.Spec.RestartPolicy)
}

// TestBuild_InitlessLayout_PureSpec drives build() for the init-less
// (supervisor) layout from a literal podBuilder and asserts the resulting pod
// spec carries every init-less divergence: zero init containers, supervisor
// aux container, argoexec-bin image volume + /argo-bin mount on main, and
// ARGO_WAIT_FOR_READY on main.
func TestBuild_InitlessLayout_PureSpec(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tmpl := simpleContainerTemplate()
	main := tmpl.Container.DeepCopy()

	pb := newTestPodBuilder(tmpl, []apiv1.Container{*main}, true, newStubDeps())
	result, err := pb.build(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)
	pod := result.Pod
	require.NotNil(t, pod)

	// Shared layout-shape divergences (single source of truth, also asserted by
	// TestCreateWorkflowPod_InitlessShape against the production deps adapter).
	assertInitlessPodShape(t, pod)

	// Below: assertions UNIQUE to the pure build() path.

	// Main execs argoexec from the IMAGE-VOLUME path, not /var/run/argo.
	mainCtr := containerByName(pod.Spec.Containers, common.MainContainerName)
	require.NotNil(t, mainCtr)
	require.NotEmpty(t, mainCtr.Command)
	assert.Equal(t, argoBinExecutorPath, mainCtr.Command[0], "init-less main execs argoexec from the image volume")

	assert.Empty(t, result.ExtraObjects)
}

// TestBuild_InitlessLayout_NoAuxContainer asserts the supervisor-less init-less
// case (resource template without log archiving / data templates): no aux
// container is scheduled, and main carries ARGO_TEMPLATE inline because there
// is no supervisor to write the template file.
func TestBuild_InitlessLayout_NoAuxContainer(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tmpl := &wfv1.Template{
		Name: "t",
		Resource: &wfv1.ResourceTemplate{
			Action:   "get",
			Manifest: "apiVersion: v1\nkind: Pod\nmetadata:\n  name: example\n",
		},
	}
	main := apiv1.Container{
		Name:    common.MainContainerName,
		Image:   testStubExecutorImage,
		Command: []string{"argoexec", "resource", "get"},
	}

	deps := newStubDeps()
	deps.archiveLogs = false // resource-without-logs: no aux container
	pb := newTestPodBuilder(tmpl, []apiv1.Container{main}, true, deps)
	result, err := pb.build(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)
	pod := result.Pod
	require.NotNil(t, pod)

	// Shared supervisor-less shape (single source of truth, also asserted by
	// TestCreateWorkflowPod_InitlessNoAuxContainer against the production deps
	// adapter).
	assertInitlessNoAuxContainerShape(t, pod)
}

// manifestFromResourceTemplate is a resource template that sources its manifest
// from an input artifact instead of an inline manifest, without log archiving.
func manifestFromResourceTemplate() *wfv1.Template {
	return &wfv1.Template{
		Name: "t",
		Inputs: wfv1.Inputs{Artifacts: []wfv1.Artifact{{
			Name: "manifest",
			Path: "/tmp/manifest.yaml",
			ArtifactLocation: wfv1.ArtifactLocation{
				Raw: &wfv1.RawArtifact{Data: "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: example\n"},
			},
		}}},
		Resource: &wfv1.ResourceTemplate{
			Action:       "create",
			ManifestFrom: &wfv1.ManifestFrom{Artifact: &wfv1.Artifact{Name: "manifest"}},
		},
	}
}

// TestBuild_InitlessResourceManifestFrom_SchedulesSupervisor pins the
// manifestFrom aux-container rule in the pure build() path: a resource template
// without log archiving normally runs supervisor-less, but when its manifest
// comes from an input artifact there is no init container to stage it, so
// build() must schedule a supervisor to download the artifact pre-main.
func TestBuild_InitlessResourceManifestFrom_SchedulesSupervisor(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tmpl := manifestFromResourceTemplate()
	main := apiv1.Container{
		Name:    common.MainContainerName,
		Image:   testStubExecutorImage,
		Command: []string{"argoexec", "resource", "create"},
	}

	deps := newStubDeps()
	deps.archiveLogs = false // would be supervisor-less were it not for manifestFrom
	pb := newTestPodBuilder(tmpl, []apiv1.Container{main}, true, deps)
	result, err := pb.build(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)
	pod := result.Pod
	require.NotNil(t, pod)

	supervisor := containerByName(pod.Spec.Containers, common.SupervisorContainerName)
	require.NotNil(t, supervisor, "manifestFrom resource template must schedule a supervisor in init-less mode")
	assert.Nil(t, containerByName(pod.Spec.Containers, common.WaitContainerName))

	// With a supervisor present, main must block on its readiness so the
	// manifest artifact is staged before `argoexec resource` reads it.
	mainCtr := containerByName(pod.Spec.Containers, common.MainContainerName)
	require.NotNil(t, mainCtr)
	v, ok := envValue(mainCtr, common.EnvVarWaitForReady)
	require.True(t, ok, "main must wait for supervisor readiness so the manifest artifact is staged first")
	assert.Equal(t, "true", v)
}

// TestBuild_LegacyResourceManifestFrom_NoAuxContainer guards that the
// manifestFrom trigger is init-less-only in the pure build() path: legacy mode
// stages the manifest via the init container, so the same template must not
// gain a wait (or supervisor) container.
func TestBuild_LegacyResourceManifestFrom_NoAuxContainer(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tmpl := manifestFromResourceTemplate()
	main := apiv1.Container{
		Name:    common.MainContainerName,
		Image:   testStubExecutorImage,
		Command: []string{"argoexec", "resource", "create"},
	}

	deps := newStubDeps()
	deps.archiveLogs = false
	pb := newTestPodBuilder(tmpl, []apiv1.Container{main}, false, deps)
	result, err := pb.build(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)
	pod := result.Pod
	require.NotNil(t, pod)

	assert.Nil(t, containerByName(pod.Spec.Containers, common.WaitContainerName),
		"legacy manifestFrom resource without logs must not gain a wait container")
	assert.Nil(t, containerByName(pod.Spec.Containers, common.SupervisorContainerName))
}

// containerSetTemplate returns a ContainerSet template with the named members.
// tmpl.GetType() reports TemplateTypeContainerSet, so build() preserves each
// member's name (it does NOT collapse them to the single "main" container).
func containerSetTemplate(names ...string) *wfv1.Template {
	nodes := make([]wfv1.ContainerNode, 0, len(names))
	for _, n := range names {
		nodes = append(nodes, wfv1.ContainerNode{
			Container: apiv1.Container{
				Name:    n,
				Image:   "my-image",
				Command: []string{"echo", n},
			},
		})
	}
	return &wfv1.Template{
		Name:         "t",
		ContainerSet: &wfv1.ContainerSetTemplate{Containers: nodes},
	}
}

// TestBuild_InitlessLayout_ContainerSet_AllMembersDecorated exercises the
// correctness-critical PER-USER-CONTAINER path in build(): for a ContainerSet
// template with N members, the init-less argoexec-bin mount AND
// ARGO_WAIT_FOR_READY env var must land on EVERY member, not just one main
// container. The existing pure-build tests only use single-container templates,
// so this multi-member branch (workflowpod.go ~738 loop +
// initlessLayout.decorateUserContainer) was otherwise untested.
func TestBuild_InitlessLayout_ContainerSet_AllMembersDecorated(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	memberNames := []string{"a", "b", "c"}
	tmpl := containerSetTemplate(memberNames...)

	// mainCtrs mirrors what executeContainerSet passes in: one container per node,
	// each carrying its own name (preserved because tmpl is a ContainerSet).
	mainCtrs := make([]apiv1.Container, 0, len(memberNames))
	for _, n := range memberNames {
		mainCtrs = append(mainCtrs, apiv1.Container{
			Name:    n,
			Image:   "my-image",
			Command: []string{"echo", n},
		})
	}

	pb := newTestPodBuilder(tmpl, mainCtrs, true, newStubDeps())
	result, err := pb.build(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)
	pod := result.Pod
	require.NotNil(t, pod)

	// A ContainerSet is neither Resource nor Data, so the supervisor aux container
	// is scheduled — which is precisely the condition under which
	// decorateUserContainer applies ARGO_WAIT_FOR_READY.
	require.NotNil(t, containerByName(pod.Spec.Containers, common.SupervisorContainerName),
		"init-less ContainerSet must schedule a supervisor aux container")

	// The argoexec-bin image volume is added once at the pod level.
	assert.True(t, hasVolume(pod.Spec.Volumes, argoBinVolumeName),
		"init-less layout adds the argoexec-bin image volume")

	// EVERY ContainerSet member (and ONLY user members) must be decorated.
	for _, name := range memberNames {
		member := containerByName(pod.Spec.Containers, name)
		require.NotNil(t, member, "ContainerSet member %q must be present and keep its name", name)

		assert.True(t, hasMount(member, argoBinVolumeName),
			"member %q must mount the argoexec-bin image volume", name)

		v, ok := envValue(member, common.EnvVarWaitForReady)
		require.True(t, ok, "member %q must carry ARGO_WAIT_FOR_READY", name)
		assert.Equal(t, "true", v, "member %q ARGO_WAIT_FOR_READY must be true", name)

		// Each member execs argoexec from the IMAGE-VOLUME path, not /var/run/argo.
		require.NotEmpty(t, member.Command)
		assert.Equal(t, argoBinExecutorPath, member.Command[0],
			"member %q must exec argoexec from the image volume", name)
	}

	// The supervisor is an argo sidecar and must NOT be decorated as a user
	// container: no ARGO_WAIT_FOR_READY (it is the thing being waited ON).
	supervisor := containerByName(pod.Spec.Containers, common.SupervisorContainerName)
	require.NotNil(t, supervisor)
	_, hasWaitForReady := envValue(supervisor, common.EnvVarWaitForReady)
	assert.False(t, hasWaitForReady, "supervisor must not carry ARGO_WAIT_FOR_READY")

	assert.Empty(t, result.ExtraObjects)
}

// TestBuild_NoFakeClientset documents the contract of these tests: build() ran
// to completion with a stub deps and a literal podBuilder, no clientset, no
// cluster, and produced a pod. (A guard against any future change that
// re-introduces a hidden live dependency into the pure path.)
func TestBuild_NoFakeClientset(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tmpl := simpleContainerTemplate()
	main := tmpl.Container.DeepCopy()
	for _, initless := range []bool{false, true} {
		pb := newTestPodBuilder(tmpl, []apiv1.Container{*main}, initless, newStubDeps())
		result, err := pb.build(ctx)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Pod)
		assert.Equal(t, "ns", result.Pod.Namespace)
	}
}

// envConfigMountName is the volume/mount name build() uses for the offload
// ConfigMap (workflowpod.go: "argo-env-config"). Kept as a literal here so the
// test fails loudly if the production name changes underneath it.
const envConfigMountName = "argo-env-config"

// oversizedTemplate returns a container template whose JSON marshalling exceeds
// common.MaxEnvVarLen, forcing build()'s ARGO_TEMPLATE offload branch. The bulk
// lives in tmpl.Metadata.Annotations, which build()'s simplifiedTmpl step
// preserves (only Inputs.Parameters are stripped), so the marshalled
// ARGO_TEMPLATE value is guaranteed to be oversized. The stub addMetadata is a
// no-op, so this does NOT inflate the pod object itself — only the template env
// value, isolating the offloadEnvVarTemplate path.
func oversizedTemplate() *wfv1.Template {
	tmpl := simpleContainerTemplate()
	tmpl.Metadata = wfv1.Metadata{
		Annotations: map[string]string{
			"big": strings.Repeat("x", common.MaxEnvVarLen+1024),
		},
	}
	return tmpl
}

// TestBuild_OffloadTemplate_InitlessSupervisor covers the oversized ARGO_TEMPLATE
// path for the init-less layout: the supervisor carries ARGO_TEMPLATE, so when
// it exceeds MaxEnvVarLen build() must emit a ConfigMap into result.ExtraObjects
// and point the supervisor's ARGO_TEMPLATE at it (sentinel value + mounted
// volume) instead of an oversized inline value.
func TestBuild_OffloadTemplate_InitlessSupervisor(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tmpl := oversizedTemplate()
	main := tmpl.Container.DeepCopy()

	pb := newTestPodBuilder(tmpl, []apiv1.Container{*main}, true, newStubDeps())
	result, err := pb.build(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)
	pod := result.Pod
	require.NotNil(t, pod)

	// (a) a ConfigMap is emitted into result.ExtraObjects, named after the pod,
	// carrying the full (oversized) template under the ARGO_TEMPLATE key.
	require.Len(t, result.ExtraObjects, 1, "oversized template must emit exactly one offload ConfigMap")
	cm := result.ExtraObjects[0]
	require.NotNil(t, cm)
	assert.Equal(t, pod.Name, cm.Name, "offload ConfigMap is named after the pod")
	assert.Equal(t, pod.Namespace, cm.Namespace)
	tmplData, ok := cm.Data[common.EnvVarTemplate]
	require.True(t, ok, "offload ConfigMap must carry the template under the ARGO_TEMPLATE key")
	assert.Greater(t, len(tmplData), common.MaxEnvVarLen, "offloaded template data must be the oversized value")

	// The pod must mount the ConfigMap so emissary can read the offloaded template.
	require.True(t, hasVolume(pod.Spec.Volumes, envConfigMountName), "pod must add the offload ConfigMap volume")

	// (b) the supervisor references the ConfigMap: ARGO_TEMPLATE is the offloaded
	// sentinel (NOT the oversized inline value), and it mounts the ConfigMap.
	supervisor := containerByName(pod.Spec.Containers, common.SupervisorContainerName)
	require.NotNil(t, supervisor, "init-less layout must schedule a supervisor")
	tv, ok := envValue(supervisor, common.EnvVarTemplate)
	require.True(t, ok, "supervisor must still carry ARGO_TEMPLATE")
	assert.Equal(t, common.EnvVarTemplateOffloaded, tv, "supervisor ARGO_TEMPLATE must be the offloaded sentinel, not inline")
	assert.LessOrEqual(t, len(tv), common.MaxEnvVarLen, "supervisor must not carry the oversized inline template")
	assert.True(t, hasMount(supervisor, envConfigMountName), "supervisor must mount the offload ConfigMap")
}

// TestBuild_OffloadTemplate_LegacyInitContainer covers the oversized
// ARGO_TEMPLATE path for the legacy layout, where ARGO_TEMPLATE rides on the
// init container. build() must offload it to a ConfigMap and rewrite the init
// container's ARGO_TEMPLATE to the sentinel + ConfigMap mount.
func TestBuild_OffloadTemplate_LegacyInitContainer(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tmpl := oversizedTemplate()
	main := tmpl.Container.DeepCopy()

	pb := newTestPodBuilder(tmpl, []apiv1.Container{*main}, false, newStubDeps())
	result, err := pb.build(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)
	pod := result.Pod
	require.NotNil(t, pod)

	// (a) ConfigMap emitted with the oversized template.
	require.Len(t, result.ExtraObjects, 1, "oversized template must emit exactly one offload ConfigMap")
	cm := result.ExtraObjects[0]
	tmplData, ok := cm.Data[common.EnvVarTemplate]
	require.True(t, ok, "offload ConfigMap must carry the template under the ARGO_TEMPLATE key")
	assert.Greater(t, len(tmplData), common.MaxEnvVarLen)
	require.True(t, hasVolume(pod.Spec.Volumes, envConfigMountName), "pod must add the offload ConfigMap volume")

	// (b) the init container references the ConfigMap rather than an inline value.
	require.Len(t, pod.Spec.InitContainers, 1, "legacy layout schedules one init container")
	initCtr := &pod.Spec.InitContainers[0]
	tv, ok := envValue(initCtr, common.EnvVarTemplate)
	require.True(t, ok, "legacy init container must still carry ARGO_TEMPLATE")
	assert.Equal(t, common.EnvVarTemplateOffloaded, tv, "init container ARGO_TEMPLATE must be the offloaded sentinel, not inline")
	assert.True(t, hasMount(initCtr, envConfigMountName), "init container must mount the offload ConfigMap")
}

// TestBuild_OffloadContainerArgs covers the second offload trigger: a main
// container whose Args marshal larger than MaxEnvVarLen. build() must offload
// the args into the ConfigMap (under ARGO_CONTAINER_ARGS_FILE), clear the
// inline Args, and point the container at the args file via env var + mount.
func TestBuild_OffloadContainerArgs(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tmpl := simpleContainerTemplate()
	main := tmpl.Container.DeepCopy()
	main.Name = common.MainContainerName
	// One oversized arg pushes the marshalled args JSON past MaxEnvVarLen.
	main.Args = []string{strings.Repeat("a", common.MaxEnvVarLen+1024)}

	pb := newTestPodBuilder(tmpl, []apiv1.Container{*main}, true, newStubDeps())
	result, err := pb.build(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)
	pod := result.Pod
	require.NotNil(t, pod)

	// (a) ConfigMap emitted carrying the oversized args under the args-file key.
	require.Len(t, result.ExtraObjects, 1, "oversized args must emit exactly one offload ConfigMap")
	cm := result.ExtraObjects[0]
	argsData, ok := cm.Data[common.EnvVarContainerArgsFile]
	require.True(t, ok, "offload ConfigMap must carry the args under the ARGO_CONTAINER_ARGS_FILE key")
	assert.Greater(t, len(argsData), common.MaxEnvVarLen)
	require.True(t, hasVolume(pod.Spec.Volumes, envConfigMountName), "pod must add the offload ConfigMap volume")

	// (b) the main container references the args file: inline Args cleared, env
	// var points at the mounted file, and the ConfigMap is mounted.
	mainCtr := containerByName(pod.Spec.Containers, common.MainContainerName)
	require.NotNil(t, mainCtr)
	assert.Nil(t, mainCtr.Args, "offloaded main container must have its inline Args cleared")
	argsFile, ok := envValue(mainCtr, common.EnvVarContainerArgsFile)
	require.True(t, ok, "main container must carry ARGO_CONTAINER_ARGS_FILE pointing at the offloaded file")
	assert.Equal(t, common.EnvConfigMountPath+"/"+common.EnvVarContainerArgsFile, argsFile)
	assert.True(t, hasMount(mainCtr, envConfigMountName), "main container must mount the offload ConfigMap")
}
