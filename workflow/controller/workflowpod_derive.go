package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/intstr"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// This file holds the derivations of a task's pod-level fields, executor
// containers and deadlines. Each is a function of the values passed to it: the
// template and its boundary template, the workflow spec, controller
// configuration (including execContainerConfig and the executor log flags),
// the artifact repository, the pod layout, the workflow and execution
// deadlines, a clock reading, and values the caller looks up from status or the
// cluster: the node's status (templateTimeoutDeadlines reads its StartedAt and
// Phase) and the executor token's Secret name (executorTokenVolume). None reads
// a live pod, the workflow object or the API, so anything building a task's
// execution environment derives it the same way the pod builder does. The
// apply* and add* functions update the value they are given in place.
//
// podBuilder.build combines the deadlines in this order: taskActiveDeadlineSeconds
// (no pod when start is false), then dropping that for a debug pause
// (debugPauseRequested), then the template's podSpecPatch, then
// templateTimeoutDeadlines and applyTemplateDeadline, so the template timeout
// also lowers an activeDeadlineSeconds the patch set. taskDeadline is the
// ARGO_DEADLINE alongside it.

// applyMainContainerDefaults names the template's user containers and merges
// the controller's main container defaults (Config.MainContainer) under each:
// the defaults first, then the container from the template. Every container is
// named "main" except ContainerSet members, which keep their own names when
// set. ctrs is updated in place.
func applyMainContainerDefaults(ctrs []apiv1.Container, tmpl *wfv1.Template, defaults *apiv1.Container) error {
	for i, c := range ctrs {
		if c.Name == "" || tmpl.GetType() != wfv1.TemplateTypeContainerSet {
			c.Name = common.MainContainerName
		}
		// Allow customization of main container resources.
		if defaults != nil {
			// essentially merge the defaults, then the template, into the container
			a, err := json.Marshal(defaults)
			if err != nil {
				return err
			}
			b, err := json.Marshal(c)
			if err != nil {
				return err
			}
			mergedContainerByte, err := strategicpatch.StrategicMergePatch(a, b, apiv1.Container{})
			if err != nil {
				return err
			}
			c = apiv1.Container{}
			if unmarshalErr := json.Unmarshal(mergedContainerByte, &c); unmarshalErr != nil {
				return unmarshalErr
			}
		}
		ctrs[i] = c
	}
	return nil
}

// taskDeadline is the absolute deadline a task's executor enforces on itself
// (ARGO_DEADLINE): the earlier of the workflow deadline and the execution
// deadline (retryStrategy.backoff.maxDuration after the first attempt started).
// Exit handlers ignore the workflow deadline so they still run after it has
// passed. The zero time means no deadline.
func taskDeadline(workflowDeadline *time.Time, executionDeadline time.Time, onExit bool) time.Time {
	deadline := time.Time{}
	if workflowDeadline != nil && !onExit {
		deadline = *workflowDeadline
	}
	if !executionDeadline.IsZero() && (deadline.IsZero() || executionDeadline.Before(deadline)) {
		deadline = executionDeadline
	}
	return deadline
}

// taskActiveDeadlineSeconds is a task's activeDeadlineSeconds before the
// template timeout is applied (see applyTemplateDeadline): the template's
// activeDeadlineSeconds, lowered to the whole seconds left before the workflow
// deadline as of now. Exit handlers ignore the workflow deadline. start is
// false when less than a whole second is left before the workflow deadline, in
// which case the task must not be started.
func taskActiveDeadlineSeconds(tmpl *wfv1.Template, workflowDeadline *time.Time, onExit bool, now time.Time) (activeDeadlineSeconds *int64, start bool, err error) {
	tmplActiveDeadlineSeconds, err := intstr.Int64(tmpl.ActiveDeadlineSeconds)
	if err != nil {
		return nil, false, err
	}
	if workflowDeadline == nil || onExit { // ignore the workflow deadline for exit handler so they still run if the deadline has passed
		return tmplActiveDeadlineSeconds, true, nil
	}
	wfActiveDeadlineSeconds := int64(workflowDeadline.Sub(now).Seconds())
	switch {
	case wfActiveDeadlineSeconds <= 0:
		return nil, false, nil
	case tmpl.ActiveDeadlineSeconds == nil || wfActiveDeadlineSeconds < *tmplActiveDeadlineSeconds:
		return &wfActiveDeadlineSeconds, true, nil
	default:
		return tmplActiveDeadlineSeconds, true, nil
	}
}

// debugPauseRequested reports whether any of the containers asks the executor
// to pause for debugging, in which case the task's activeDeadlineSeconds from
// the template and the workflow deadline is dropped. A template timeout still
// applies (see applyTemplateDeadline).
func debugPauseRequested(ctrs []apiv1.Container) bool {
	for _, c := range ctrs {
		for _, env := range c.Env {
			if env.Name == "ARGO_DEBUG_PAUSE_BEFORE" || env.Name == "ARGO_DEBUG_PAUSE_AFTER" {
				return true
			}
		}
	}
	return false
}

// templateTimeoutDeadlines returns the node's absolute deadlines from the
// template's timeout and pendingTimeout, both measured from the node's
// StartedAt. pendingDeadline is only set while the node is Pending. It returns
// ErrTimeout if the node is pending past either deadline as of now.
func templateTimeoutDeadlines(tmpl *wfv1.Template, node *wfv1.NodeStatus, now time.Time) (deadline, pendingDeadline *time.Time, err error) {
	if node == nil {
		return nil, nil, nil
	}

	if tmpl.Timeout != "" {
		deadline, err = getTimeoutAsDeadline(&node.StartedAt.Time, tmpl.Timeout)
		if err != nil {
			return nil, nil, err
		}
		if node.Phase == wfv1.NodePending && now.After(*deadline) {
			return nil, nil, ErrTimeout
		}
	}

	if tmpl.PendingTimeout != "" && node.Phase == wfv1.NodePending {
		pendingDeadline, err = getTimeoutAsDeadline(&node.StartedAt.Time, tmpl.PendingTimeout)
		if err != nil {
			return nil, nil, err
		}
		if now.After(*pendingDeadline) {
			return nil, nil, ErrTimeout
		}
	}

	return deadline, pendingDeadline, nil
}

// applyTemplateDeadline lowers activeDeadlineSeconds to the whole seconds left
// before templateDeadline (the node's StartedAt plus the template timeout, see
// templateTimeoutDeadlines) as of now. It only ever lowers it: a smaller
// activeDeadlineSeconds is kept. It fails when the seconds left are 1 or less,
// which, as they are truncated, means under two seconds left, even when
// activeDeadlineSeconds is smaller, so a pod whose timeout has already run out
// is never created. The cut-off has been 1 since template timeouts were added
// (#3686). A nil templateDeadline leaves activeDeadlineSeconds unchanged.
func applyTemplateDeadline(nodeName string, activeDeadlineSeconds *int64, templateDeadline *time.Time, now time.Time) (*int64, error) {
	if templateDeadline == nil {
		return activeDeadlineSeconds, nil
	}
	newActiveDeadlineSeconds := int64(templateDeadline.Sub(now).Seconds())
	if newActiveDeadlineSeconds <= 1 {
		return nil, fmt.Errorf("%s exceeded its deadline", nodeName)
	}
	if activeDeadlineSeconds != nil && *activeDeadlineSeconds <= newActiveDeadlineSeconds {
		return activeDeadlineSeconds, nil
	}
	return &newActiveDeadlineSeconds, nil
}

// podVolumes returns the volumes every task pod starts with: the controller's
// kubeconfig secret (when configured), the shared /var/run/argo and tmp
// emptyDirs, then the template's own volumes.
func podVolumes(kubeConfig *config.KubeConfig, tmpl *wfv1.Template) []apiv1.Volume {
	var volumes []apiv1.Volume
	if kubeConfig != nil {
		name := kubeConfigVolumeName(kubeConfig)
		volumes = append(volumes, apiv1.Volume{
			Name: name,
			VolumeSource: apiv1.VolumeSource{
				Secret: &apiv1.SecretVolumeSource{
					SecretName: kubeConfig.SecretName,
				},
			},
		})
	}

	volumes = append(volumes, volumeVarArgo, volumeTmpDir)
	volumes = append(volumes, tmpl.Volumes...)
	return volumes
}

// podServiceAccount is the service account configuration of a task's pod.
type podServiceAccount struct {
	// serviceAccountName is the pod's service account; empty means the
	// namespace default.
	serviceAccountName string
	// automountServiceAccountToken is set only when the token mount is
	// disabled; nil leaves the Kubernetes default.
	automountServiceAccountToken *bool
	// executorServiceAccountName is the account whose token the argoexec
	// containers mount instead (see executorTokenVolume); empty when unset.
	executorServiceAccountName string
}

// derivePodServiceAccount resolves a task's service accounts, the template's
// settings taking precedence over the workflow's. It rejects disabling the
// token mount without an executor service account, since the executor then
// has no credentials.
func derivePodServiceAccount(tmpl *wfv1.Template, wfSpec *wfv1.WorkflowSpec) (podServiceAccount, error) {
	var sa podServiceAccount
	if tmpl.ServiceAccountName != "" {
		sa.serviceAccountName = tmpl.ServiceAccountName
	} else if wfSpec.ServiceAccountName != "" {
		sa.serviceAccountName = wfSpec.ServiceAccountName
	}

	var automountServiceAccountToken *bool
	if tmpl.AutomountServiceAccountToken != nil {
		automountServiceAccountToken = tmpl.AutomountServiceAccountToken
	} else if wfSpec.AutomountServiceAccountToken != nil {
		automountServiceAccountToken = wfSpec.AutomountServiceAccountToken
	}
	if automountServiceAccountToken != nil && !*automountServiceAccountToken {
		sa.automountServiceAccountToken = automountServiceAccountToken
	}

	sa.executorServiceAccountName = executorServiceAccountName(tmpl, wfSpec)
	if sa.executorServiceAccountName == "" && sa.automountServiceAccountToken != nil {
		return podServiceAccount{}, errors.Errorf(errors.CodeBadRequest, "executor.serviceAccountName must not be empty if automountServiceAccountToken is false")
	}
	return sa, nil
}

// applyTo sets the pod-level service account fields. The executor token volume
// is added separately (executorTokenVolume), since its Secret name is looked up
// from the cluster.
func (sa podServiceAccount) applyTo(spec *apiv1.PodSpec) {
	if sa.serviceAccountName != "" {
		spec.ServiceAccountName = sa.serviceAccountName
	}
	if sa.automountServiceAccountToken != nil {
		spec.AutomountServiceAccountToken = sa.automountServiceAccountToken
	}
}

// executorServiceAccountName resolves the service account the executor
// containers run as: the template's executor SA takes precedence over the
// workflow-level executor SA. Returns "" when neither is set.
func executorServiceAccountName(tmpl *wfv1.Template, wfSpec *wfv1.WorkflowSpec) string {
	if tmpl.Executor != nil && tmpl.Executor.ServiceAccountName != "" {
		return tmpl.Executor.ServiceAccountName
	}
	if wfSpec.Executor != nil && wfSpec.Executor.ServiceAccountName != "" {
		return wfSpec.Executor.ServiceAccountName
	}
	return ""
}

// executorTokenVolume is the volume holding the executor service account's
// token, mounted by every argoexec container (see buildExecContainer).
func executorTokenVolume(tokenSecretName string) apiv1.Volume {
	return apiv1.Volume{
		Name: common.ServiceAccountTokenVolumeName,
		VolumeSource: apiv1.VolumeSource{
			Secret: &apiv1.SecretVolumeSource{
				SecretName: tokenSecretName,
			},
		},
	}
}

// applyDNSConfig applies the workflow's DNS policy and config.
func applyDNSConfig(spec *apiv1.PodSpec, wfSpec *wfv1.WorkflowSpec) {
	if wfSpec.DNSPolicy != nil {
		spec.DNSPolicy = *wfSpec.DNSPolicy
	}

	if wfSpec.DNSConfig != nil {
		spec.DNSConfig = wfSpec.DNSConfig
	}
}

// applySchedulingConstraints sets the pod's node selector, affinity,
// tolerations, scheduler name, priority class, host aliases and security
// context from the template or the workflow spec, the template taking
// precedence. For the node selector, affinity and tolerations only,
// boundaryTemplate (the template of the node's boundary, the enclosing steps or
// DAG; nil when there is none) comes between them: the template wins over the
// boundary, and the boundary over the workflow. Host aliases are combined, the
// workflow's first, rather than replaced.
func applySchedulingConstraints(spec *apiv1.PodSpec, wfSpec *wfv1.WorkflowSpec, tmpl *wfv1.Template, boundaryTemplate *wfv1.Template) {
	// Set nodeSelector (if specified)
	switch {
	case len(tmpl.NodeSelector) > 0:
		spec.NodeSelector = tmpl.NodeSelector
	case boundaryTemplate != nil && len(boundaryTemplate.NodeSelector) > 0:
		spec.NodeSelector = boundaryTemplate.NodeSelector
	case len(wfSpec.NodeSelector) > 0:
		spec.NodeSelector = wfSpec.NodeSelector
	}
	// Set affinity (if specified)
	switch {
	case tmpl.Affinity != nil:
		spec.Affinity = tmpl.Affinity
	case boundaryTemplate != nil && boundaryTemplate.Affinity != nil:
		spec.Affinity = boundaryTemplate.Affinity
	case wfSpec.Affinity != nil:
		spec.Affinity = wfSpec.Affinity
	}
	// Set tolerations (if specified)
	switch {
	case len(tmpl.Tolerations) > 0:
		spec.Tolerations = tmpl.Tolerations
	case boundaryTemplate != nil && len(boundaryTemplate.Tolerations) > 0:
		spec.Tolerations = boundaryTemplate.Tolerations
	case len(wfSpec.Tolerations) > 0:
		spec.Tolerations = wfSpec.Tolerations
	}

	// Set scheduler name (if specified)
	if tmpl.SchedulerName != "" {
		spec.SchedulerName = tmpl.SchedulerName
	} else if wfSpec.SchedulerName != "" {
		spec.SchedulerName = wfSpec.SchedulerName
	}
	// Set priorityClass (if specified)
	if tmpl.PriorityClassName != "" {
		spec.PriorityClassName = tmpl.PriorityClassName
	} else if wfSpec.PodPriorityClassName != "" {
		spec.PriorityClassName = wfSpec.PodPriorityClassName
	}

	// set hostaliases
	spec.HostAliases = append(spec.HostAliases, wfSpec.HostAliases...)
	spec.HostAliases = append(spec.HostAliases, tmpl.HostAliases...)

	// set pod security context
	if tmpl.SecurityContext != nil {
		spec.SecurityContext = tmpl.SecurityContext
	} else if wfSpec.SecurityContext != nil {
		spec.SecurityContext = wfSpec.SecurityContext
	}
}

// applyPodResources sets the pod-level resources and DRA resource claims, the
// template level replacing the workflow level. This is deliberately not part
// of applySchedulingConstraints: the agent pod also applies those, and its
// containers have fixed sizing a task-sized budget would violate.
func applyPodResources(spec *apiv1.PodSpec, wfSpec *wfv1.WorkflowSpec, tmpl *wfv1.Template) {
	if tmpl.PodResources != nil {
		spec.Resources = tmpl.PodResources
	} else if wfSpec.PodResources != nil {
		spec.Resources = wfSpec.PodResources
	}

	// The claims replace rather than merge. The list is cloned so the pod does
	// not share the slice with the spec. podSpecPatch is applied after this and
	// keeps its precedence, merging over these entries by claim name.
	if len(tmpl.ResourceClaims) > 0 {
		spec.ResourceClaims = slices.Clone(tmpl.ResourceClaims)
	} else if len(wfSpec.ResourceClaims) > 0 {
		spec.ResourceClaims = slices.Clone(wfSpec.ResourceClaims)
	}
}

// applyPodMetadata copies the workflow's podMetadata, then the template's
// metadata, onto the pod's labels and annotations, which must be non-nil.
func applyPodMetadata(meta *metav1.ObjectMeta, wfSpec *wfv1.WorkflowSpec, tmpl *wfv1.Template) {
	if wfSpec.PodMetadata != nil {
		// add workflow-level pod annotations and labels
		maps.Copy(meta.Annotations, wfSpec.PodMetadata.Annotations)
		maps.Copy(meta.Labels, wfSpec.PodMetadata.Labels)
	}

	maps.Copy(meta.Annotations, tmpl.Metadata.Annotations)
	maps.Copy(meta.Labels, tmpl.Metadata.Labels)
}

// needsAuxContainer reports whether a task runs with the layout's auxiliary
// executor container (wait, or supervisor in the init-less layout). Data
// templates don't, because argoexec runs as their main container and reports
// their outputs itself; resource templates only do to save their logs, unless
// the layout needs it for them.
func needsAuxContainer(layout podLayout, tmpl *wfv1.Template) bool {
	needsAuxCtr := (tmpl.GetType() != wfv1.TemplateTypeResource && tmpl.GetType() != wfv1.TemplateTypeData) || (tmpl.GetType() == wfv1.TemplateTypeResource && tmpl.SaveLogsAsArtifact())
	// The layout may require the aux container for templates the base rule
	// skips (e.g. init-less `manifestFrom` resources, which need the supervisor
	// to stage the manifest artifact pre-main).
	return needsAuxCtr || layout.templateRequiresAuxContainer(tmpl)
}

// isArchiveLogs determines if a task should archive its logs.
// priorities: controller(on) > template > workflow > controller(off)
func isArchiveLogs(repo *wfv1.ArtifactRepository, wfSpec *wfv1.WorkflowSpec, tmpl *wfv1.Template) bool {
	archiveLogs := repo.IsArchiveLogs()
	if !archiveLogs {
		if wfSpec.ArchiveLogs != nil {
			archiveLogs = *wfSpec.ArchiveLogs
		}
		if tmpl.ArchiveLocation != nil && tmpl.ArchiveLocation.ArchiveLogs != nil {
			archiveLogs = *tmpl.ArchiveLocation.ArchiveLogs
		}
	}
	return archiveLogs
}

// addArchiveLocation conditionally updates the template with the artifact
// repository's location, for the purposes of archiving outputs. The key format
// is copied unresolved. This is skipped for templates which do not need to
// archive anything (archiveLogs is the result of isArchiveLogs), or have
// explicitly set an archive location in the template.
func addArchiveLocation(ctx context.Context, tmpl *wfv1.Template, repo *wfv1.ArtifactRepository, archiveLogs bool) {
	if tmpl.ArchiveLocation.HasLocation() {
		// User explicitly set the location. nothing else to do.
		return
	}
	needLocation := archiveLogs
	artifacts := slices.Concat(tmpl.Inputs.Artifacts, tmpl.Outputs.Artifacts)
	if tmpl.Data != nil {
		if art, exist := tmpl.Data.Source.GetArtifactIfNeeded(); exist {
			artifacts = append(artifacts, *art)
		}
	}
	for _, art := range artifacts {
		if !art.HasLocation() {
			needLocation = true
		}
	}
	logging.RequireLoggerFromContext(ctx).WithField("needLocation", needLocation).Debug(ctx, "addArchiveLocation")
	if !needLocation {
		return
	}
	tmpl.ArchiveLocation = repo.ToArtifactLocation()
	tmpl.ArchiveLocation.ArchiveLogs = &archiveLogs
}

// populateAllPluginArtifactTimeouts populates connection timeouts for all
// plugin artifacts in the template from the configured artifact drivers.
func populateAllPluginArtifactTimeouts(tmpl *wfv1.Template, cfg *config.Config) {
	populateTimeout := func(plugin *wfv1.PluginArtifact) {
		if plugin != nil && plugin.ConnectionTimeoutSeconds == 0 {
			driver, err := cfg.GetArtifactDriver(plugin.Name)
			if err == nil {
				plugin.ConnectionTimeoutSeconds = driver.ConnectionTimeoutSeconds
			}
		}
	}

	// Populate timeout for archive location
	if tmpl.ArchiveLocation != nil {
		populateTimeout(tmpl.ArchiveLocation.Plugin)
	}

	// Populate timeouts for input artifacts
	for i := range tmpl.Inputs.Artifacts {
		populateTimeout(tmpl.Inputs.Artifacts[i].Plugin)
	}

	// Populate timeouts for output artifacts
	for i := range tmpl.Outputs.Artifacts {
		populateTimeout(tmpl.Outputs.Artifacts[i].Plugin)
	}
}

// executorTemplate is the template as the executor receives it (ARGO_TEMPLATE),
// simplified by clearing the already-substituted `inputs.parameters` and
// preserving `inputs.artifacts`.
func executorTemplate(tmpl *wfv1.Template) string {
	simplifiedTmpl := tmpl.DeepCopy()
	simplifiedTmpl.Inputs = wfv1.Inputs{
		Artifacts: simplifiedTmpl.Inputs.Artifacts,
	}
	return wfv1.MustMarshallJSON(simplifiedTmpl)
}

// execContainerConfig is the controller and workflow configuration every
// argoexec container of a pod is built from.
type execContainerConfig struct {
	// image and imagePullPolicy are the effective executor image settings
	// (command-line flags first, then Config.Executor).
	image           string
	imagePullPolicy apiv1.PullPolicy
	// executor is Config.GetExecutor(), never nil.
	executor    *apiv1.Container
	kubeConfig  *config.KubeConfig
	instanceID  string
	workflowUID types.UID
	// executorServiceAccountName is the account whose token the containers
	// mount, which must be on the pod as executorTokenVolume; empty when unset.
	executorServiceAccountName string
}

// executorEnvVars is the environment every argoexec container gets.
func executorEnvVars(cfg execContainerConfig) []apiv1.EnvVar {
	execEnvVars := []apiv1.EnvVar{
		{
			Name: common.EnvVarPodName,
			ValueFrom: &apiv1.EnvVarSource{
				FieldRef: &apiv1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.name",
				},
			},
		},
		{
			Name: common.EnvVarPodUID,
			ValueFrom: &apiv1.EnvVarSource{
				FieldRef: &apiv1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.uid",
				},
			},
		},
		{
			Name:  common.EnvVarWorkflowUID,
			Value: string(cfg.workflowUID),
		},
	}
	if v := cfg.instanceID; v != "" {
		execEnvVars = append(execEnvVars,
			apiv1.EnvVar{Name: common.EnvVarInstanceID, Value: v},
		)
	}
	execEnvVars = append(execEnvVars, cfg.executor.Env...)
	return execEnvVars
}

// buildExecContainer builds an argoexec container named name. When
// cfg.executorServiceAccountName is set it mounts that account's token.
func buildExecContainer(name string, tmpl *wfv1.Template, cfg execContainerConfig) *apiv1.Container {
	exec := apiv1.Container{
		Name:            name,
		Image:           cfg.image,
		ImagePullPolicy: cfg.imagePullPolicy,
		Env:             executorEnvVars(cfg),
		Resources:       cfg.executor.Resources,
		SecurityContext: cfg.executor.SecurityContext,
		Args:            cfg.executor.Args,
	}
	// lock down resource pods by default
	if tmpl.GetType() == wfv1.TemplateTypeResource && exec.SecurityContext == nil {
		exec.SecurityContext = common.MinimalCtrSC()
		// TODO: always set RO FS once #10787 is fixed
		exec.SecurityContext.ReadOnlyRootFilesystem = nil
		if exec.Name != common.InitContainerName && exec.Name != common.WaitContainerName && exec.Name != common.SupervisorContainerName {
			exec.SecurityContext.ReadOnlyRootFilesystem = new(true)
		}
	}
	if cfg.kubeConfig != nil {
		path := cfg.kubeConfig.MountPath
		if path == "" {
			path = common.KubeConfigDefaultMountPath
		}
		name := kubeConfigVolumeName(cfg.kubeConfig)
		exec.VolumeMounts = append(exec.VolumeMounts, apiv1.VolumeMount{
			Name:      name,
			MountPath: path,
			ReadOnly:  true,
			SubPath:   cfg.kubeConfig.SecretKey,
		})
		exec.Args = append(exec.Args, "--kubeconfig="+path)
	}

	if cfg.executorServiceAccountName != "" {
		exec.VolumeMounts = append(exec.VolumeMounts, apiv1.VolumeMount{
			Name:      common.ServiceAccountTokenVolumeName,
			MountPath: common.ServiceAccountTokenMountPath,
			ReadOnly:  true,
		})
	}
	return &exec
}

// standardInitContainer builds the legacy `init` container, which stages the
// template, script and input artifacts before main starts. logOpts are the
// executor's log flags.
func standardInitContainer(tmpl *wfv1.Template, execCfg execContainerConfig, logOpts []string) *apiv1.Container {
	ctr := buildExecContainer(common.InitContainerName, tmpl, execCfg)
	ctr.Command = append([]string{"argoexec", "init"}, logOpts...)
	return ctr
}

// newWaitContainer builds the legacy layout's `wait` container, which
// observes main and collects its outputs, logs and artifacts. logOpts are the
// executor's log flags.
func newWaitContainer(tmpl *wfv1.Template, execCfg execContainerConfig, logOpts []string) *apiv1.Container {
	ctr := buildExecContainer(common.WaitContainerName, tmpl, execCfg)
	ctr.Command = append([]string{"argoexec", "wait"}, logOpts...)
	return ctr
}

// newSupervisorContainer builds the init-less auxiliary container that
// replaces `wait` when initlessPod is enabled. It takes on both the pre-main
// responsibilities of the legacy init container (template write, script
// staging, input artifact download, readiness signaling) and the post-main
// responsibilities of `wait` (observe main, collect outputs/logs/artifacts).
// logOpts are the executor's log flags.
func newSupervisorContainer(tmpl *wfv1.Template, execCfg execContainerConfig, logOpts []string) *apiv1.Container {
	ctr := buildExecContainer(common.SupervisorContainerName, tmpl, execCfg)
	ctr.Command = append([]string{"argoexec", "supervisor"}, logOpts...)
	// Tell the executor it's running in init-less mode so init-less-only
	// code paths (e.g. the input-artifacts overlap fallback in stageArchiveFile)
	// activate. Without this, those paths would not run.
	ctr.Env = append(ctr.Env, apiv1.EnvVar{Name: common.EnvVarInitlessPod, Value: "true"})
	return ctr
}

// argoBinVolume builds the ImageVolume that mounts the argoexec binary into
// `main` for init-less pods. It takes the executor image and pull policy so the
// mounted binary matches the running supervisor.
func argoBinVolume(executorImage string, pullPolicy apiv1.PullPolicy) apiv1.Volume {
	return apiv1.Volume{
		Name: argoBinVolumeName,
		VolumeSource: apiv1.VolumeSource{
			Image: &apiv1.ImageVolumeSource{
				Reference:  executorImage,
				PullPolicy: pullPolicy,
			},
		},
	}
}
