package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/util/strategicpatch"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/errors"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	cmdutil "github.com/argoproj/argo-workflows/v4/util/cmd"
	errorsutil "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/intstr"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
	"github.com/argoproj/argo-workflows/v4/util/template"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/entrypoint"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
	"github.com/argoproj/argo-workflows/v4/workflow/validate"
)

var (
	volumeVarArgo = apiv1.Volume{
		Name: "var-run-argo",
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{},
		},
	}
	volumeMountVarArgo = apiv1.VolumeMount{
		Name:      volumeVarArgo.Name,
		MountPath: common.VarRunArgoPath,
	}
	volumeTmpDir = apiv1.Volume{
		Name: "tmp-dir-argo",
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{},
		},
	}
)

// argoexec-bin image volume: used by init-less pod mode to mount the
// argoexec binary into `main` without running an init container.
const (
	argoBinVolumeName   = common.ArgoExecBinImageVolumeName
	argoBinMountPath    = common.ArgoExecBinMountPath
	argoBinExecutorPath = common.ArgoExecBinPath
)

// legacyArgoexecBinaryPath is where the init container copies argoexec in the
// legacy (non-init-less) layout. Init-less containers instead reach argoexec
// through the argoexec-bin image volume at argoBinExecutorPath.
const legacyArgoexecBinaryPath = common.LegacyArgoExecBinPath

// inputArtifactsVolumeName is the name of the emptyDir volume that holds a
// template's downloaded input artifacts. Shared between the volume definition
// (addInputArtifactsVolumes) and the executor mount derivation
// (inputArtifactExecutorMounts).
const inputArtifactsVolumeName = "input-artifacts"

// scheduleOnDifferentHost adds affinity to prevent retry on the same host when
// retryStrategy.affinity.nodeAntiAffinity{} is specified
func (woc *wfOperationCtx) scheduleOnDifferentHost(ctx context.Context, node *wfv1.NodeStatus, tmpl *wfv1.Template, pod *apiv1.Pod) error {
	if node != nil && pod != nil {
		if retryNode := FindRetryNode(woc.wf.Status.Nodes, node.ID); retryNode != nil {
			// recover template for the retry node
			retryTmpl := tmpl
			if retryNode.TemplateName != "" || retryNode.TemplateRef != nil {
				scope, name := retryNode.GetTemplateScope()
				tmplCtx, err := woc.createTemplateContext(ctx, scope, name)
				if err != nil {
					return err
				}
				_, resolvedTmpl, _, err := tmplCtx.ResolveTemplate(ctx, retryNode)
				if err != nil {
					return err
				}
				retryTmpl = resolvedTmpl
			}
			if retryStrategy := woc.retryStrategy(retryTmpl); retryStrategy != nil {
				RetryOnDifferentHost(retryNode.ID)(*retryStrategy, woc.wf.Status.Nodes, pod)
			}
		}
	}
	return nil
}

type createWorkflowPodOpts struct {
	includeScriptOutput bool
	onExitPod           bool
	executionDeadline   time.Time
}

func (woc *wfOperationCtx) processPodSpecPatch(ctx context.Context, tmpl *wfv1.Template, pod *apiv1.Pod) ([]string, error) {
	podSpecPatches := []string{}
	localParams := make(map[string]string)
	if tmpl.IsPodType() {
		localParams[common.LocalVarPodName] = pod.Name
	}
	toProcess := []string{}
	if woc.execWf.Spec.HasPodSpecPatch() {
		toProcess = append(toProcess, woc.execWf.Spec.PodSpecPatch)
	}
	if tmpl.HasPodSpecPatch() {
		toProcess = append(toProcess, tmpl.PodSpecPatch)
	}

	for _, patch := range toProcess {
		newTmpl := tmpl.DeepCopy()
		newTmpl.PodSpecPatch = patch
		processedTmpl, err := common.ProcessArgs(ctx, newTmpl, &wfv1.Arguments{}, woc.globalParams, localParams, false, woc.wf.Namespace, woc.controller.configMapInformer.GetIndexer())
		if err != nil {
			return nil, errors.Wrap(err, "", "Failed to substitute the PodSpecPatch variables")
		}
		podSpecPatches = append(podSpecPatches, processedTmpl.PodSpecPatch)
	}
	return podSpecPatches, nil
}

func (woc *wfOperationCtx) createWorkflowPod(ctx context.Context, nodeName string, mainCtrs []apiv1.Container, tmpl *wfv1.Template, opts *createWorkflowPodOpts) (*apiv1.Pod, error) {
	nodeID := woc.wf.NodeID(nodeName)

	// we must check to see if the pod exists rather than just optimistically creating the pod and see if we get
	// an `AlreadyExists` error because we won't get that error if there is not enough resources.
	// Performance enhancement: Code later in this func is expensive to execute, so return quickly if we can.
	existing, exists, err := woc.podExists(nodeID)
	if err != nil {
		return nil, err
	}

	if exists {
		woc.log.WithFields(logging.Fields{"podPhase": existing.Status.Phase, "nodeName": nodeName, "nodeID": nodeID}).Debug(ctx, "Skipped pod creation: already exists")
		return existing, nil
	}

	ctx, span := woc.controller.tracing.StartCreateWorkflowPod(ctx, nodeID)
	defer span.End()

	if !woc.GetShutdownStrategy().ShouldExecute(opts.onExitPod) {
		// Do not create pods if we are shutting down
		woc.markNodePhase(ctx, nodeName, wfv1.NodeFailed, fmt.Sprintf("workflow shutdown with strategy: %s", woc.GetShutdownStrategy()))
		return nil, nil
	}
	log := woc.log.WithFields(logging.Fields{"nodeName": nodeName, "nodeID": nodeID})
	ctx = logging.WithLogger(ctx, log)
	tmpl = tmpl.DeepCopy()
	wfSpec := woc.execWf.Spec.DeepCopy()

	for i, c := range mainCtrs {
		if c.Name == "" || tmpl.GetType() != wfv1.TemplateTypeContainerSet {
			c.Name = common.MainContainerName
		}
		// Allow customization of main container resources.
		if ctrDefaults := woc.controller.Config.MainContainer; ctrDefaults != nil {
			// essentially merge the defaults, then the template, into the container
			var a []byte
			a, err = json.Marshal(ctrDefaults)
			if err != nil {
				return nil, err
			}
			var b []byte
			b, err = json.Marshal(c)
			if err != nil {
				return nil, err
			}

			var mergedContainerByte []byte
			mergedContainerByte, err = strategicpatch.StrategicMergePatch(a, b, apiv1.Container{})
			if err != nil {
				return nil, err
			}
			c = apiv1.Container{}
			if unmarshalErr := json.Unmarshal(mergedContainerByte, &c); unmarshalErr != nil {
				return nil, unmarshalErr
			}
		}

		mainCtrs[i] = c
	}

	var activeDeadlineSeconds *int64
	wfDeadline := woc.getWorkflowDeadline()
	tmplActiveDeadlineSeconds, err := intstr.Int64(tmpl.ActiveDeadlineSeconds)
	if err != nil {
		return nil, err
	}
	if wfDeadline == nil || opts.onExitPod { // ignore the workflow deadline for exit handler so they still run if the deadline has passed
		activeDeadlineSeconds = tmplActiveDeadlineSeconds
	} else {
		wfActiveDeadlineSeconds := int64(wfDeadline.Sub(time.Now().UTC()).Seconds())
		switch {
		case wfActiveDeadlineSeconds <= 0:
			return nil, nil
		case tmpl.ActiveDeadlineSeconds == nil || wfActiveDeadlineSeconds < *tmplActiveDeadlineSeconds:
			activeDeadlineSeconds = &wfActiveDeadlineSeconds
		default:
			activeDeadlineSeconds = tmplActiveDeadlineSeconds
		}
	}

	// If the template is marked for debugging no deadline will be set
	for _, c := range mainCtrs {
		for _, env := range c.Env {
			if env.Name == "ARGO_DEBUG_PAUSE_BEFORE" || env.Name == "ARGO_DEBUG_PAUSE_AFTER" {
				activeDeadlineSeconds = nil
			}
		}
	}
	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.GeneratePodName(woc.wf.Name, nodeName, tmpl.Name, nodeID, util.GetWorkflowPodNameVersion(woc.wf)),
			Namespace: woc.wf.Namespace,
			Labels: map[string]string{
				common.LabelKeyWorkflow:  woc.wf.Name, // Allows filtering by pods related to specific workflow
				common.LabelKeyCompleted: "false",     // Allows filtering by incomplete workflow pods
			},
			Annotations: map[string]string{
				common.AnnotationKeyNodeName: nodeName,
				common.AnnotationKeyNodeID:   nodeID,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(woc.wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)),
			},
		},
		Spec: apiv1.PodSpec{
			RestartPolicy:         apiv1.RestartPolicyNever,
			Volumes:               woc.createVolumes(tmpl),
			ActiveDeadlineSeconds: activeDeadlineSeconds,
			ImagePullSecrets:      woc.execWf.Spec.ImagePullSecrets,
		},
	}

	if woc.controller.isInitlessPodEnabled() {
		pod.Spec.Volumes = append(pod.Spec.Volumes, woc.buildArgoBinVolume())
	}

	// Only set trace/span ID annotations when the span context is valid to avoid writing all-zero IDs
	if sc := trace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
		pod.Annotations[common.AnnotationKeyTraceID] = sc.TraceID().String()
		pod.Annotations[common.AnnotationKeySpanID] = sc.SpanID().String()
	}

	if os.Getenv(common.EnvVarPodStatusCaptureFinalizer) == "true" {
		pod.Finalizers = append(pod.Finalizers, common.FinalizerPodStatus)
	}

	if opts.onExitPod {
		// This pod is part of an onExit handler, label it so
		pod.Labels[common.LabelKeyOnExit] = "true"
	}

	if woc.execWf.Spec.HostNetwork != nil {
		pod.Spec.HostNetwork = *woc.execWf.Spec.HostNetwork
	}

	woc.addDNSConfig(pod)

	if woc.controller.Config.InstanceID != "" {
		pod.Labels[common.LabelKeyControllerInstanceID] = woc.controller.Config.InstanceID
	}

	woc.addArchiveLocation(ctx, tmpl)

	// Populate plugin artifact connection timeouts for all input and output artifacts
	woc.populateAllPluginArtifactTimeouts(tmpl)

	err = woc.setupServiceAccount(ctx, pod, tmpl)
	if err != nil {
		return nil, err
	}

	hasAuxCtr := false
	// We do not need the aux (wait/supervisor) container for data templates because
	// argoexec runs as the main container and will perform the job of annotating the
	// outputs or errors, making it redundant. For resource templates we add one to
	// collect logs.
	needsAuxCtr := (tmpl.GetType() != wfv1.TemplateTypeResource && tmpl.GetType() != wfv1.TemplateTypeData) || (tmpl.GetType() == wfv1.TemplateTypeResource && tmpl.SaveLogsAsArtifact())
	// In init-less mode there is no init container to stage input artifacts, and a
	// resource template that isn't archiving logs otherwise runs without a supervisor.
	// A `manifestFrom` resource sources its manifest from an input artifact that
	// `argoexec resource` reads from disk, so it needs the supervisor to download that
	// artifact during pre-main. (Legacy mode is unaffected: its init container stages
	// the artifact regardless of whether a wait container is present.)
	if woc.controller.isInitlessPodEnabled() && tmpl.GetType() == wfv1.TemplateTypeResource && tmpl.Resource.ManifestFrom != nil {
		needsAuxCtr = true
	}
	if needsAuxCtr {
		var auxCtr *apiv1.Container
		if woc.controller.isInitlessPodEnabled() {
			auxCtr = woc.newSupervisorContainer(ctx, tmpl)
		} else {
			auxCtr = woc.newWaitContainer(ctx, tmpl)
		}
		pod.Spec.Containers = append(pod.Spec.Containers, *auxCtr)
		hasAuxCtr = true
	}
	// NOTE: the order of the container list is significant. kubelet will pull, create, and start
	// each container sequentially in the order that they appear in this list. For PNS we want the
	// wait container to start before the main, so that it always has the chance to see the main
	// container's PID and root filesystem.
	pod.Spec.Containers = append(pod.Spec.Containers, mainCtrs...)

	// Configuring default container to be used with commands like "kubectl exec/logs".
	// Select "main" container if it's available. In other case use the last container (can happen when pod created from ContainerSet).
	defaultContainer := pod.Spec.Containers[len(pod.Spec.Containers)-1].Name
	for _, c := range pod.Spec.Containers {
		if c.Name == common.MainContainerName {
			defaultContainer = common.MainContainerName
			break
		}
	}
	pod.Annotations[common.AnnotationKeyDefaultContainer] = defaultContainer

	if podGC := woc.execWf.Spec.PodGC; podGC != nil {
		pod.Annotations[common.AnnotationKeyPodGCStrategy] = fmt.Sprintf("%s/%s", podGC.GetStrategy(), woc.getPodGCDelay(ctx, podGC))
	}

	// Add init containers only if it needs input artifacts. This is also true for
	// script templates (which needs to populate the script).
	// In init-less pod mode the supervisor performs these responsibilities as a
	// regular container, so pods are scheduled with zero init containers.
	if !woc.controller.isInitlessPodEnabled() {
		initContainers, initErr := woc.newInitContainers(ctx, tmpl)
		if initErr != nil {
			return nil, initErr
		}
		pod.Spec.InitContainers = initContainers
	}

	woc.addSchedulingConstraints(ctx, pod, wfSpec, tmpl, nodeName)
	woc.addMetadata(pod, tmpl)

	// Set initial progress from pod metadata if exists.
	if x, ok := pod.Annotations[common.AnnotationKeyProgress]; ok {
		if p, ok := wfv1.ParseProgress(x); ok {
			node, getNodeErr := woc.wf.Status.Nodes.Get(nodeID)
			if getNodeErr != nil {
				log.WithPanic().Error(ctx, "was unable to obtain node")
			}
			node.Progress = p
			woc.wf.Status.Nodes.Set(ctx, nodeID, *node)
		}
	}

	err = woc.addVolumeReferences(ctx, pod, woc.volumes, tmpl, woc.wf.Status.PersistentVolumeClaims)
	if err != nil {
		return nil, err
	}

	err = woc.addInputArtifactsVolumes(ctx, pod, tmpl)
	if err != nil {
		return nil, err
	}

	if tmpl.GetType() == wfv1.TemplateTypeScript {
		addScriptStagingVolume(pod)
	}

	// addInitContainers, addSidecars and addOutputArtifactsVolumes should be called after all
	// volumes have been manipulated in the main container since volumeMounts are mirrored
	addInitContainers(ctx, pod, tmpl)
	err = woc.addSidecars(ctx, pod, tmpl, &woc.controller.Config)
	if err != nil {
		return nil, err
	}
	addOutputArtifactsVolumes(pod, tmpl)

	for i, c := range pod.Spec.InitContainers {
		c.VolumeMounts = append(c.VolumeMounts, volumeMountVarArgo)
		pod.Spec.InitContainers[i] = c
	}

	// simplify template by clearing useless `inputs.parameters` and preserving `inputs.artifacts`.
	simplifiedTmpl := tmpl.DeepCopy()
	simplifiedTmpl.Inputs = wfv1.Inputs{
		Artifacts: simplifiedTmpl.Inputs.Artifacts,
	}
	envVarTemplateValue := wfv1.MustMarshallJSON(simplifiedTmpl)

	// Add standard environment variables, making pod spec larger
	envVars := []apiv1.EnvVar{
		{Name: common.EnvVarNodeID, Value: nodeID},
		{Name: common.EnvVarIncludeScriptOutput, Value: strconv.FormatBool(opts.includeScriptOutput)},
		{Name: common.EnvVarDeadline, Value: woc.getDeadline(opts).Format(time.RFC3339)},
		{Name: common.EnvVarWorkflowName, Value: woc.wf.Name},
	}

	carrier := telemetry.Carrier{SetEnvFunc: func(key, value string) {
		envVars = append(envVars, apiv1.EnvVar{Name: key, Value: value})
	}}
	prop := propagation.TraceContext{}
	prop.Inject(ctx, carrier)
	// only set tick durations/EnvVarProgressFile if progress is enabled.
	// The progress is only monitored if the tick durations are >0.

	if woc.controller.progressPatchTickDuration != 0 && woc.controller.progressFileTickDuration != 0 {
		envVars = append(envVars,
			apiv1.EnvVar{
				Name:  common.EnvVarProgressFile,
				Value: common.ArgoProgressPath,
			},
			apiv1.EnvVar{
				Name:  common.EnvVarProgressPatchTickDuration,
				Value: woc.controller.progressPatchTickDuration.String(),
			},
			apiv1.EnvVar{
				Name:  common.EnvVarProgressFileTickDuration,
				Value: woc.controller.progressFileTickDuration.String(),
			},
		)
	}

	for i, c := range pod.Spec.InitContainers {
		c.Env = append(c.Env, apiv1.EnvVar{Name: common.EnvVarContainerName, Value: c.Name})
		c.Env = append(c.Env, apiv1.EnvVar{Name: common.EnvVarTemplate, Value: envVarTemplateValue})
		c.Env = append(c.Env, envVars...)
		pod.Spec.InitContainers[i] = c
	}
	for i, c := range pod.Spec.Containers {
		c.Env = append(c.Env, apiv1.EnvVar{Name: common.EnvVarContainerName, Value: c.Name})
		c.Env = append(c.Env, envVars...)
		// Init-less supervisor runs as a regular container but still needs
		// ARGO_TEMPLATE set, since it starts concurrently with main and
		// cannot read /var/run/argo/template (which it itself is about to write).
		if c.Name == common.SupervisorContainerName {
			c.Env = append(c.Env, apiv1.EnvVar{Name: common.EnvVarTemplate, Value: envVarTemplateValue})
		}
		// Data and resource-without-logs templates don't run a supervisor in
		// init-less mode; there is no one to write /var/run/argo/template for
		// main's emissary to read. Give main the template via env var instead.
		if woc.controller.isInitlessPodEnabled() && !hasAuxCtr && c.Name == common.MainContainerName {
			c.Env = append(c.Env, apiv1.EnvVar{Name: common.EnvVarTemplate, Value: envVarTemplateValue})
		}
		pod.Spec.Containers[i] = c
	}

	// Perform one last variable substitution here. Some variables come from the from workflow
	// configmap (e.g. archive location) or volumes attribute, and were not substituted
	// in executeTemplate.
	pod, err = substitutePodParams(ctx, pod, woc.globalParams, tmpl)
	if err != nil {
		return nil, err
	}

	// One final check to verify all variables are resolvable for select fields. We are choosing
	// only to check ArchiveLocation for now, since everything else should have been substituted
	// earlier (i.e. in executeTemplate). But archive location is unique in that the variables
	// are formulated from the configmap. We can expand this to other fields as necessary.
	// Legacy mode carries ARGO_TEMPLATE on the init container; init-less mode carries it on
	// supervisor (and on main for templates without a supervisor) instead, so we scan both
	// container lists to cover all configured layouts.
	err = eachContainer(pod, func(c *apiv1.Container) error {
		for _, e := range c.Env {
			if e.Name == common.EnvVarTemplate {
				if uerr := json.Unmarshal([]byte(e.Value), tmpl); uerr != nil {
					return uerr
				}
				for _, obj := range []any{tmpl.ArchiveLocation} {
					if verr := validate.VerifyResolvedVariables(obj); verr != nil {
						return verr
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Apply the patch string from workflow and template
	var podSpecPatchs []string
	podSpecPatchs, err = woc.processPodSpecPatch(ctx, tmpl, pod)
	if err != nil {
		return nil, err
	}
	if len(podSpecPatchs) > 0 {
		var patchedPodSpec *apiv1.PodSpec
		patchedPodSpec, err = util.ApplyPodSpecPatch(pod.Spec, podSpecPatchs...)
		if err != nil {
			return nil, errors.Wrap(err, "", "Error applying PodSpecPatch")
		}
		pod.Spec = *patchedPodSpec
	}

	initless := woc.controller.isInitlessPodEnabled()
	// In init-less mode the emissary binary is mounted at /argo-bin via the
	// argoexec-bin image volume, not copied to /var/run/argo by an init container.
	// K8s image volumes expose the image's root filesystem as-is.
	argoexecBinaryPath := woc.argoexecBinaryPath()
	for i, c := range pod.Spec.Containers {
		if !common.IsArgoSidecar(c.Name) {
			// https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#notes
			if len(c.Command) == 0 {
				var x *entrypoint.Image
				x, err = woc.controller.entrypoint.Lookup(ctx, c.Image, entrypoint.Options{
					Namespace: woc.wf.Namespace, ServiceAccountName: woc.execWf.Spec.ServiceAccountName, ImagePullSecrets: woc.execWf.Spec.ImagePullSecrets,
				})
				if err != nil {
					return nil, fmt.Errorf("failed to look-up entrypoint/cmd for image %q, you must either explicitly specify the command, or list the image's command in the index: https://argo-workflows.readthedocs.io/en/latest/workflow-executors/#emissary-emissary: %w", c.Image, err)
				}
				c.Command = x.Entrypoint
				if c.Args == nil { // check nil rather than length, as zero-length is valid args
					c.Args = x.Cmd
				}
			}
			execCmd := append(append([]string{argoexecBinaryPath, "emissary"}, woc.getExecutorLogOpts(ctx)...), "--")
			c.Command = append(execCmd, c.Command...)
			if initless {
				if hasAuxCtr {
					// Tell emissary to block on the supervisor's ready marker
					// before reading the template and exec'ing the user command
					// — main and supervisor start concurrently without the
					// usual init barrier. Templates that don't run a supervisor
					// (data / resource-without-logs) skip this: there's nothing
					// to wait for, and the template is delivered via env var.
					c.Env = append(c.Env, apiv1.EnvVar{Name: common.EnvVarWaitForReady, Value: "true"})
				}
				// Read-only mount of the argoexec image volume so emissary can
				// exec from /argo-bin/bin/argoexec. Legacy containers reach argoexec
				// from /var/run/argo/argoexec (populated by the init container).
				c.VolumeMounts = append(c.VolumeMounts, argoBinVolumeMount())
			}
		}
		if c.Image == woc.controller.executorImage() {
			// mount tmp dir to the executor container (wait, or supervisor in init-less mode)
			c.VolumeMounts = append(c.VolumeMounts, apiv1.VolumeMount{
				Name:      volumeTmpDir.Name,
				MountPath: "/tmp",
				SubPath:   strconv.Itoa(i),
			})
		}
		// Shared tmp subpath for shared output directory
		outputArtifactPlugins := len(tmpl.Outputs.Artifacts.GetPluginNames(ctx, woc.artifactRepository, wfv1.IncludeLogs, tmpl.ArchiveLocation)) > 0
		if outputArtifactPlugins && common.IsArgoSidecar(c.Name) {
			c.VolumeMounts = append(c.VolumeMounts, apiv1.VolumeMount{
				Name:      volumeTmpDir.Name,
				MountPath: "/tmp/argo/outputs",
				SubPath:   "argo/outputs",
			})
		}
		c.VolumeMounts = append(c.VolumeMounts, volumeMountVarArgo)
		if x := pod.Spec.TerminationGracePeriodSeconds; x != nil {
			c.Env = append(c.Env, apiv1.EnvVar{Name: common.EnvVarTerminationGracePeriodSeconds, Value: fmt.Sprint(*x)})
		}
		pod.Spec.Containers[i] = c
	}

	offloadEnvVarTemplate := false
	offloadContainerArgs := false
	var containerArgsValue string
	var containerArgsName string

	// Check if main container args need offloading (too large for exec)
	for _, c := range pod.Spec.Containers {
		if common.IsArgoSidecar(c.Name) { // skip wait, supervisor, and artifact plugin sidecars
			continue
		}
		if c.Args != nil {
			var argsJSON []byte
			argsJSON, err = json.Marshal(c.Args)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal container args for %s: %w", c.Name, err)
			}
			if len(argsJSON) > common.MaxEnvVarLen {
				offloadContainerArgs = true
				containerArgsValue = string(argsJSON)
				containerArgsName = c.Name
				break
			}
		}
	}

	// Scan every container that may carry ARGO_TEMPLATE. Legacy mode puts it on
	// the init container; init-less mode puts it on the supervisor (always) and
	// on the main container for templates without a supervisor (data /
	// resource-without-logs). Missing any of those when oversized re-introduces
	// the "pod spec too large" rejection that the offload path is here to fix.
	_ = eachContainer(pod, func(c *apiv1.Container) error {
		for _, e := range c.Env {
			if e.Name == common.EnvVarTemplate {
				envVarTemplateValue = e.Value
				if len(envVarTemplateValue) > common.MaxEnvVarLen {
					offloadEnvVarTemplate = true
				}
			}
		}
		return nil
	})

	if offloadEnvVarTemplate || offloadContainerArgs { // Either init container's ARGO_TEMPLATE or main container's args are too large and need offloading
		cmName := pod.Name
		cmData := make(map[string]string)

		// Add template data if needed
		if offloadEnvVarTemplate {
			cmData[common.EnvVarTemplate] = envVarTemplateValue
		}

		// Add container args data if needed
		if offloadContainerArgs {
			cmData[common.EnvVarContainerArgsFile] = containerArgsValue
		}

		cm := &apiv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cmName,
				Namespace: woc.wf.Namespace,
				Labels: map[string]string{
					common.LabelKeyWorkflow: woc.wf.Name,
				},
				Annotations: map[string]string{
					common.AnnotationKeyNodeName: nodeName,
					common.AnnotationKeyNodeID:   nodeID,
				},
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(woc.wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)),
				},
			},
			Data: cmData,
		}
		var created *apiv1.ConfigMap
		created, err = woc.controller.kubeclientset.CoreV1().ConfigMaps(woc.wf.ObjectMeta.Namespace).Create(ctx, cm, metav1.CreateOptions{})
		if err != nil {
			if !apierr.IsAlreadyExists(err) {
				return nil, err
			}
			log.WithField("name", cm.Name).Info(ctx, "Configmap already exists")
		} else {
			log.WithField("name", created.Name).Info(ctx, "Created configmap")
		}

		volumeConfig := apiv1.Volume{
			Name: "argo-env-config",
			VolumeSource: apiv1.VolumeSource{
				ConfigMap: &apiv1.ConfigMapVolumeSource{
					LocalObjectReference: apiv1.LocalObjectReference{
						Name: cmName,
					},
				},
			},
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, volumeConfig)

		volumeMountConfig := apiv1.VolumeMount{
			Name:      volumeConfig.Name,
			MountPath: common.EnvConfigMountPath,
		}

		// Offload ARGO_TEMPLATE env var: replace its inline value with the
		// offloaded sentinel and mount the configmap on every container that
		// carried it (legacy: init container; init-less: supervisor, plus main
		// for supervisor-less templates). emissary's readTemplate falls back
		// to the configmap when it sees the sentinel.
		if offloadEnvVarTemplate {
			_ = eachContainer(pod, func(c *apiv1.Container) error {
				replaced := false
				for j, e := range c.Env {
					if e.Name == common.EnvVarTemplate {
						e.Value = common.EnvVarTemplateOffloaded
						c.Env[j] = e
						replaced = true
					}
				}
				if replaced {
					c.VolumeMounts = append(c.VolumeMounts, volumeMountConfig)
				}
				return nil
			})
		}

		// Handle main conatainers - offload args to file
		if offloadContainerArgs {
			for i, c := range pod.Spec.Containers {
				if c.Name == containerArgsName {
					// Clear the args - they will be read from file
					c.Args = nil
					// Add env var pointing to the args file
					c.Env = append(c.Env, apiv1.EnvVar{
						Name:  common.EnvVarContainerArgsFile,
						Value: common.EnvConfigMountPath + "/" + common.EnvVarContainerArgsFile,
					})
					c.VolumeMounts = append(c.VolumeMounts, volumeMountConfig)
					pod.Spec.Containers[i] = c
					log.WithField("container", containerArgsName).Info(ctx, "Offloaded container args to configmap. Args >128KB will use @filename syntax")
				}
			}
		}
	}

	// Check if the template has exceeded its timeout duration. If it hasn't set the applicable activeDeadlineSeconds
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		log.Warn(ctx, "couldn't retrieve node, will get nil templateDeadline")
	}
	templateDeadline, err := woc.checkTemplateTimeout(tmpl, node)
	if err != nil {
		return nil, err
	}

	if scheduleErr := woc.scheduleOnDifferentHost(ctx, node, tmpl, pod); scheduleErr != nil {
		return nil, scheduleErr
	}

	if templateDeadline != nil && (pod.Spec.ActiveDeadlineSeconds == nil || time.Since(*templateDeadline).Seconds() < float64(*pod.Spec.ActiveDeadlineSeconds)) {
		newActiveDeadlineSeconds := int64(time.Until(*templateDeadline).Seconds())
		if newActiveDeadlineSeconds <= 1 {
			return nil, fmt.Errorf("%s exceeded its deadline", nodeName)
		}
		log.WithFields(logging.Fields{"newActiveDeadlineSeconds": newActiveDeadlineSeconds, "podNamespace": pod.Namespace, "podName": pod.Name}).Debug(ctx, "Setting new activeDeadlineSeconds")
		pod.Spec.ActiveDeadlineSeconds = &newActiveDeadlineSeconds
	}

	reservation := woc.controller.rateLimiter.Reserve()
	if !reservation.OK() {
		reservation.Cancel()
		return nil, ErrResourceRateLimitReached
	}
	delay := reservation.Delay()
	woc.controller.metrics.RecordResourceRateLimiterLatency(ctx, delay.Seconds())
	if delay > 0 {
		reservation.Cancel()
		return nil, ErrResourceRateLimitReached
	}

	log = log.WithField("podName", pod.Name)
	ctx = logging.WithLogger(ctx, log)
	log.Debug(ctx, "Creating Pod")

	created, err := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.ObjectMeta.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		if apierr.IsAlreadyExists(err) {
			// workflow pod names are deterministic. We can get here if the
			// controller fails to persist the workflow after creating the pod.
			log.Info(ctx, "Failed pod creation: already exists")
			// get a reference to the currently existing Pod since the created pod returned before was nil.
			if existing, err = woc.controller.kubeclientset.CoreV1().Pods(woc.wf.ObjectMeta.Namespace).Get(ctx, pod.Name, metav1.GetOptions{}); err == nil {
				return existing, nil
			}
		}
		if errorsutil.IsTransientErr(ctx, err) {
			return nil, err
		}
		log.WithError(err).Info(ctx, "Failed to create pod")
		return nil, errors.InternalWrapError(err)
	}
	log.Info(ctx, "Created pod")
	woc.activePods++
	return created, nil
}

func (woc *wfOperationCtx) podExists(nodeID string) (existing *apiv1.Pod, exists bool, err error) {
	objs, err := woc.controller.PodController.GetPodsByIndex(indexes.NodeIDIndex, woc.wf.Namespace+"/"+nodeID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get pod from informer store: %w", err)
	}

	objectCount := len(objs)

	if objectCount == 0 {
		return nil, false, nil
	}

	if objectCount > 1 {
		return nil, false, fmt.Errorf("expected 1 pod, got %d. This can happen when multiple workflow-controller "+
			"pods are running and both reconciling this Workflow. Check your Argo Workflows installation for a rogue "+
			"workflow-controller. Otherwise, this is a bug", len(objs))
	}

	if existing, ok := objs[0].(*apiv1.Pod); ok {
		return existing, true, nil
	}

	return nil, false, nil
}

func (woc *wfOperationCtx) getDeadline(opts *createWorkflowPodOpts) *time.Time {
	deadline := time.Time{}
	if woc.workflowDeadline != nil && !opts.onExitPod {
		deadline = *woc.workflowDeadline
	}
	if !opts.executionDeadline.IsZero() && (deadline.IsZero() || opts.executionDeadline.Before(deadline)) {
		deadline = opts.executionDeadline
	}
	return &deadline
}

// substitutePodParams returns a pod spec with parameter references substituted as well as pod.name
func substitutePodParams(ctx context.Context, pod *apiv1.Pod, globalParams common.Parameters, tmpl *wfv1.Template) (*apiv1.Pod, error) {
	podParams := globalParams.DeepCopy()
	for _, inParam := range tmpl.Inputs.Parameters {
		podParams["inputs.parameters."+inParam.Name] = inParam.Value.String()
	}
	podParams[common.LocalVarPodName] = pod.Name
	specBytes, err := json.Marshal(pod)
	if err != nil {
		return nil, err
	}
	newSpecBytes, err := template.Replace(ctx, string(specBytes), podParams, true)
	if err != nil {
		return nil, err
	}
	var newSpec apiv1.Pod
	err = json.Unmarshal([]byte(newSpecBytes), &newSpec)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return &newSpec, nil
}

func (woc *wfOperationCtx) standardInitContainer(ctx context.Context, tmpl *wfv1.Template) *apiv1.Container {
	ctr := woc.newExecContainer(common.InitContainerName, tmpl)
	ctr.Command = append([]string{"argoexec", "init"}, woc.getExecutorLogOpts(ctx)...)
	return ctr
}

// artifactContainer builds an argoexec-based artifact container. binaryPath is
// where argoexec is found inside the container: legacy containers reach it at
// /var/run/argo/argoexec (copied there by the init container), while init-less
// plugin sidecars reach it through the argoexec-bin image volume (see
// argoBinExecutorPath) since they run the plugin's own image and are skipped by
// the main image-volume mount loop.
func (woc *wfOperationCtx) artifactContainer(ctx context.Context, tmpl *wfv1.Template, driver config.ArtifactDriver, prefix, binaryPath string, execCmd []string) (*apiv1.Container, error) {
	name := prefix + string(driver.Name)
	ctr := woc.newExecContainer(name, tmpl)
	ctr.Env = append(ctr.Env, apiv1.EnvVar{Name: common.EnvVarContainerName, Value: name})
	ctr.Image = driver.Image
	x, err := woc.controller.entrypoint.Lookup(ctx, driver.Image, entrypoint.Options{
		Namespace: woc.wf.Namespace, ServiceAccountName: woc.execWf.Spec.ServiceAccountName, ImagePullSecrets: woc.execWf.Spec.ImagePullSecrets,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to look-up entrypoint/cmd for image %q, you must either explicitly specify the command, or list the image's command in the index: https://argo-workflows.readthedocs.io/en/latest/workflow-executors/#emissary-emissary: %w", driver.Image, err)
	}

	cmd := []string{binaryPath}
	cmd = append(cmd, execCmd...)
	cmd = append(cmd, woc.getExecutorLogOpts(ctx)...)
	cmd = append(cmd, "--")
	cmd = append(cmd, x.Entrypoint...)
	cmd = append(cmd, driver.Name.SocketPath())
	ctr.Command = cmd

	return ctr, nil
}

func (woc *wfOperationCtx) artifactSidecarGCContainer(ctx context.Context, tmpl *wfv1.Template, driver config.ArtifactDriver) (*apiv1.Container, error) {
	execCmd := []string{"emissary"}
	ctr, err := woc.artifactContainer(ctx, tmpl, driver, common.ArtifactPluginSidecarPrefix, legacyArgoexecBinaryPath, execCmd)
	if err != nil {
		return nil, err
	}
	ctr.VolumeMounts = []apiv1.VolumeMount{driver.Name.VolumeMount()}
	return ctr, nil
}

func (woc *wfOperationCtx) artifactSidecarContainer(ctx context.Context, tmpl *wfv1.Template, driver config.ArtifactDriver) (*apiv1.Container, error) {
	execCmd := []string{"artifact-plugin-sidecar"}
	// In init-less mode the plugin sidecar runs the plugin's own image and is
	// skipped by the main image-volume mount loop, so argoexec is delivered via
	// the argoexec-bin image volume (mounted in addArtifactPluginsInitless).
	// Legacy mode copies argoexec to /var/run/argo via the init container.
	ctr, err := woc.artifactContainer(ctx, tmpl, driver, common.ArtifactPluginSidecarPrefix, woc.argoexecBinaryPath(), execCmd)
	if err != nil {
		return nil, err
	}
	ctr.VolumeMounts = []apiv1.VolumeMount{driver.Name.VolumeMount()}
	return ctr, nil
}

func (woc *wfOperationCtx) artifactInitContainer(ctx context.Context, tmpl *wfv1.Template, driver config.ArtifactDriver) (*apiv1.Container, error) {
	execCmd := []string{"artifact-plugin-init", "--plugin-name", string(driver.Name)}
	return woc.artifactContainer(ctx, tmpl, driver, common.ArtifactPluginInitPrefix, legacyArgoexecBinaryPath, execCmd)
}

func (woc *wfOperationCtx) newInitContainers(ctx context.Context, tmpl *wfv1.Template) ([]apiv1.Container, error) {
	log := logging.RequireLoggerFromContext(ctx)
	plugins := tmpl.Inputs.Artifacts.GetPluginNames(ctx, woc.artifactRepository, wfv1.ExcludeLogs, tmpl.ArchiveLocation)

	log.WithFields(logging.Fields{"plugins": plugins, "pluginLen": len(plugins)}).Debug(ctx, "newInitContainers")
	initContainers := make([]apiv1.Container, len(plugins)+1)
	initContainers[0] = *woc.standardInitContainer(ctx, tmpl)

	drivers, err := woc.controller.Config.GetArtifactDrivers(plugins)
	if err != nil {
		log.WithError(err).Error(ctx, "failed to get artifact drivers")
	}
	for i, driver := range drivers {
		log.WithFields(logging.Fields{"plugin": driver.Name, "i": i}).Debug(ctx, "adding init container")
		ctr, err := woc.artifactInitContainer(ctx, tmpl, driver)
		if err != nil {
			return nil, err
		}

		initContainers[i+1] = *ctr
	}
	return initContainers, nil
}

func (woc *wfOperationCtx) newWaitContainer(ctx context.Context, tmpl *wfv1.Template) *apiv1.Container {
	ctr := woc.newExecContainer(common.WaitContainerName, tmpl)
	ctr.Command = append([]string{"argoexec", "wait"}, woc.getExecutorLogOpts(ctx)...)
	return ctr
}

// newSupervisorContainer builds the init-less auxiliary container that
// replaces `wait` when initlessPod is enabled. It takes on both the pre-main
// responsibilities of the legacy init container (template write, script
// staging, input artifact download, readiness signaling) and the post-main
// responsibilities of `wait` (observe main, collect outputs/logs/artifacts).
func (woc *wfOperationCtx) newSupervisorContainer(ctx context.Context, tmpl *wfv1.Template) *apiv1.Container {
	ctr := woc.newExecContainer(common.SupervisorContainerName, tmpl)
	ctr.Command = append([]string{"argoexec", "supervisor"}, woc.getExecutorLogOpts(ctx)...)
	// Tell the executor it's running in init-less mode so init-less-only
	// code paths (e.g. the input-artifacts overlap fallback in stageArchiveFile)
	// activate. Without this, those paths would not run.
	ctr.Env = append(ctr.Env, apiv1.EnvVar{Name: common.EnvVarInitlessPod, Value: "true"})
	return ctr
}

// argoexecBinaryPath returns where the argoexec binary is found inside a
// container: the argoexec-bin image volume in init-less mode, or the path the
// init container copies it to in the legacy layout.
func (woc *wfOperationCtx) argoexecBinaryPath() string {
	if woc.controller.isInitlessPodEnabled() {
		return argoBinExecutorPath
	}
	return legacyArgoexecBinaryPath
}

// buildArgoBinVolume builds the ImageVolume that mounts the argoexec binary
// into `main` for init-less pods. Reuses the executor image/pullPolicy so
// the mounted binary matches the running supervisor.
func (woc *wfOperationCtx) buildArgoBinVolume() apiv1.Volume {
	return apiv1.Volume{
		Name: argoBinVolumeName,
		VolumeSource: apiv1.VolumeSource{
			Image: &apiv1.ImageVolumeSource{
				Reference:  woc.controller.executorImage(),
				PullPolicy: woc.controller.executorImagePullPolicy(),
			},
		},
	}
}

// argoBinVolumeMount is the read-only mount of the argoexec-bin image volume so
// a container running a non-argoexec image can exec argoexec from argoBinMountPath.
// Used by main-level containers and by plugin sidecars; both must agree on the
// name and path or the binary won't be found, so the mount is built in one place.
func argoBinVolumeMount() apiv1.VolumeMount {
	return apiv1.VolumeMount{
		Name:      argoBinVolumeName,
		MountPath: argoBinMountPath,
		ReadOnly:  true,
	}
}

// buildPluginSidecars emits one artifact-plugin sidecar per unique plugin
// used by either this template's input OR output artifacts. Unlike the
// legacy layout (plugin init containers for Load + sidecars for Save),
// init-less mode has a single sidecar per plugin that supervisor drives
// for both directions. Dedup by plugin identity: a plugin referenced by
// both inputs and outputs produces exactly one container.
func (woc *wfOperationCtx) buildPluginSidecars(ctx context.Context, tmpl *wfv1.Template) ([]apiv1.Container, []wfv1.ArtifactPluginName, error) {
	inputPlugins := tmpl.Inputs.Artifacts.GetPluginNames(ctx, woc.artifactRepository, wfv1.ExcludeLogs, tmpl.ArchiveLocation)
	outputPlugins := tmpl.Outputs.Artifacts.GetPluginNames(ctx, woc.artifactRepository, wfv1.IncludeLogs, tmpl.ArchiveLocation)

	seen := make(map[wfv1.ArtifactPluginName]bool, len(inputPlugins)+len(outputPlugins))
	unionOrdered := make([]wfv1.ArtifactPluginName, 0, len(inputPlugins)+len(outputPlugins))
	for _, n := range inputPlugins {
		if !seen[n] {
			seen[n] = true
			unionOrdered = append(unionOrdered, n)
		}
	}
	for _, n := range outputPlugins {
		if !seen[n] {
			seen[n] = true
			unionOrdered = append(unionOrdered, n)
		}
	}

	drivers, err := woc.controller.Config.GetArtifactDrivers(unionOrdered)
	if err != nil {
		return nil, nil, err
	}
	containers := make([]apiv1.Container, 0, len(drivers))
	for _, driver := range drivers {
		ctr, err := woc.artifactSidecarContainer(ctx, tmpl, driver)
		if err != nil {
			return nil, nil, err
		}
		containers = append(containers, *ctr)
	}
	return containers, inputPlugins, nil
}

func (woc *wfOperationCtx) getExecutorLogOpts(ctx context.Context) []string {
	log := logging.RequireLoggerFromContext(ctx)
	log.WithField("loglevel", string(log.Level())).Info(ctx, "getExecutorLogOpts")
	return []string{"--loglevel", string(log.Level()), "--log-format", woc.controller.executorLogFormat(), "--gloglevel", cmdutil.GetGLogLevel()}
}

func (woc *wfOperationCtx) createEnvVars() []apiv1.EnvVar {
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
			Name:  common.EnvVarWorkflowName,
			Value: woc.wf.Name,
		},
		{
			Name:  common.EnvVarWorkflowUID,
			Value: string(woc.wf.UID),
		},
	}
	if v := woc.controller.Config.InstanceID; v != "" {
		execEnvVars = append(execEnvVars,
			apiv1.EnvVar{Name: common.EnvVarInstanceID, Value: v},
		)
	}
	if woc.controller.Config.Executor != nil {
		execEnvVars = append(execEnvVars, woc.controller.Config.Executor.Env...)
	}
	return execEnvVars
}

// kubeConfigVolumeName returns the configured kubeconfig volume name, or the
// default when none is set.
func kubeConfigVolumeName(kc *config.KubeConfig) string {
	if kc.VolumeName != "" {
		return kc.VolumeName
	}
	return common.KubeConfigDefaultVolumeName
}

func (woc *wfOperationCtx) createVolumes(tmpl *wfv1.Template) []apiv1.Volume {
	var volumes []apiv1.Volume
	if woc.controller.Config.KubeConfig != nil {
		name := kubeConfigVolumeName(woc.controller.Config.KubeConfig)
		volumes = append(volumes, apiv1.Volume{
			Name: name,
			VolumeSource: apiv1.VolumeSource{
				Secret: &apiv1.SecretVolumeSource{
					SecretName: woc.controller.Config.KubeConfig.SecretName,
				},
			},
		})
	}

	volumes = append(volumes, volumeVarArgo, volumeTmpDir)
	volumes = append(volumes, tmpl.Volumes...)
	return volumes
}

func (woc *wfOperationCtx) newExecContainer(name string, tmpl *wfv1.Template) *apiv1.Container {
	exec := apiv1.Container{
		Name:            name,
		Image:           woc.controller.executorImage(),
		ImagePullPolicy: woc.controller.executorImagePullPolicy(),
		Env:             woc.createEnvVars(),
		Resources:       woc.controller.Config.GetExecutor().Resources,
		SecurityContext: woc.controller.Config.GetExecutor().SecurityContext,
		Args:            woc.controller.Config.GetExecutor().Args,
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
	if woc.controller.Config.KubeConfig != nil {
		path := woc.controller.Config.KubeConfig.MountPath
		if path == "" {
			path = common.KubeConfigDefaultMountPath
		}
		name := kubeConfigVolumeName(woc.controller.Config.KubeConfig)
		exec.VolumeMounts = append(exec.VolumeMounts, apiv1.VolumeMount{
			Name:      name,
			MountPath: path,
			ReadOnly:  true,
			SubPath:   woc.controller.Config.KubeConfig.SecretKey,
		})
		exec.Args = append(exec.Args, "--kubeconfig="+path)
	}

	if woc.executorServiceAccountName(tmpl) != "" {
		exec.VolumeMounts = append(exec.VolumeMounts, apiv1.VolumeMount{
			Name:      common.ServiceAccountTokenVolumeName,
			MountPath: common.ServiceAccountTokenMountPath,
			ReadOnly:  true,
		})
	}
	return &exec
}

// addMetadata applies metadata specified in the template
func (woc *wfOperationCtx) addMetadata(pod *apiv1.Pod, tmpl *wfv1.Template) {
	if woc.execWf.Spec.PodMetadata != nil {
		// add workflow-level pod annotations and labels
		maps.Copy(pod.Annotations, woc.execWf.Spec.PodMetadata.Annotations)
		maps.Copy(pod.Labels, woc.execWf.Spec.PodMetadata.Labels)
	}

	maps.Copy(pod.Annotations, tmpl.Metadata.Annotations)
	maps.Copy(pod.Labels, tmpl.Metadata.Labels)
}

// addDNSConfig applies DNSConfig to the pod
func (woc *wfOperationCtx) addDNSConfig(pod *apiv1.Pod) {
	if woc.execWf.Spec.DNSPolicy != nil {
		pod.Spec.DNSPolicy = *woc.execWf.Spec.DNSPolicy
	}

	if woc.execWf.Spec.DNSConfig != nil {
		pod.Spec.DNSConfig = woc.execWf.Spec.DNSConfig
	}
}

// addSchedulingConstraints applies any node selectors or affinity rules to the pod, either set in the workflow or the template
func (woc *wfOperationCtx) addSchedulingConstraints(ctx context.Context, pod *apiv1.Pod, wfSpec *wfv1.WorkflowSpec, tmpl *wfv1.Template, nodeName string) {
	// Get boundaryNode Template (if specified)
	boundaryTemplate, err := woc.GetBoundaryTemplate(ctx, nodeName)
	if err != nil {
		woc.log.WithField("nodeName", nodeName).Warn(ctx, "couldn't get boundaryTemplate")
	}
	// Set nodeSelector (if specified)
	switch {
	case len(tmpl.NodeSelector) > 0:
		pod.Spec.NodeSelector = tmpl.NodeSelector
	case boundaryTemplate != nil && len(boundaryTemplate.NodeSelector) > 0:
		pod.Spec.NodeSelector = boundaryTemplate.NodeSelector
	case len(wfSpec.NodeSelector) > 0:
		pod.Spec.NodeSelector = wfSpec.NodeSelector
	}
	// Set affinity (if specified)
	switch {
	case tmpl.Affinity != nil:
		pod.Spec.Affinity = tmpl.Affinity
	case boundaryTemplate != nil && boundaryTemplate.Affinity != nil:
		pod.Spec.Affinity = boundaryTemplate.Affinity
	case wfSpec.Affinity != nil:
		pod.Spec.Affinity = wfSpec.Affinity
	}
	// Set tolerations (if specified)
	switch {
	case len(tmpl.Tolerations) > 0:
		pod.Spec.Tolerations = tmpl.Tolerations
	case boundaryTemplate != nil && len(boundaryTemplate.Tolerations) > 0:
		pod.Spec.Tolerations = boundaryTemplate.Tolerations
	case len(wfSpec.Tolerations) > 0:
		pod.Spec.Tolerations = wfSpec.Tolerations
	}

	// Set scheduler name (if specified)
	if tmpl.SchedulerName != "" {
		pod.Spec.SchedulerName = tmpl.SchedulerName
	} else if wfSpec.SchedulerName != "" {
		pod.Spec.SchedulerName = wfSpec.SchedulerName
	}
	// Set priorityClass (if specified)
	if tmpl.PriorityClassName != "" {
		pod.Spec.PriorityClassName = tmpl.PriorityClassName
	} else if wfSpec.PodPriorityClassName != "" {
		pod.Spec.PriorityClassName = wfSpec.PodPriorityClassName
	}

	// set hostaliases
	pod.Spec.HostAliases = append(pod.Spec.HostAliases, wfSpec.HostAliases...)
	pod.Spec.HostAliases = append(pod.Spec.HostAliases, tmpl.HostAliases...)

	// set pod security context
	if tmpl.SecurityContext != nil {
		pod.Spec.SecurityContext = tmpl.SecurityContext
	} else if wfSpec.SecurityContext != nil {
		pod.Spec.SecurityContext = wfSpec.SecurityContext
	}
}

// GetBoundaryTemplate get a template through the nodeName
func (woc *wfOperationCtx) GetBoundaryTemplate(ctx context.Context, nodeName string) (*wfv1.Template, error) {
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		woc.log.WithField("nodeName", nodeName).Warn(ctx, "couldn't retrieve node, will get nil templateDeadline")
		return nil, err
	}
	boundaryTmpl, _, err := woc.GetTemplateByBoundaryID(ctx, node.BoundaryID)
	if err != nil {
		return nil, err
	}
	return boundaryTmpl, nil
}

// GetTemplateByBoundaryID get a template through the node's BoundaryID.
func (woc *wfOperationCtx) GetTemplateByBoundaryID(ctx context.Context, boundaryID string) (*wfv1.Template, bool, error) {
	boundaryNode, err := woc.wf.Status.Nodes.Get(boundaryID)
	if err != nil {
		return nil, false, err
	}
	if boundaryNode.TemplateName == "" && boundaryNode.TemplateRef == nil {
		return nil, false, nil
	}
	scope, name := boundaryNode.GetTemplateScope()
	tmplCtx, err := woc.createTemplateContext(ctx, scope, name)
	if err != nil {
		return nil, false, err
	}
	_, boundaryTmpl, templateStored, err := tmplCtx.ResolveTemplate(ctx, boundaryNode)
	if err != nil {
		return nil, templateStored, err
	}
	return boundaryTmpl, templateStored, nil
}

// addVolumeReferences adds any volume mounts that a container/sidecar is referencing, to the pod.spec.volumes
// These are either specified in the workflow.spec.volumes or the workflow.spec.volumeClaimTemplate section
func (woc *wfOperationCtx) addVolumeReferences(ctx context.Context, pod *apiv1.Pod, vols []apiv1.Volume, tmpl *wfv1.Template, pvcs []apiv1.Volume) error {
	switch tmpl.GetType() {
	case wfv1.TemplateTypeContainer, wfv1.TemplateTypeContainerSet, wfv1.TemplateTypeScript, wfv1.TemplateTypeResource, wfv1.TemplateTypeData:
	default:
		return nil
	}

	// getVolByName is a helper to retrieve a volume by its name, either from the volumes or claims section
	getVolByName := func(name string) *apiv1.Volume {
		// Find a volume from template-local volumes.
		for _, vol := range tmpl.Volumes {
			if vol.Name == name {
				return &vol
			}
		}
		// Find a volume from global volumes.
		for _, vol := range vols {
			if vol.Name == name {
				return &vol
			}
		}
		// Find a volume from pvcs.
		for _, pvc := range pvcs {
			if pvc.Name == name {
				return &pvc
			}
		}
		return nil
	}

	addVolumeRef := func(volMounts []apiv1.VolumeMount) error {
		for _, volMnt := range volMounts {
			vol := getVolByName(volMnt.Name)
			if vol == nil {
				return errors.Errorf(errors.CodeBadRequest, "volume '%s' not found in workflow spec", volMnt.Name)
			}
			found := false
			for _, v := range pod.Spec.Volumes {
				if v.Name == vol.Name {
					found = true
					break
				}
			}
			if !found {
				if pod.Spec.Volumes == nil {
					pod.Spec.Volumes = make([]apiv1.Volume, 0)
				}
				pod.Spec.Volumes = append(pod.Spec.Volumes, *vol)
			}
		}
		return nil
	}

	err := addVolumeRef(tmpl.GetVolumeMounts())
	if err != nil {
		return err
	}

	for _, container := range tmpl.InitContainers {
		err := addVolumeRef(container.VolumeMounts)
		if err != nil {
			return err
		}
	}

	for _, sidecar := range tmpl.Sidecars {
		err := addVolumeRef(sidecar.VolumeMounts)
		if err != nil {
			return err
		}
	}

	volumes, volumeMounts := createSecretVolumesAndMounts(tmpl)
	pod.Spec.Volumes = append(pod.Spec.Volumes, volumes...)

	for idx, container := range pod.Spec.Containers {
		if container.Name == common.WaitContainerName || container.Name == common.SupervisorContainerName {
			pod.Spec.Containers[idx].VolumeMounts = append(pod.Spec.Containers[idx].VolumeMounts, volumeMounts...)
			break
		}
	}
	for idx, container := range pod.Spec.InitContainers {
		if container.Name == common.InitContainerName {
			pod.Spec.InitContainers[idx].VolumeMounts = append(pod.Spec.InitContainers[idx].VolumeMounts, volumeMounts...)
			break
		}
	}
	if tmpl.Data != nil {
		for idx, container := range pod.Spec.Containers {
			if container.Name == common.MainContainerName {
				pod.Spec.Containers[idx].VolumeMounts = append(pod.Spec.Containers[idx].VolumeMounts, volumeMounts...)
				break
			}
		}
	}
	artifactVolumeMounts := woc.createArtifactVolumeMounts(ctx, tmpl)
	pod.Spec.Volumes = append(pod.Spec.Volumes, artifactVolumeMounts...)

	return nil
}

// addInputArtifactsVolumes sets up the artifacts volume to the pod to support input artifacts to containers.
// In order support input artifacts, the init container shares a emptydir volume with the main container.
// It is the responsibility of the init container to load all artifacts to the mounted emptydir location.
// (e.g. /inputs/artifacts/CODE). The shared emptydir is mapped to the user's desired location in the main
// container.
//
// It is possible that a user specifies overlapping paths of an artifact path with a volume mount,
// (e.g. user wants an external volume mounted at /src, while simultaneously wanting an input artifact
// placed at /src/some/subdirectory). When this occurs, we need to prevent the duplicate bind mounting of
// overlapping volumes, since the outer volume will not see the changes made in the artifact emptydir.
//
// To prevent overlapping bind mounts, both the controller and executor will recognize the overlap between
// the explicit volume mount and the artifact emptydir and prevent all uses of the emptydir for purposes of
// loading data. The controller will omit mounting the emptydir to the artifact path, and the executor
// will load the artifact in the in user's volume (as opposed to the emptydir)
func (woc *wfOperationCtx) addInputArtifactsVolumes(ctx context.Context, pod *apiv1.Pod, tmpl *wfv1.Template) error {
	if len(tmpl.Inputs.Artifacts) == 0 {
		return nil
	}
	// In init-less mode the emissary stages each input artifact by clearing
	// art.Path (os.RemoveAll) and symlinking the downloaded file into place. If
	// art.Path is an ancestor of a mounted volume (e.g. art.Path /data with a
	// volume mounted at /data/shared), that RemoveAll would recurse into and
	// destroy the live volume. Reject the configuration up front. (Legacy mode is
	// unaffected: it delivers via a kubelet bind mount and never deletes, so the
	// check is scoped to init-less to avoid rejecting configs that work today.)
	if woc.controller.isInitlessPodEnabled() {
		for _, art := range tmpl.Inputs.Artifacts {
			if art.Path == "" {
				continue
			}
			if mnt := common.FindVolumeMountNestedUnderPath(tmpl, art.Path); mnt != nil {
				return errors.Errorf(errors.CodeBadRequest,
					"input artifact %q path %q is an ancestor of volume mount %q (%s); this is not supported in init-less pod mode because staging the artifact would clear the mounted volume",
					art.Name, art.Path, mnt.Name, mnt.MountPath)
			}
		}
	}
	artVol := apiv1.Volume{
		Name: inputArtifactsVolumeName,
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{},
		},
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, artVol)
	logger := logging.RequireLoggerFromContext(ctx)
	logger.Debug(ctx, "addInputArtifactsVolumes")
	// The shared input-artifacts mount plus the extra mounts mirroring the
	// user's volumes (used when the executor must load artifacts into a
	// user-specified volume instead of the emptydir). Shared verbatim with the
	// init-less plugin sidecars via initlessPluginSidecarArtifactMounts.
	executorMounts := woc.inputArtifactExecutorMounts(tmpl)
	applyArtifactMounts := func(c *apiv1.Container) {
		c.VolumeMounts = append(c.VolumeMounts, executorMounts...)
	}
	for i, initCtr := range pod.Spec.InitContainers {
		logger.WithFields(logging.Fields{"name": initCtr.Name}).Debug(ctx, "checking init container volumes")
		if initCtr.Name == common.InitContainerName || common.IsArtifactPluginInit(initCtr.Name) {
			logger.WithFields(logging.Fields{"name": initCtr.Name}).Debug(ctx, "adding input artifacts volume mount")
			applyArtifactMounts(&initCtr)
			pod.Spec.InitContainers[i] = initCtr
		}
	}
	// In init-less mode, supervisor runs as a regular container and still
	// needs the same input-artifacts mount to load artifacts onto. Plugin
	// sidecars also need this mount (they receive the same path over gRPC
	// from supervisor's Load call and write to it directly), but they
	// haven't been appended to pod.Spec.Containers yet — that happens in
	// addArtifactPluginsInitless, called later by addSidecars. The plugin-
	// sidecar mount wiring lives there alongside the sidecar construction.
	if woc.controller.isInitlessPodEnabled() {
		for i, c := range pod.Spec.Containers {
			if c.Name == common.SupervisorContainerName {
				logger.WithFields(logging.Fields{"name": c.Name}).Debug(ctx, "adding input artifacts volume mount (init-less supervisor)")
				applyArtifactMounts(&c)
				pod.Spec.Containers[i] = c
			}
		}
	}

	for i, c := range pod.Spec.Containers {
		if c.Name != common.MainContainerName {
			continue
		}
		if woc.controller.isInitlessPodEnabled() {
			// Init-less mode can't use per-artifact SubPath mounts: kubelet
			// races the supervisor and pre-creates each SubPath as an empty
			// directory in the shared emptyDir before supervisor writes the
			// file (main and supervisor start concurrently). Instead, mount
			// the whole input-artifacts volume on main at a known location
			// and let the emissary symlink each artifact into place post-ready.
			// Keep the mount read-write to preserve the legacy bind-mount
			// behavior where a user writing through art.Path mutates the file
			// in the shared emptyDir.
			c.VolumeMounts = append(c.VolumeMounts, apiv1.VolumeMount{
				Name:      artVol.Name,
				MountPath: common.ExecutorArtifactBaseDir,
			})
			pod.Spec.Containers[i] = c
			continue
		}
		for _, art := range tmpl.Inputs.Artifacts {
			err := art.CleanPath()
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "error in inputs.artifacts.%s: %s", art.Name, err.Error())
			}
			if !art.HasLocationOrKey() && art.Optional {
				woc.log.WithFields(logging.Fields{"name": art.Name, "path": art.Path}).Info(ctx, "skip volume mount")
				continue
			}
			overlap := common.FindOverlappingVolume(tmpl, art.Path)
			if overlap != nil {
				// artifact path overlaps with a mounted volume. do not mount the
				// artifacts emptydir to the main container. init would have copied
				// the artifact to the user's volume instead
				woc.log.WithFields(logging.Fields{"name": art.Name, "path": art.Path, "overlapName": overlap.Name, "overlapMountPath": overlap.MountPath}).Debug(ctx, "skip volume mount")
				continue
			}
			volMount := apiv1.VolumeMount{
				Name:      artVol.Name,
				MountPath: art.Path,
				SubPath:   art.Name,
			}
			c.VolumeMounts = append(c.VolumeMounts, volMount)
		}
		pod.Spec.Containers[i] = c
	}
	return nil
}

func (woc *wfOperationCtx) createArtifactVolumeMounts(ctx context.Context, tmpl *wfv1.Template) []apiv1.Volume {
	artifactVolumeMounts := []apiv1.Volume{}
	plugins := tmpl.Outputs.Artifacts.GetPluginNames(ctx, woc.artifactRepository, wfv1.IncludeLogs, tmpl.ArchiveLocation)

	for _, plugin := range plugins {
		artifactVolumeMounts = append(artifactVolumeMounts, plugin.Volume())
	}
	return artifactVolumeMounts
}

// addOutputArtifactsVolumes mirrors any volume mounts in the main container to the wait sidecar
// and artifact-plugin sidecars.
// For any output artifacts that were produced in mounted volumes (e.g. PVCs, emptyDirs), the
// wait container will collect the artifacts directly from volumeMount instead of `docker cp`-ing
// them to the wait sidecar. In order for this to work, we mirror all volume mounts in the main
// container under a well-known path.
func addOutputArtifactsVolumes(pod *apiv1.Pod, tmpl *wfv1.Template) {
	if tmpl.GetType() == wfv1.TemplateTypeResource || tmpl.GetType() == wfv1.TemplateTypeData {
		return
	}

	// Collect main container volume mounts to mirror
	var mainContainerMounts []apiv1.VolumeMount
	for _, c := range pod.Spec.Containers {
		if c.Name == common.MainContainerName {
			mainContainerMounts = c.VolumeMounts
			break
		}
	}

	// Mirror main container mounts to wait container and plugin sidecar containers
	for i, c := range pod.Spec.Containers {
		if !common.IsArgoSidecar(c.Name) {
			continue
		}
		// Track mountPaths already on the sidecar so we don't add a duplicate.
		// In init-less mode the supervisor and plugin sidecars already received
		// the mirrored user-volume mounts (at ExecutorMainFilesystemDir) from
		// addInputArtifactsVolumes / addArtifactPluginsInitless; mirroring main's
		// mounts here would re-add the same mountPaths, which Kubernetes rejects
		// at admission. Legacy mode has no such pre-existing mounts, so this is a
		// no-op there.
		existingMountPaths := make(map[string]bool, len(c.VolumeMounts))
		for _, vm := range c.VolumeMounts {
			existingMountPaths[vm.MountPath] = true
		}
		for _, mnt := range mainContainerMounts {
			if util.IsWindowsUNCPath(mnt.MountPath, tmpl) {
				continue
			}
			mnt.MountPath = filepath.Join(common.ExecutorMainFilesystemDir, mnt.MountPath)
			// ReadOnly is needed to be false for overlapping volume mounts
			mnt.ReadOnly = false
			if existingMountPaths[mnt.MountPath] {
				continue
			}
			existingMountPaths[mnt.MountPath] = true
			c.VolumeMounts = append(c.VolumeMounts, mnt)
		}
		pod.Spec.Containers[i] = c
	}
}

// addArchiveLocation conditionally updates the template with the default artifact repository
// information configured in the controller, for the purposes of archiving outputs. This is skipped
// for templates which do not need to archive anything, or have explicitly set an archive location
// in the template.
func (woc *wfOperationCtx) addArchiveLocation(ctx context.Context, tmpl *wfv1.Template) {
	if tmpl.ArchiveLocation.HasLocation() {
		// User explicitly set the location. nothing else to do.
		return
	}
	archiveLogs := woc.IsArchiveLogs(tmpl)
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
	tmpl.ArchiveLocation = woc.artifactRepository.ToArtifactLocation()
	tmpl.ArchiveLocation.ArchiveLogs = &archiveLogs
}

// populateAllPluginArtifactTimeouts populates connection timeouts for all plugin artifacts in the template
func (woc *wfOperationCtx) populateAllPluginArtifactTimeouts(tmpl *wfv1.Template) {
	populateTimeout := func(plugin *wfv1.PluginArtifact) {
		if plugin != nil && plugin.ConnectionTimeoutSeconds == 0 {
			driver, err := woc.controller.Config.GetArtifactDriver(plugin.Name)
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

// IsArchiveLogs determines if container should archive logs
// priorities: controller(on) > template > workflow > controller(off)
func (woc *wfOperationCtx) IsArchiveLogs(tmpl *wfv1.Template) bool {
	archiveLogs := woc.artifactRepository.IsArchiveLogs()
	if !archiveLogs {
		if woc.execWf.Spec.ArchiveLogs != nil {
			archiveLogs = *woc.execWf.Spec.ArchiveLogs
		}
		if tmpl.ArchiveLocation != nil && tmpl.ArchiveLocation.ArchiveLogs != nil {
			archiveLogs = *tmpl.ArchiveLocation.ArchiveLogs
		}
	}
	return archiveLogs
}

// executorServiceAccountName resolves the service account the executor
// containers run as: the template's executor SA takes precedence over the
// workflow-level executor SA. Returns "" when neither is set.
func (woc *wfOperationCtx) executorServiceAccountName(tmpl *wfv1.Template) string {
	if tmpl.Executor != nil && tmpl.Executor.ServiceAccountName != "" {
		return tmpl.Executor.ServiceAccountName
	}
	if woc.execWf.Spec.Executor != nil && woc.execWf.Spec.Executor.ServiceAccountName != "" {
		return woc.execWf.Spec.Executor.ServiceAccountName
	}
	return ""
}

// setupServiceAccount sets up service account and token.
func (woc *wfOperationCtx) setupServiceAccount(ctx context.Context, pod *apiv1.Pod, tmpl *wfv1.Template) error {
	if tmpl.ServiceAccountName != "" {
		pod.Spec.ServiceAccountName = tmpl.ServiceAccountName
	} else if woc.execWf.Spec.ServiceAccountName != "" {
		pod.Spec.ServiceAccountName = woc.execWf.Spec.ServiceAccountName
	}

	var automountServiceAccountToken *bool
	if tmpl.AutomountServiceAccountToken != nil {
		automountServiceAccountToken = tmpl.AutomountServiceAccountToken
	} else if woc.execWf.Spec.AutomountServiceAccountToken != nil {
		automountServiceAccountToken = woc.execWf.Spec.AutomountServiceAccountToken
	}
	if automountServiceAccountToken != nil && !*automountServiceAccountToken {
		pod.Spec.AutomountServiceAccountToken = automountServiceAccountToken
	}

	executorServiceAccountName := woc.executorServiceAccountName(tmpl)
	if executorServiceAccountName != "" {
		tokenName, err := woc.getServiceAccountTokenName(ctx, executorServiceAccountName)
		if err != nil {
			return err
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, apiv1.Volume{
			Name: common.ServiceAccountTokenVolumeName,
			VolumeSource: apiv1.VolumeSource{
				Secret: &apiv1.SecretVolumeSource{
					SecretName: tokenName,
				},
			},
		})
	} else if automountServiceAccountToken != nil && !*automountServiceAccountToken {
		return errors.Errorf(errors.CodeBadRequest, "executor.serviceAccountName must not be empty if automountServiceAccountToken is false")
	}
	return nil
}

// addScriptStagingVolume sets up a shared staging volume between the init
// (or supervisor, in init-less mode) and main container for the purpose of
// holding the script source code for script templates.
func addScriptStagingVolume(pod *apiv1.Pod) {
	volName := "argo-staging"
	stagingVol := apiv1.Volume{
		Name: volName,
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{},
		},
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, stagingVol)

	volMount := apiv1.VolumeMount{
		Name:      volName,
		MountPath: common.ExecutorStagingEmptyDir,
	}
	for i, initCtr := range pod.Spec.InitContainers {
		if initCtr.Name == common.InitContainerName {
			initCtr.VolumeMounts = append(initCtr.VolumeMounts, volMount)
			pod.Spec.InitContainers[i] = initCtr
			break
		}
	}
	for i, ctr := range pod.Spec.Containers {
		if ctr.Name == common.SupervisorContainerName {
			ctr.VolumeMounts = append(ctr.VolumeMounts, volMount)
			pod.Spec.Containers[i] = ctr
			break
		}
	}
	found := false
	for i, ctr := range pod.Spec.Containers {
		if ctr.Name == common.MainContainerName {
			volMount := apiv1.VolumeMount{
				Name:      volName,
				MountPath: common.ExecutorStagingEmptyDir,
			}
			ctr.VolumeMounts = append(ctr.VolumeMounts, volMount)
			pod.Spec.Containers[i] = ctr
			found = true
			break
		}
	}
	if !found {
		panic("Unable to locate main container")
	}
}

// addInitContainers adds all init containers to the pod spec of the step
// Optionally volume mounts from the main container to the init containers
func addInitContainers(ctx context.Context, pod *apiv1.Pod, tmpl *wfv1.Template) {
	mainCtr := findMainContainer(pod)
	for _, ctr := range tmpl.InitContainers {
		logging.RequireLoggerFromContext(ctx).WithField("name", ctr.Name).Debug(ctx, "Adding init container")
		if mainCtr != nil && ctr.MirrorVolumeMounts != nil && *ctr.MirrorVolumeMounts {
			mirrorVolumeMounts(ctx, mainCtr, &ctr.Container)
		}
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, ctr.Container)
	}
}

// addSidecars adds all sidecars to the pod spec of the step.
// Optionally volume mounts from the main container to the sidecar
func (woc *wfOperationCtx) addSidecars(ctx context.Context, pod *apiv1.Pod, tmpl *wfv1.Template, config *config.Config) error {
	mainCtr := findMainContainer(pod)
	for _, sidecar := range tmpl.Sidecars {
		logging.RequireLoggerFromContext(ctx).WithField("name", sidecar.Name).Debug(ctx, "Adding sidecar container")
		if mainCtr != nil && sidecar.MirrorVolumeMounts != nil && *sidecar.MirrorVolumeMounts {
			mirrorVolumeMounts(ctx, mainCtr, &sidecar.Container)
		}
		pod.Spec.Containers = append(pod.Spec.Containers, sidecar.Container)
	}
	return woc.addArtifactPlugins(ctx, pod, tmpl, config)
}

func (woc *wfOperationCtx) addArtifactPlugins(ctx context.Context, pod *apiv1.Pod, tmpl *wfv1.Template, config *config.Config) error {
	if woc.controller.isInitlessPodEnabled() {
		return woc.addArtifactPluginsInitless(ctx, pod, tmpl)
	}
	plugins := tmpl.Outputs.Artifacts.GetPluginNames(ctx, woc.artifactRepository, wfv1.IncludeLogs, tmpl.ArchiveLocation)
	drivers, err := config.GetArtifactDrivers(plugins)
	if err != nil {
		return err
	}

	sidecarNames := make([]string, len(drivers))
	for i, driver := range drivers {
		logging.RequireLoggerFromContext(ctx).WithField("name", driver.Name).Debug(ctx, "Adding artifact plugin")
		var ctr *apiv1.Container
		ctr, err = woc.artifactSidecarContainer(ctx, tmpl, driver)
		if err != nil {
			return err
		}
		pod.Spec.Containers = append(pod.Spec.Containers, *ctr)
		sidecarNames[i] = ctr.Name
	}

	// Mount plugin volumes to wait container if it exists
	waitCtrIndex, err := util.FindWaitCtrIndex(pod)
	if err == nil {
		waitCtr := &pod.Spec.Containers[waitCtrIndex]
		for _, driver := range drivers {
			waitCtr.VolumeMounts = append(waitCtr.VolumeMounts, driver.Name.VolumeMount())
		}
		waitCtr.Env = append(waitCtr.Env, apiv1.EnvVar{Name: common.EnvVarArtifactPluginNames, Value: common.JoinPluginNames(sidecarNames)})
		pod.Spec.Containers[waitCtrIndex] = *waitCtr
	}

	return nil
}

// addArtifactPluginsInitless emits one plugin sidecar per unique plugin
// used by either the template's input OR output artifacts, dedup'd by
// plugin identity. Unlike legacy mode (input plugins → init containers,
// output plugins → sidecars), init-less mode has a single sidecar that
// the supervisor drives for both Load (pre-main) and Save (post-main).
func (woc *wfOperationCtx) addArtifactPluginsInitless(ctx context.Context, pod *apiv1.Pod, tmpl *wfv1.Template) error {
	sidecars, inputPlugins, err := woc.buildPluginSidecars(ctx, tmpl)
	if err != nil {
		return err
	}
	if len(sidecars) == 0 {
		return nil
	}

	// The supervisor is what drives every plugin sidecar (Load pre-main, Save
	// post-main). Locate it BEFORE mutating the pod: if there is no supervisor
	// (e.g. a data template), appending the sidecars would leave them orphaned
	// with nothing to invoke Load/Save against, so bail without touching the pod.
	// FindAuxiliaryCtrIndex returns -1 (and an error) when no wait/supervisor
	// container is present; the error only signals "not found", so we key off
	// the index and skip gracefully rather than treating it as a failure.
	supervisorIdx, _ := util.FindAuxiliaryCtrIndex(pod)
	if supervisorIdx < 0 {
		logging.RequireLoggerFromContext(ctx).WithField("template", tmpl.Name).
			Warn(ctx, "skipping artifact plugin sidecars: no supervisor container to drive them")
		return nil
	}

	// Each plugin sidecar mounts its plugin's socket volume. Output-plugin
	// volumes are already on the pod (createArtifactVolumeMounts scans outputs),
	// but input-only plugins have no such volume, so their sidecar would mount a
	// volume absent from the pod and admission would reject it. Add the input
	// plugins' socket volumes here, deduped by name against volumes already
	// present (covers plugins used for both input and output).
	existingVols := make(map[string]bool, len(pod.Spec.Volumes))
	for _, v := range pod.Spec.Volumes {
		existingVols[v.Name] = true
	}
	for _, p := range inputPlugins {
		vol := p.Volume()
		if !existingVols[vol.Name] {
			pod.Spec.Volumes = append(pod.Spec.Volumes, vol)
			existingVols[vol.Name] = true
		}
	}

	// Each plugin sidecar needs the same input-artifacts mount as supervisor
	// (and the mirrored user volumes for the overlap case): supervisor calls
	// the plugin's Load over gRPC with a path under ExecutorArtifactBaseDir
	// (or ExecutorMainFilesystemDir for user-volume overlaps), and the plugin
	// process writes to that path directly. Without these mounts the plugin
	// would not be able to populate the file. Returns nil when there are no
	// input artifacts (the "input-artifacts" volume isn't on the pod).
	pluginSidecarMounts := woc.initlessPluginSidecarArtifactMounts(tmpl)
	sidecarNames := make([]string, 0, len(sidecars))
	// Capture each sidecar's socket-only mounts (set by artifactSidecarContainer)
	// BEFORE extending the sidecar with the binary/input-artifacts mounts.
	// Supervisor needs only the socket mounts — it already received the shared
	// input-artifacts and mirrored user-volume mounts from addInputArtifactsVolumes,
	// and runs the argoexec image directly so it has no need for the argoexec-bin
	// volume; appending those via the sidecar's mounts would produce duplicate
	// mountPaths and fail Kubernetes pod admission.
	socketMounts := make([]apiv1.VolumeMount, 0, len(sidecars))
	for _, ctr := range sidecars {
		socketMounts = append(socketMounts, ctr.VolumeMounts...)
	}
	// Plugin sidecars run the plugin's own image, so they need argoexec
	// delivered via the argoexec-bin image volume (their command points at
	// argoBinExecutorPath — see artifactSidecarContainer).
	argoBinMount := argoBinVolumeMount()
	for i := range sidecars {
		sidecars[i].VolumeMounts = append(sidecars[i].VolumeMounts, argoBinMount)
		sidecars[i].VolumeMounts = append(sidecars[i].VolumeMounts, pluginSidecarMounts...)
		pod.Spec.Containers = append(pod.Spec.Containers, sidecars[i])
		sidecarNames = append(sidecarNames, sidecars[i].Name)
	}

	// Appending sidecars above may have reallocated the backing array, so
	// re-resolve the supervisor by its (still valid) index.
	supervisor := &pod.Spec.Containers[supervisorIdx]
	supervisor.VolumeMounts = append(supervisor.VolumeMounts, socketMounts...)
	// All plugin sidecars supervisor drives for Save post-main.
	supervisor.Env = append(supervisor.Env, apiv1.EnvVar{Name: common.EnvVarArtifactPluginNames, Value: common.JoinPluginNames(sidecarNames)})
	// Subset supervisor calls Load on pre-main.
	inputNames := make([]string, 0, len(inputPlugins))
	for _, n := range inputPlugins {
		inputNames = append(inputNames, string(n))
	}
	if len(inputNames) > 0 {
		supervisor.Env = append(supervisor.Env, apiv1.EnvVar{Name: common.EnvVarInputArtifactPluginNames, Value: common.JoinPluginNames(inputNames)})
	}
	pod.Spec.Containers[supervisorIdx] = *supervisor
	return nil
}

// inputArtifactExecutorMounts returns the volume mounts an argoexec container
// (legacy init container, init-less supervisor, or init-less plugin sidecar)
// needs to load this template's input artifacts: the shared "input-artifacts"
// emptyDir at ExecutorArtifactBaseDir, plus mirrored user volume mounts at
// ExecutorMainFilesystemDir for the overlapping-volume case. Callers that may
// run with no input artifacts must guard themselves — the "input-artifacts"
// volume is only added to the pod when the template has input artifacts.
func (woc *wfOperationCtx) inputArtifactExecutorMounts(tmpl *wfv1.Template) []apiv1.VolumeMount {
	mounts := []apiv1.VolumeMount{{
		Name:      inputArtifactsVolumeName,
		MountPath: common.ExecutorArtifactBaseDir,
	}}
	for _, mnt := range tmpl.GetVolumeMounts() {
		if util.IsWindowsUNCPath(mnt.MountPath, tmpl) {
			continue
		}
		mnt.MountPath = filepath.Join(common.ExecutorMainFilesystemDir, mnt.MountPath)
		mounts = append(mounts, mnt)
	}
	return mounts
}

// initlessPluginSidecarArtifactMounts returns the volume mounts each plugin
// sidecar needs in init-less mode (see inputArtifactExecutorMounts), or nil
// when the template has no input artifacts (the "input-artifacts" volume is
// not on the pod).
func (woc *wfOperationCtx) initlessPluginSidecarArtifactMounts(tmpl *wfv1.Template) []apiv1.VolumeMount {
	if len(tmpl.Inputs.Artifacts) == 0 {
		return nil
	}
	return woc.inputArtifactExecutorMounts(tmpl)
}

// secretVolumesAndMountsFromMap turns a name→Volume map into the parallel
// volume and read-only volume-mount slices the pod spec needs, mounting each
// secret under SecretVolMountPath.
func secretVolumesAndMountsFromMap(m map[string]apiv1.Volume) ([]apiv1.Volume, []apiv1.VolumeMount) {
	var secretVolumes []apiv1.Volume
	var secretVolMounts []apiv1.VolumeMount
	for volMountName, val := range m {
		secretVolumes = append(secretVolumes, val)
		secretVolMounts = append(secretVolMounts, apiv1.VolumeMount{
			Name:      volMountName,
			MountPath: common.SecretVolMountPath + "/" + val.Name,
			ReadOnly:  true,
		})
	}
	return secretVolumes, secretVolMounts
}

// createSecretVolumesAndMounts will retrieve and create Volumes and Volumemount object for Pod
func createSecretVolumesAndMounts(tmpl *wfv1.Template) ([]apiv1.Volume, []apiv1.VolumeMount) {
	allVolumesMap := make(map[string]apiv1.Volume)
	uniqueKeyMap := make(map[string]bool)

	createArchiveLocationSecret(tmpl, allVolumesMap, uniqueKeyMap)

	for _, art := range tmpl.Outputs.Artifacts {
		createSecretVolume(allVolumesMap, art, uniqueKeyMap)
	}
	for _, art := range tmpl.Inputs.Artifacts {
		createSecretVolume(allVolumesMap, art, uniqueKeyMap)
	}

	if tmpl.Data != nil {
		if art, needed := tmpl.Data.Source.GetArtifactIfNeeded(); needed {
			createSecretVolume(allVolumesMap, *art, uniqueKeyMap)
		}
	}

	return secretVolumesAndMountsFromMap(allVolumesMap)
}

func createArchiveLocationSecret(tmpl *wfv1.Template, volMap map[string]apiv1.Volume, uniqueKeyMap map[string]bool) {
	if tmpl.ArchiveLocation == nil {
		return
	}
	createSecretVolumesFromArtifactLocations(volMap, []*wfv1.ArtifactLocation{tmpl.ArchiveLocation}, uniqueKeyMap)
}

func createSecretVolume(volMap map[string]apiv1.Volume, art wfv1.Artifact, keyMap map[string]bool) {
	createSecretVolumesFromArtifactLocations(volMap, []*wfv1.ArtifactLocation{&art.ArtifactLocation}, keyMap)
}

func createSecretVolumesAndMountsFromArtifactLocations(artifactLocations []*wfv1.ArtifactLocation) ([]apiv1.Volume, []apiv1.VolumeMount) {
	allVolumesMap := make(map[string]apiv1.Volume)
	uniqueKeyMap := make(map[string]bool)

	createSecretVolumesFromArtifactLocations(allVolumesMap, artifactLocations, uniqueKeyMap)

	return secretVolumesAndMountsFromMap(allVolumesMap)
}

func createSecretVolumesFromArtifactLocations(volMap map[string]apiv1.Volume, artifactLocations []*wfv1.ArtifactLocation, keyMap map[string]bool) {
	for _, artifactLocation := range artifactLocations {
		if artifactLocation == nil {
			continue
		}
		switch {
		case artifactLocation.S3 != nil:
			createSecretVal(volMap, artifactLocation.S3.AccessKeySecret, keyMap)
			createSecretVal(volMap, artifactLocation.S3.SecretKeySecret, keyMap)
			if artifactLocation.S3.SessionTokenSecret != nil {
				createSecretVal(volMap, artifactLocation.S3.SessionTokenSecret, keyMap)
			}
			sseCUsed := artifactLocation.S3.EncryptionOptions != nil && artifactLocation.S3.EncryptionOptions.EnableEncryption && artifactLocation.S3.EncryptionOptions.ServerSideCustomerKeySecret != nil
			if sseCUsed {
				createSecretVal(volMap, artifactLocation.S3.EncryptionOptions.ServerSideCustomerKeySecret, keyMap)
			}
			if artifactLocation.S3.CASecret != nil {
				createSecretVal(volMap, artifactLocation.S3.CASecret, keyMap)
			}
		case artifactLocation.Git != nil:
			createSecretVal(volMap, artifactLocation.Git.UsernameSecret, keyMap)
			createSecretVal(volMap, artifactLocation.Git.PasswordSecret, keyMap)
			createSecretVal(volMap, artifactLocation.Git.SSHPrivateKeySecret, keyMap)
		case artifactLocation.Artifactory != nil:
			createSecretVal(volMap, artifactLocation.Artifactory.UsernameSecret, keyMap)
			createSecretVal(volMap, artifactLocation.Artifactory.PasswordSecret, keyMap)
		case artifactLocation.HDFS != nil:
			createSecretVal(volMap, artifactLocation.HDFS.KrbCCacheSecret, keyMap)
			createSecretVal(volMap, artifactLocation.HDFS.KrbKeytabSecret, keyMap)
		case artifactLocation.OSS != nil:
			createSecretVal(volMap, artifactLocation.OSS.AccessKeySecret, keyMap)
			createSecretVal(volMap, artifactLocation.OSS.SecretKeySecret, keyMap)
		case artifactLocation.GCS != nil:
			createSecretVal(volMap, artifactLocation.GCS.ServiceAccountKeySecret, keyMap)
		case artifactLocation.HTTP != nil && artifactLocation.HTTP.Auth != nil:
			createSecretVal(volMap, artifactLocation.HTTP.Auth.BasicAuth.UsernameSecret, keyMap)
			createSecretVal(volMap, artifactLocation.HTTP.Auth.BasicAuth.PasswordSecret, keyMap)
			createSecretVal(volMap, artifactLocation.HTTP.Auth.ClientCert.ClientCertSecret, keyMap)
			createSecretVal(volMap, artifactLocation.HTTP.Auth.ClientCert.ClientKeySecret, keyMap)
			createSecretVal(volMap, artifactLocation.HTTP.Auth.OAuth2.ClientIDSecret, keyMap)
			createSecretVal(volMap, artifactLocation.HTTP.Auth.OAuth2.ClientSecretSecret, keyMap)
			createSecretVal(volMap, artifactLocation.HTTP.Auth.OAuth2.TokenURLSecret, keyMap)
		case artifactLocation.Azure != nil:
			createSecretVal(volMap, artifactLocation.Azure.AccountKeySecret, keyMap)
		}
	}
}

func createSecretVal(volMap map[string]apiv1.Volume, secret *apiv1.SecretKeySelector, keyMap map[string]bool) {
	if secret == nil || secret.Name == "" || secret.Key == "" {
		return
	}
	if vol, ok := volMap[secret.Name]; ok {
		key := apiv1.KeyToPath{
			Key:  secret.Key,
			Path: secret.Key,
		}
		if val := keyMap[secret.Name+"-"+secret.Key]; !val {
			keyMap[secret.Name+"-"+secret.Key] = true
			vol.Secret.Items = append(vol.Secret.Items, key)
		}
	} else {
		volume := apiv1.Volume{
			Name: secret.Name,
			VolumeSource: apiv1.VolumeSource{
				Secret: &apiv1.SecretVolumeSource{
					SecretName: secret.Name,
					Items: []apiv1.KeyToPath{
						{
							Key:  secret.Key,
							Path: secret.Key,
						},
					},
				},
			},
		}
		keyMap[secret.Name+"-"+secret.Key] = true
		volMap[secret.Name] = volume
	}
}

// eachContainer applies fn to every init and regular container, in pod order
// (init containers first), passing a pointer into the backing array so any
// mutation fn makes persists. It stops at and returns the first error fn
// returns. The init-before-regular order matches how callers historically
// concatenated the two lists, so scans that overwrite shared state observe the
// same last-write-wins behavior.
func eachContainer(pod *apiv1.Pod, fn func(c *apiv1.Container) error) error {
	for i := range pod.Spec.InitContainers {
		if err := fn(&pod.Spec.InitContainers[i]); err != nil {
			return err
		}
	}
	for i := range pod.Spec.Containers {
		if err := fn(&pod.Spec.Containers[i]); err != nil {
			return err
		}
	}
	return nil
}

// findMainContainer finds main container
func findMainContainer(pod *apiv1.Pod) *apiv1.Container {
	for _, ctr := range pod.Spec.Containers {
		if common.MainContainerName == ctr.Name {
			return &ctr
		}
	}
	return nil
}

// mirrorVolumeMounts mirrors volumeMounts of source container to target container
func mirrorVolumeMounts(ctx context.Context, sourceContainer, targetContainer *apiv1.Container) {
	for _, volMnt := range sourceContainer.VolumeMounts {
		if targetContainer.VolumeMounts == nil {
			targetContainer.VolumeMounts = make([]apiv1.VolumeMount, 0)
		}
		logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{"name": volMnt.Name, "containerName": targetContainer.Name}).Debug(ctx, "Adding volume mount")
		targetContainer.VolumeMounts = append(targetContainer.VolumeMounts, volMnt)
	}
}
