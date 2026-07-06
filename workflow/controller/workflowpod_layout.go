package controller

import (
	"context"

	apiv1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// podLayout concentrates every init-less-pod (beta) divergence in pod
// construction. The concrete layout is chosen ONCE, in newPodBuilder, and
// stored on podBuilder; build() reaches every layout-specific decision through
// this interface, so the two pod shapes (legacy init+wait, init-less
// supervisor) live in one place.
type podLayout interface {
	// auxContainer builds the auxiliary executor container that runs alongside
	// the user containers: the `wait` container in the legacy layout, or the
	// `supervisor` container in the init-less layout.
	auxContainer(ctx context.Context, pb *podBuilder, tmpl *wfv1.Template) *apiv1.Container

	// initContainers returns the pod's init containers: the standard (and
	// plugin) init containers in the legacy layout, or none in the init-less
	// layout (the supervisor runs as a regular container and takes on the
	// pre-main responsibilities, so init-less pods schedule with zero init
	// containers).
	initContainers(ctx context.Context, pb *podBuilder, tmpl *wfv1.Template) ([]apiv1.Container, error)

	// argoexecBinaryPath returns where emissary finds the argoexec binary in a
	// user container: copied to /var/run/argo by the init container (legacy),
	// or mounted via the argoexec-bin image volume (init-less).
	argoexecBinaryPath() string

	// extraVolumes returns layout-specific pod volumes: the argoexec-bin image
	// volume for the init-less layout, nothing for the legacy layout.
	extraVolumes(pb *podBuilder) []apiv1.Volume

	// wireArtifactPlugins attaches artifact-plugin containers/mounts to the pod
	// using the layout's scheme: plugin init containers + sidecars driven by
	// `wait` (legacy), or supervisor-driven plugin sidecars (init-less).
	wireArtifactPlugins(ctx context.Context, pb *podBuilder, pod *apiv1.Pod, tmpl *wfv1.Template, config *config.Config) error

	// decorateUserContainer applies layout-specific per-user-container changes.
	// In the init-less layout this mounts the argoexec-bin image volume and (for
	// templates that run a supervisor) sets WAIT_FOR_READY so emissary blocks on
	// the supervisor's ready marker. It is a no-op for the legacy layout.
	//
	// MUST be called once per user container (including every member of a
	// ContainerSet) so the mount / WAIT_FOR_READY land on all members.
	decorateUserContainer(c *apiv1.Container, hasAuxCtr bool)
}

// newPodLayout selects the pod layout once, from the controller's init-less
// feature flag, so build() never re-checks isInitlessPodEnabled().
func newPodLayout(initless bool) podLayout {
	if initless {
		return initlessLayout{}
	}
	return legacyLayout{}
}

// legacyLayout is the default pod shape: an init container stages the template,
// script and input artifacts, and a `wait` container observes main and collects
// outputs.
type legacyLayout struct{}

func (legacyLayout) auxContainer(ctx context.Context, pb *podBuilder, tmpl *wfv1.Template) *apiv1.Container {
	return pb.newWaitContainer(ctx, tmpl)
}

func (legacyLayout) initContainers(ctx context.Context, pb *podBuilder, tmpl *wfv1.Template) ([]apiv1.Container, error) {
	return pb.deps.newInitContainers(ctx, tmpl)
}

func (legacyLayout) argoexecBinaryPath() string {
	return legacyArgoexecBinaryPath
}

func (legacyLayout) extraVolumes(pb *podBuilder) []apiv1.Volume {
	return nil
}

func (legacyLayout) wireArtifactPlugins(ctx context.Context, pb *podBuilder, pod *apiv1.Pod, tmpl *wfv1.Template, config *config.Config) error {
	return pb.deps.addArtifactPluginsLegacy(ctx, pod, tmpl, config)
}

func (legacyLayout) decorateUserContainer(c *apiv1.Container, hasAuxCtr bool) {}

// initlessLayout is the opt-in (beta) pod shape: no init container; a
// `supervisor` container runs concurrently with main and takes on both the
// legacy init container's pre-main work and `wait`'s post-main work, and the
// argoexec binary reaches user containers through an image volume.
type initlessLayout struct{}

func (initlessLayout) auxContainer(ctx context.Context, pb *podBuilder, tmpl *wfv1.Template) *apiv1.Container {
	return pb.deps.newSupervisorContainer(ctx, tmpl)
}

func (initlessLayout) initContainers(ctx context.Context, pb *podBuilder, tmpl *wfv1.Template) ([]apiv1.Container, error) {
	// Init-less pods schedule with zero init containers; the supervisor performs
	// the pre-main responsibilities as a regular container.
	return nil, nil
}

func (initlessLayout) argoexecBinaryPath() string {
	return argoBinExecutorPath
}

func (initlessLayout) extraVolumes(pb *podBuilder) []apiv1.Volume {
	return []apiv1.Volume{pb.deps.buildArgoBinVolume()}
}

func (initlessLayout) wireArtifactPlugins(ctx context.Context, pb *podBuilder, pod *apiv1.Pod, tmpl *wfv1.Template, config *config.Config) error {
	return pb.deps.addArtifactPluginsInitless(ctx, pod, tmpl)
}

func (initlessLayout) decorateUserContainer(c *apiv1.Container, hasAuxCtr bool) {
	if hasAuxCtr {
		// Tell emissary to block on the supervisor's ready marker before reading
		// the template and exec'ing the user command — main and supervisor start
		// concurrently without the usual init barrier. Templates that don't run a
		// supervisor (data / resource-without-logs) skip this: there's nothing to
		// wait for, and the template is delivered via env var.
		c.Env = append(c.Env, apiv1.EnvVar{Name: common.EnvVarWaitForReady, Value: "true"})
	}
	// Read-only mount of the argoexec image volume so emissary can exec from
	// /argo-bin/bin/argoexec. Legacy containers reach argoexec from
	// /var/run/argo/argoexec (populated by the init container).
	c.VolumeMounts = append(c.VolumeMounts, argoBinVolumeMount())
}
