package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasttemplate"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"

	"github.com/argoproj/argo/v2/errors"
	"github.com/argoproj/argo/v2/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/v2/pkg/apis/workflow/v1alpha1"
	errorsutil "github.com/argoproj/argo/v2/util/errors"
	"github.com/argoproj/argo/v2/util/intstr"
	"github.com/argoproj/argo/v2/workflow/common"
	"github.com/argoproj/argo/v2/workflow/util"
)

// Reusable k8s pod spec portions used in workflow pods
var (
	// volumePodMetadata makes available the pod metadata available as a file
	// to the executor's init and sidecar containers. Specifically, the template
	// of the pod is stored as an annotation
	volumePodMetadata = apiv1.Volume{
		Name: common.PodMetadataVolumeName,
		VolumeSource: apiv1.VolumeSource{
			DownwardAPI: &apiv1.DownwardAPIVolumeSource{
				Items: []apiv1.DownwardAPIVolumeFile{
					{
						Path: common.PodMetadataAnnotationsVolumePath,
						FieldRef: &apiv1.ObjectFieldSelector{
							APIVersion: "v1",
							FieldPath:  "metadata.annotations",
						},
					},
				},
			},
		},
	}
	volumeMountPodMetadata = apiv1.VolumeMount{
		Name:      volumePodMetadata.Name,
		MountPath: common.PodMetadataMountPath,
	}

	hostPathSocket = apiv1.HostPathSocket
)

func (woc *wfOperationCtx) getVolumeMountDockerSock(tmpl *wfv1.Template) apiv1.VolumeMount {
	return apiv1.VolumeMount{
		Name:      common.DockerSockVolumeName,
		MountPath: getDockerSockPath(tmpl),
		ReadOnly:  getDockerSockReadOnly(tmpl),
	}
}

func getDockerSockReadOnly(tmpl *wfv1.Template) bool {
	return !hasWindowsOSNodeSelector(tmpl.NodeSelector)
}

func getDockerSockPath(tmpl *wfv1.Template) string {
	if hasWindowsOSNodeSelector(tmpl.NodeSelector) {
		return "\\\\.\\pipe\\docker_engine"
	}

	return "/var/run/docker.sock"
}

func getVolumeHostPathType(tmpl *wfv1.Template) *apiv1.HostPathType {
	if hasWindowsOSNodeSelector(tmpl.NodeSelector) {
		return nil
	}

	return &hostPathSocket
}

func hasWindowsOSNodeSelector(nodeSelector map[string]string) bool {
	if nodeSelector == nil {
		return false
	}

	if os, keyExists := nodeSelector["kubernetes.io/os"]; keyExists && os == "windows" {
		return true
	}

	return false
}

func (woc *wfOperationCtx) getVolumeDockerSock(tmpl *wfv1.Template) apiv1.Volume {
	dockerSockPath := getDockerSockPath(tmpl)

	if woc.controller.Config.DockerSockPath != "" {
		dockerSockPath = woc.controller.Config.DockerSockPath
	}

	// volumeDockerSock provides the wait container direct access to the minion's host docker daemon.
	// The primary purpose of this is to make available `docker cp` to collect an output artifact
	// from a container. Alternatively, we could use `kubectl cp`, but `docker cp` avoids the extra
	// hop to the kube api server.
	return apiv1.Volume{
		Name: common.DockerSockVolumeName,
		VolumeSource: apiv1.VolumeSource{
			HostPath: &apiv1.HostPathVolumeSource{
				Path: dockerSockPath,
				Type: getVolumeHostPathType(tmpl),
			},
		},
	}
}

func (woc *wfOperationCtx) hasPodSpecPatch(tmpl *wfv1.Template) bool {
	return woc.execWf.Spec.HasPodSpecPatch() || tmpl.HasPodSpecPatch()
}

type createWorkflowPodOpts struct {
	includeScriptOutput bool
	onExitPod           bool
	executionDeadline   time.Time
}

func (woc *wfOperationCtx) createWorkflowPod(nodeName string, mainCtr apiv1.Container, tmpl *wfv1.Template, opts *createWorkflowPodOpts) (*apiv1.Pod, error) {
	nodeID := woc.wf.NodeID(nodeName)
	woc.log.Debugf("Creating Pod: %s (%s)", nodeName, nodeID)

	// we must check to see if the pod exists rather than just optimistically creating the pod and see if we get
	// an `AlreadyExists` error because we won't get that error if there is not enough resources.
	// Performance enhancement: Code later in this func is expensive to execute, so return quickly if we can.
	obj, exists, err := woc.controller.podInformer.GetStore().Get(cache.ExplicitKey(woc.wf.Namespace + "/" + nodeID))
	if err != nil {
		return nil, fmt.Errorf("failed to get pod from informer store: %w", err)
	}
	if exists {
		existing, ok := obj.(*apiv1.Pod)
		if ok {
			woc.log.WithField("podPhase", existing.Status.Phase).Debugf("Skipped pod %s (%s) creation: already exists", nodeName, nodeID)
			return existing, nil
		}
	}

	if !woc.GetShutdownStrategy().ShouldExecute(opts.onExitPod) {
		// Do not create pods if we are shutting down
		woc.markNodePhase(nodeName, wfv1.NodeSkipped, fmt.Sprintf("workflow shutdown with strategy: %s", woc.GetShutdownStrategy()))
		return nil, nil
	}

	tmpl = tmpl.DeepCopy()
	wfSpec := woc.execWf.Spec.DeepCopy()

	mainCtr.Name = common.MainContainerName
	// Allow customization of main container resources.
	if isResourcesSpecified(woc.controller.Config.MainContainer) {
		mainCtr.Resources = *woc.controller.Config.MainContainer.Resources.DeepCopy()
	}
	// Container resources in workflow spec takes precedence over the main container's configuration in controller.
	if isResourcesSpecified(tmpl.Container) && tmpl.Container.Name == common.MainContainerName {
		mainCtr.Resources = *tmpl.Container.Resources.DeepCopy()
	}

	var activeDeadlineSeconds *int64
	wfDeadline := woc.getWorkflowDeadline()
	tmplActiveDeadlineSeconds, err := intstr.Int64(tmpl.ActiveDeadlineSeconds)
	if err != nil {
		return nil, err
	}
	if wfDeadline == nil || opts.onExitPod { //ignore the workflow deadline for exit handler so they still run if the deadline has passed
		activeDeadlineSeconds = tmplActiveDeadlineSeconds
	} else {
		wfActiveDeadlineSeconds := int64((*wfDeadline).Sub(time.Now().UTC()).Seconds())
		if wfActiveDeadlineSeconds <= 0 {
			return nil, nil
		} else if tmpl.ActiveDeadlineSeconds == nil || wfActiveDeadlineSeconds < *tmplActiveDeadlineSeconds {
			activeDeadlineSeconds = &wfActiveDeadlineSeconds
		} else {
			activeDeadlineSeconds = tmplActiveDeadlineSeconds
		}
	}

	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nodeID,
			Namespace: woc.wf.ObjectMeta.Namespace,
			Labels: map[string]string{
				common.LabelKeyWorkflow:  woc.wf.ObjectMeta.Name, // Allows filtering by pods related to specific workflow
				common.LabelKeyCompleted: "false",                // Allows filtering by incomplete workflow pods
			},
			Annotations: map[string]string{
				common.AnnotationKeyNodeName: nodeName,
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

	if opts.onExitPod {
		// This pod is part of an onExit handler, label it so
		pod.ObjectMeta.Labels[common.LabelKeyOnExit] = "true"
	}

	if woc.execWf.Spec.HostNetwork != nil {
		pod.Spec.HostNetwork = *woc.execWf.Spec.HostNetwork
	}

	if woc.execWf.Spec.DNSPolicy != nil {
		pod.Spec.DNSPolicy = *woc.execWf.Spec.DNSPolicy
	}

	if woc.execWf.Spec.DNSConfig != nil {
		pod.Spec.DNSConfig = woc.execWf.Spec.DNSConfig
	}

	if woc.controller.Config.InstanceID != "" {
		pod.ObjectMeta.Labels[common.LabelKeyControllerInstanceID] = woc.controller.Config.InstanceID
	}
	if woc.controller.GetContainerRuntimeExecutor() == common.ContainerRuntimeExecutorPNS {
		pod.Spec.ShareProcessNamespace = pointer.BoolPtr(true)
	}

	err = woc.addArchiveLocation(tmpl)
	if err != nil {
		return nil, err
	}

	err = woc.setupServiceAccount(pod, tmpl)
	if err != nil {
		return nil, err
	}

	if tmpl.GetType() != wfv1.TemplateTypeResource {
		// we do not need the wait container for resource templates because
		// argoexec runs as the main container and will perform the job of
		// annotating the outputs or errors, making the wait container redundant.
		waitCtr, err := woc.newWaitContainer(tmpl)
		if err != nil {
			return nil, err
		}
		pod.Spec.Containers = append(pod.Spec.Containers, *waitCtr)
	}
	// NOTE: the order of the container list is significant. kubelet will pull, create, and start
	// each container sequentially in the order that they appear in this list. For PNS we want the
	// wait container to start before the main, so that it always has the chance to see the main
	// container's PID and root filesystem.
	pod.Spec.Containers = append(pod.Spec.Containers, mainCtr)

	// Add init container only if it needs input artifacts. This is also true for
	// script templates (which needs to populate the script)
	if len(tmpl.Inputs.Artifacts) > 0 || tmpl.GetType() == wfv1.TemplateTypeScript {
		initCtr := woc.newInitContainer(tmpl)
		pod.Spec.InitContainers = []apiv1.Container{initCtr}
	}

	addSchedulingConstraints(pod, wfSpec, tmpl)
	woc.addMetadata(pod, tmpl, opts)

	err = addVolumeReferences(pod, woc.volumes, tmpl, woc.wf.Status.PersistentVolumeClaims)
	if err != nil {
		return nil, err
	}

	err = woc.addInputArtifactsVolumes(pod, tmpl)
	if err != nil {
		return nil, err
	}

	if tmpl.GetType() == wfv1.TemplateTypeScript {
		addScriptStagingVolume(pod)
	}

	// addInitContainers, addSidecars and addOutputArtifactsVolumes should be called after all
	// volumes have been manipulated in the main container since volumeMounts are mirrored
	err = addInitContainers(pod, tmpl)
	if err != nil {
		return nil, err
	}
	err = addSidecars(pod, tmpl)
	if err != nil {
		return nil, err
	}
	addOutputArtifactsVolumes(pod, tmpl)

	// Set the container template JSON in pod annotations, which executor examines for things like
	// artifact location/path.
	tmplBytes, err := json.Marshal(tmpl)
	if err != nil {
		return nil, err
	}
	pod.ObjectMeta.Annotations[common.AnnotationKeyTemplate] = string(tmplBytes)

	// Perform one last variable substitution here. Some variables come from the from workflow
	// configmap (e.g. archive location) or volumes attribute, and were not substituted
	// in executeTemplate.
	pod, err = substitutePodParams(pod, woc.globalParams, tmpl)
	if err != nil {
		return nil, err
	}

	// One final check to verify all variables are resolvable for select fields. We are choosing
	// only to check ArchiveLocation for now, since everything else should have been substituted
	// earlier (i.e. in executeTemplate). But archive location is unique in that the variables
	// are formulated from the configmap. We can expand this to other fields as necessary.
	err = json.Unmarshal([]byte(pod.ObjectMeta.Annotations[common.AnnotationKeyTemplate]), &tmpl)
	if err != nil {
		return nil, err
	}
	for _, obj := range []interface{}{tmpl.ArchiveLocation} {
		err = verifyResolvedVariables(obj)
		if err != nil {
			return nil, err
		}
	}

	// Apply the patch string from template
	if woc.hasPodSpecPatch(tmpl) {
		jsonstr, err := json.Marshal(pod.Spec)
		if err != nil {
			return nil, errors.Wrap(err, "", "Fail to marshal the Pod spec")
		}

		tmpl.PodSpecPatch, err = util.PodSpecPatchMerge(woc.wf, tmpl)

		if err != nil {
			return nil, errors.Wrap(err, "", "Fail to marshal the Pod spec")
		}

		// Final substitution for workflow level PodSpecPatch
		localParams := make(map[string]string)
		if tmpl.IsPodType() {
			localParams[common.LocalVarPodName] = woc.wf.NodeID(nodeName)
		}
		tmpl, err := common.ProcessArgs(tmpl, &wfv1.Arguments{}, woc.globalParams, localParams, false)
		if err != nil {
			return nil, errors.Wrap(err, "", "Fail to substitute the PodSpecPatch variables")
		}

		var spec apiv1.PodSpec

		if !util.ValidateJsonStr(tmpl.PodSpecPatch, spec) {
			return nil, errors.New("", "Invalid PodSpecPatch String")
		}

		modJson, err := strategicpatch.StrategicMergePatch(jsonstr, []byte(tmpl.PodSpecPatch), apiv1.PodSpec{})

		if err != nil {
			return nil, errors.Wrap(err, "", "Error occurred during strategic merge patch")
		}
		err = json.Unmarshal(modJson, &pod.Spec)
		if err != nil {
			return nil, errors.Wrap(err, "", "Error in Unmarshalling after merge the patch")
		}
	}

	// Check if the template has exceeded its timeout duration. If it hasn't set the applicable activeDeadlineSeconds
	node := woc.wf.GetNodeByName(nodeName)
	templateDeadline, err := woc.checkTemplateTimeout(tmpl, node)
	if err != nil {
		return nil, err
	}

	if templateDeadline != nil && (pod.Spec.ActiveDeadlineSeconds == nil || time.Since(*templateDeadline).Seconds() < float64(*pod.Spec.ActiveDeadlineSeconds)) {
		newActiveDeadlineSeconds := int64(time.Until(*templateDeadline).Seconds())
		if newActiveDeadlineSeconds <= 1 {
			return nil, fmt.Errorf("%s exceeded its deadline", nodeName)
		}
		woc.log.Debugf("Setting new activeDeadlineSeconds %d for pod %s/%s due to templateDeadline", newActiveDeadlineSeconds, pod.Namespace, pod.Name)
		pod.Spec.ActiveDeadlineSeconds = &newActiveDeadlineSeconds
	}

	created, err := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.ObjectMeta.Namespace).Create(pod)
	if err != nil {
		if apierr.IsAlreadyExists(err) {
			// workflow pod names are deterministic. We can get here if the
			// controller fails to persist the workflow after creating the pod.
			woc.log.Infof("Failed pod %s (%s) creation: already exists", nodeName, nodeID)
			return created, nil
		}
		if errorsutil.IsTransientErr(err) {
			return nil, err
		}
		woc.log.Infof("Failed to create pod %s (%s): %v", nodeName, nodeID, err)
		return nil, errors.InternalWrapError(err)
	}
	woc.log.Infof("Created pod: %s (%s)", nodeName, created.Name)
	woc.activePods++
	return created, nil
}

// substitutePodParams returns a pod spec with parameter references substituted as well as pod.name
func substitutePodParams(pod *apiv1.Pod, globalParams common.Parameters, tmpl *wfv1.Template) (*apiv1.Pod, error) {
	podParams := globalParams.DeepCopy()
	for _, inParam := range tmpl.Inputs.Parameters {
		podParams["inputs.parameters."+inParam.Name] = inParam.Value.String()
	}
	podParams[common.LocalVarPodName] = pod.Name
	specBytes, err := json.Marshal(pod)
	if err != nil {
		return nil, err
	}
	fstTmpl, err := fasttemplate.NewTemplate(string(specBytes), "{{", "}}")
	if err != nil {
		return nil, fmt.Errorf("unable to parse argo varaible: %w", err)
	}
	newSpecBytes, err := common.Replace(fstTmpl, podParams, true)
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

func (woc *wfOperationCtx) newInitContainer(tmpl *wfv1.Template) apiv1.Container {
	ctr := woc.newExecContainer(common.InitContainerName, tmpl)
	ctr.Command = []string{"argoexec", "init"}
	return *ctr
}

func (woc *wfOperationCtx) newWaitContainer(tmpl *wfv1.Template) (*apiv1.Container, error) {
	ctr := woc.newExecContainer(common.WaitContainerName, tmpl)
	ctr.Command = []string{"argoexec", "wait"}
	switch woc.controller.GetContainerRuntimeExecutor() {
	case common.ContainerRuntimeExecutorPNS:
		ctr.SecurityContext = &apiv1.SecurityContext{
			Capabilities: &apiv1.Capabilities{
				Add: []apiv1.Capability{
					// necessary to access main's root filesystem when run with a different user id
					apiv1.Capability("SYS_PTRACE"),
					apiv1.Capability("SYS_CHROOT"),
				},
			},
		}
		if hasPrivilegedContainers(tmpl) {
			// if the main or sidecar is privileged, the wait sidecar must also run privileged,
			// in order to SIGTERM/SIGKILL the pid
			ctr.SecurityContext.Privileged = pointer.BoolPtr(true)
		}
	case "", common.ContainerRuntimeExecutorDocker:
		ctr.VolumeMounts = append(ctr.VolumeMounts, woc.getVolumeMountDockerSock(tmpl))
	}
	return ctr, nil
}

// hasPrivilegedContainers tests if the main container or sidecars is privileged
func hasPrivilegedContainers(tmpl *wfv1.Template) bool {
	if containerIsPrivileged(tmpl.Container) {
		return true
	}
	for _, side := range tmpl.Sidecars {
		if containerIsPrivileged(&side.Container) {
			return true
		}
	}
	return false
}

func containerIsPrivileged(ctr *apiv1.Container) bool {
	if ctr != nil && ctr.SecurityContext != nil && ctr.SecurityContext.Privileged != nil && *ctr.SecurityContext.Privileged {
		return true
	}
	return false
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
			// This flag was introduced in Go 15 and will be removed in Go 16.
			// x509: cannot validate certificate for ... because it doesn't contain any IP SANs
			// https://github.com/argoproj/argo-workflows/issues/5563 - Upgrade to Go 16
			// https://github.com/golang/go/issues/39568
			Name:  "GODEBUG",
			Value: "x509ignoreCN=0",
		},
	}
	if woc.controller.Config.Executor != nil {
		execEnvVars = append(execEnvVars, woc.controller.Config.Executor.Env...)
	}
	switch woc.controller.GetContainerRuntimeExecutor() {
	case common.ContainerRuntimeExecutorK8sAPI:
		execEnvVars = append(execEnvVars,
			apiv1.EnvVar{
				Name:  common.EnvVarContainerRuntimeExecutor,
				Value: woc.controller.GetContainerRuntimeExecutor(),
			},
		)
	case common.ContainerRuntimeExecutorKubelet:
		execEnvVars = append(execEnvVars,
			apiv1.EnvVar{
				Name:  common.EnvVarContainerRuntimeExecutor,
				Value: woc.controller.GetContainerRuntimeExecutor(),
			},
			apiv1.EnvVar{
				Name: common.EnvVarDownwardAPINodeIP,
				ValueFrom: &apiv1.EnvVarSource{
					FieldRef: &apiv1.ObjectFieldSelector{
						FieldPath: "status.hostIP",
					},
				},
			},
			apiv1.EnvVar{
				Name:  common.EnvVarKubeletPort,
				Value: strconv.Itoa(woc.controller.Config.KubeletPort),
			},
			apiv1.EnvVar{
				Name:  common.EnvVarKubeletInsecure,
				Value: strconv.FormatBool(woc.controller.Config.KubeletInsecure),
			},
		)
	case common.ContainerRuntimeExecutorPNS:
		execEnvVars = append(execEnvVars,
			apiv1.EnvVar{
				Name:  common.EnvVarContainerRuntimeExecutor,
				Value: woc.controller.GetContainerRuntimeExecutor(),
			},
		)
	}
	return execEnvVars
}

func (woc *wfOperationCtx) createVolumes(tmpl *wfv1.Template) []apiv1.Volume {
	volumes := []apiv1.Volume{
		volumePodMetadata,
	}
	if woc.controller.Config.KubeConfig != nil {
		name := woc.controller.Config.KubeConfig.VolumeName
		if name == "" {
			name = common.KubeConfigDefaultVolumeName
		}
		volumes = append(volumes, apiv1.Volume{
			Name: name,
			VolumeSource: apiv1.VolumeSource{
				Secret: &apiv1.SecretVolumeSource{
					SecretName: woc.controller.Config.KubeConfig.SecretName,
				},
			},
		})
	}
	switch woc.controller.GetContainerRuntimeExecutor() {
	case common.ContainerRuntimeExecutorKubelet, common.ContainerRuntimeExecutorK8sAPI, common.ContainerRuntimeExecutorPNS:
		return volumes
	default:
		return append(volumes, woc.getVolumeDockerSock(tmpl))
	}
}

func (woc *wfOperationCtx) newExecContainer(name string, tmpl *wfv1.Template) *apiv1.Container {
	exec := apiv1.Container{
		Name:            name,
		Image:           woc.controller.executorImage(),
		ImagePullPolicy: woc.controller.executorImagePullPolicy(),
		Env:             woc.createEnvVars(),
		VolumeMounts: []apiv1.VolumeMount{
			volumeMountPodMetadata,
		},
	}
	if woc.controller.Config.Executor != nil {
		exec.Args = woc.controller.Config.Executor.Args
		if woc.controller.Config.Executor.SecurityContext != nil {
			exec.SecurityContext = woc.controller.Config.Executor.SecurityContext.DeepCopy()
		}
	}
	if isResourcesSpecified(woc.controller.Config.Executor) {
		exec.Resources = *woc.controller.Config.Executor.Resources.DeepCopy()
	} else if woc.controller.Config.ExecutorResources != nil {
		exec.Resources = *woc.controller.Config.ExecutorResources.DeepCopy()
	}
	if woc.controller.Config.KubeConfig != nil {
		path := woc.controller.Config.KubeConfig.MountPath
		if path == "" {
			path = common.KubeConfigDefaultMountPath
		}
		name := woc.controller.Config.KubeConfig.VolumeName
		if name == "" {
			name = common.KubeConfigDefaultVolumeName
		}
		exec.VolumeMounts = append(exec.VolumeMounts, apiv1.VolumeMount{
			Name:      name,
			MountPath: path,
			ReadOnly:  true,
			SubPath:   woc.controller.Config.KubeConfig.SecretKey,
		})
		exec.Args = append(exec.Args, "--kubeconfig="+path)
	}
	executorServiceAccountName := ""
	if tmpl.Executor != nil && tmpl.Executor.ServiceAccountName != "" {
		executorServiceAccountName = tmpl.Executor.ServiceAccountName
	} else if woc.execWf.Spec.Executor != nil && woc.execWf.Spec.Executor.ServiceAccountName != "" {
		executorServiceAccountName = woc.execWf.Spec.Executor.ServiceAccountName
	}
	if executorServiceAccountName != "" {
		exec.VolumeMounts = append(exec.VolumeMounts, apiv1.VolumeMount{
			Name:      common.ServiceAccountTokenVolumeName,
			MountPath: common.ServiceAccountTokenMountPath,
			ReadOnly:  true,
		})
	}
	return &exec
}

func isResourcesSpecified(ctr *apiv1.Container) bool {
	return ctr != nil && len(ctr.Resources.Limits) != 0
}

// addMetadata applies metadata specified in the template
func (woc *wfOperationCtx) addMetadata(pod *apiv1.Pod, tmpl *wfv1.Template, opts *createWorkflowPodOpts) {
	for k, v := range tmpl.Metadata.Annotations {
		pod.ObjectMeta.Annotations[k] = v
	}
	for k, v := range tmpl.Metadata.Labels {
		pod.ObjectMeta.Labels[k] = v
	}

	execCtl := common.ExecutionControl{
		IncludeScriptOutput: opts.includeScriptOutput,
	}

	if woc.workflowDeadline != nil {
		execCtl.Deadline = woc.workflowDeadline
	}

	// If we're passed down an executionDeadline, only set it if there isn't one set already, or if it's before than
	// the one already set.
	if !opts.executionDeadline.IsZero() && (execCtl.Deadline == nil || opts.executionDeadline.Before(*execCtl.Deadline)) {
		execCtl.Deadline = &opts.executionDeadline
	}

	if execCtl.Deadline != nil || opts.includeScriptOutput {
		execCtlBytes, err := json.Marshal(execCtl)
		if err != nil {
			panic(err)
		}

		pod.ObjectMeta.Annotations[common.AnnotationKeyExecutionControl] = string(execCtlBytes)
	}
}

// addSchedulingConstraints applies any node selectors or affinity rules to the pod, either set in the workflow or the template
func addSchedulingConstraints(pod *apiv1.Pod, wfSpec *wfv1.WorkflowSpec, tmpl *wfv1.Template) {
	// Set nodeSelector (if specified)
	if len(tmpl.NodeSelector) > 0 {
		pod.Spec.NodeSelector = tmpl.NodeSelector
	} else if len(wfSpec.NodeSelector) > 0 {
		pod.Spec.NodeSelector = wfSpec.NodeSelector
	}
	// Set affinity (if specified)
	if tmpl.Affinity != nil {
		pod.Spec.Affinity = tmpl.Affinity
	} else if wfSpec.Affinity != nil {
		pod.Spec.Affinity = wfSpec.Affinity
	}
	// Set tolerations (if specified)
	if len(tmpl.Tolerations) > 0 {
		pod.Spec.Tolerations = tmpl.Tolerations
	} else if len(wfSpec.Tolerations) > 0 {
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
	// Set priority (if specified)
	if tmpl.Priority != nil {
		pod.Spec.Priority = tmpl.Priority
	} else if wfSpec.PodPriority != nil {
		pod.Spec.Priority = wfSpec.PodPriority
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

// addVolumeReferences adds any volumeMounts that a container/sidecar is referencing, to the pod.spec.volumes
// These are either specified in the workflow.spec.volumes or the workflow.spec.volumeClaimTemplate section
func addVolumeReferences(pod *apiv1.Pod, vols []apiv1.Volume, tmpl *wfv1.Template, pvcs []apiv1.Volume) error {
	switch tmpl.GetType() {
	case wfv1.TemplateTypeContainer, wfv1.TemplateTypeScript:
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

	if tmpl.Container != nil {
		err := addVolumeRef(tmpl.Container.VolumeMounts)
		if err != nil {
			return err
		}
	}
	if tmpl.Script != nil {
		err := addVolumeRef(tmpl.Script.VolumeMounts)
		if err != nil {
			return err
		}
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

	volumes, volumeMounts := createSecretVolumes(tmpl)
	pod.Spec.Volumes = append(pod.Spec.Volumes, volumes...)

	for idx, container := range pod.Spec.Containers {
		if container.Name == common.WaitContainerName {
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

	return nil
}

// addInputArtifactVolumes sets up the artifacts volume to the pod to support input artifacts to containers.
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
func (woc *wfOperationCtx) addInputArtifactsVolumes(pod *apiv1.Pod, tmpl *wfv1.Template) error {
	if len(tmpl.Inputs.Artifacts) == 0 {
		return nil
	}
	artVol := apiv1.Volume{
		Name: "input-artifacts",
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{},
		},
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, artVol)

	for i, initCtr := range pod.Spec.InitContainers {
		if initCtr.Name == common.InitContainerName {
			volMount := apiv1.VolumeMount{
				Name:      artVol.Name,
				MountPath: common.ExecutorArtifactBaseDir,
			}
			initCtr.VolumeMounts = append(initCtr.VolumeMounts, volMount)

			// We also add the user supplied mount paths to the init container,
			// in case the executor needs to load artifacts to this volume
			// instead of the artifacts volume
			tmplVolumeMounts := []apiv1.VolumeMount{}
			if tmpl.Container != nil {
				tmplVolumeMounts = tmpl.Container.VolumeMounts
			} else if tmpl.Script != nil {
				tmplVolumeMounts = tmpl.Script.Container.VolumeMounts
			}
			for _, mnt := range tmplVolumeMounts {
				mnt.MountPath = filepath.Join(common.ExecutorMainFilesystemDir, mnt.MountPath)
				initCtr.VolumeMounts = append(initCtr.VolumeMounts, mnt)
			}

			pod.Spec.InitContainers[i] = initCtr
			break
		}
	}

	mainCtrIndex := -1
	for i, ctr := range pod.Spec.Containers {
		switch ctr.Name {
		case common.MainContainerName:
			mainCtrIndex = i
		}
	}
	if mainCtrIndex == -1 {
		panic("Could not find main or wait container in pod spec")
	}
	mainCtr := &pod.Spec.Containers[mainCtrIndex]

	for _, art := range tmpl.Inputs.Artifacts {
		if art.Path == "" {
			return errors.Errorf(errors.CodeBadRequest, "inputs.artifacts.%s did not specify a path", art.Name)
		}
		if !art.HasLocation() && art.Optional {
			woc.log.Infof("skip volume mount of %s (%s): optional artifact was not provided",
				art.Name, art.Path)
			continue
		}
		overlap := common.FindOverlappingVolume(tmpl, art.Path)
		if overlap != nil {
			// artifact path overlaps with a mounted volume. do not mount the
			// artifacts emptydir to the main container. init would have copied
			// the artifact to the user's volume instead
			woc.log.Debugf("skip volume mount of %s (%s): overlaps with mount %s at %s",
				art.Name, art.Path, overlap.Name, overlap.MountPath)
			continue
		}
		volMount := apiv1.VolumeMount{
			Name:      artVol.Name,
			MountPath: art.Path,
			SubPath:   art.Name,
		}
		if mainCtr.VolumeMounts == nil {
			mainCtr.VolumeMounts = make([]apiv1.VolumeMount, 0)
		}
		mainCtr.VolumeMounts = append(mainCtr.VolumeMounts, volMount)
	}
	pod.Spec.Containers[mainCtrIndex] = *mainCtr
	return nil
}

// addOutputArtifactsVolumes mirrors any volume mounts in the main container to the wait sidecar.
// For any output artifacts that were produced in mounted volumes (e.g. PVCs, emptyDirs), the
// wait container will collect the artifacts directly from volumeMount instead of `docker cp`-ing
// them to the wait sidecar. In order for this to work, we mirror all volume mounts in the main
// container under a well-known path.
func addOutputArtifactsVolumes(pod *apiv1.Pod, tmpl *wfv1.Template) {
	if tmpl.GetType() == wfv1.TemplateTypeResource {
		return
	}
	mainCtrIndex := -1
	waitCtrIndex := -1
	var mainCtr *apiv1.Container
	for i, ctr := range pod.Spec.Containers {
		switch ctr.Name {
		case common.MainContainerName:
			mainCtrIndex = i
		case common.WaitContainerName:
			waitCtrIndex = i
		}
	}
	if mainCtrIndex == -1 || waitCtrIndex == -1 {
		panic("Could not find main or wait container in pod spec")
	}
	mainCtr = &pod.Spec.Containers[mainCtrIndex]
	waitCtr := &pod.Spec.Containers[waitCtrIndex]

	for _, mnt := range mainCtr.VolumeMounts {
		mnt.MountPath = filepath.Join(common.ExecutorMainFilesystemDir, mnt.MountPath)
		// ReadOnly is needed to be false for overlapping volume mounts
		mnt.ReadOnly = false
		waitCtr.VolumeMounts = append(waitCtr.VolumeMounts, mnt)
	}
	pod.Spec.Containers[waitCtrIndex] = *waitCtr
}

// addArchiveLocation conditionally updates the template with the default artifact repository
// information configured in the controller, for the purposes of archiving outputs. This is skipped
// for templates which do not need to archive anything, or have explicitly set an archive location
// in the template.
func (woc *wfOperationCtx) addArchiveLocation(tmpl *wfv1.Template) error {
	// needLocation keeps track if the workflow needs to have an archive location set.
	// If so, and one was not supplied (or defaulted), we will return error
	var needLocation bool

	if tmpl.ArchiveLocation != nil {
		if tmpl.ArchiveLocation.S3 != nil || tmpl.ArchiveLocation.Artifactory != nil || tmpl.ArchiveLocation.HDFS != nil || tmpl.ArchiveLocation.OSS != nil || tmpl.ArchiveLocation.GCS != nil {
			// User explicitly set the location. nothing else to do.
			return nil
		}
		if tmpl.ArchiveLocation.ArchiveLogs != nil && *tmpl.ArchiveLocation.ArchiveLogs {
			needLocation = true
		}
	}
	for _, art := range tmpl.Outputs.Artifacts {
		if !art.HasLocation() {
			needLocation = true
			break
		}
	}
	if woc.artifactRepository.IsArchiveLogs() {
		needLocation = true
	}
	if !needLocation {
		woc.log.Debugf("archive location unnecessary")
		return nil
	}
	tmpl.ArchiveLocation = &wfv1.ArtifactLocation{
		ArchiveLogs: woc.artifactRepository.ArchiveLogs,
	}
	// artifact location is defaulted using the following formula:
	// <worflow_name>/<pod_name>/<artifact_name>.tgz
	// (e.g. myworkflowartifacts/argo-wf-fhljp/argo-wf-fhljp-123291312382/src.tgz)
	if s3Location := woc.artifactRepository.S3; s3Location != nil {
		woc.log.Debugf("Setting s3 artifact repository information")
		artLocationKey := s3Location.KeyFormat
		// NOTE: we use unresolved variables, will get substituted later
		if artLocationKey == "" {
			artLocationKey = path.Join(s3Location.KeyPrefix, common.DefaultArchivePattern)
		}
		tmpl.ArchiveLocation.S3 = &wfv1.S3Artifact{
			S3Bucket: s3Location.S3Bucket,
			Key:      artLocationKey,
		}
	} else if woc.artifactRepository.Artifactory != nil {
		woc.log.Debugf("Setting artifactory artifact repository information")
		repoURL := ""
		if woc.artifactRepository.Artifactory.RepoURL != "" {
			repoURL = woc.artifactRepository.Artifactory.RepoURL + "/"
		}
		artURL := fmt.Sprintf("%s%s", repoURL, common.DefaultArchivePattern)
		tmpl.ArchiveLocation.Artifactory = &wfv1.ArtifactoryArtifact{
			ArtifactoryAuth: woc.artifactRepository.Artifactory.ArtifactoryAuth,
			URL:             artURL,
		}
	} else if hdfsLocation := woc.artifactRepository.HDFS; hdfsLocation != nil {
		woc.log.Debugf("Setting HDFS artifact repository information")
		tmpl.ArchiveLocation.HDFS = &wfv1.HDFSArtifact{
			HDFSConfig: hdfsLocation.HDFSConfig,
			Path:       hdfsLocation.PathFormat,
			Force:      hdfsLocation.Force,
		}
	} else if ossLocation := woc.artifactRepository.OSS; ossLocation != nil {
		woc.log.Debugf("Setting OSS artifact repository information")
		artLocationKey := ossLocation.KeyFormat
		// NOTE: we use unresolved variables, will get substituted later
		if artLocationKey == "" {
			artLocationKey = path.Join(ossLocation.KeyFormat, common.DefaultArchivePattern)
		}
		tmpl.ArchiveLocation.OSS = &wfv1.OSSArtifact{
			OSSBucket: wfv1.OSSBucket{
				Bucket:          ossLocation.Bucket,
				Endpoint:        ossLocation.Endpoint,
				AccessKeySecret: ossLocation.AccessKeySecret,
				SecretKeySecret: ossLocation.SecretKeySecret,
			},
			Key: artLocationKey,
		}
	} else if gcsLocation := woc.artifactRepository.GCS; gcsLocation != nil {
		woc.log.Debugf("Setting GCS artifact repository information")
		artLocationKey := gcsLocation.KeyFormat
		if artLocationKey == "" {
			artLocationKey = common.DefaultArchivePattern
		}
		tmpl.ArchiveLocation.GCS = &wfv1.GCSArtifact{
			GCSBucket: wfv1.GCSBucket{
				Bucket:                  gcsLocation.Bucket,
				ServiceAccountKeySecret: gcsLocation.ServiceAccountKeySecret,
			},
			Key: artLocationKey,
		}
	} else {
		return errors.Errorf(errors.CodeBadRequest, "controller is not configured with a default archive location")
	}
	return nil
}

// setupServiceAccount sets up service account and token.
func (woc *wfOperationCtx) setupServiceAccount(pod *apiv1.Pod, tmpl *wfv1.Template) error {
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

	executorServiceAccountName := ""
	if tmpl.Executor != nil && tmpl.Executor.ServiceAccountName != "" {
		executorServiceAccountName = tmpl.Executor.ServiceAccountName
	} else if woc.execWf.Spec.Executor != nil && woc.execWf.Spec.Executor.ServiceAccountName != "" {
		executorServiceAccountName = woc.execWf.Spec.Executor.ServiceAccountName
	}
	if executorServiceAccountName != "" {
		tokenName, err := common.GetServiceAccountTokenName(woc.controller.kubeclientset, pod.Namespace, executorServiceAccountName)
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

// addScriptStagingVolume sets up a shared staging volume between the init container
// and main container for the purpose of holding the script source code for script templates
func addScriptStagingVolume(pod *apiv1.Pod) {
	volName := "argo-staging"
	stagingVol := apiv1.Volume{
		Name: volName,
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{},
		},
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, stagingVol)

	for i, initCtr := range pod.Spec.InitContainers {
		if initCtr.Name == common.InitContainerName {
			volMount := apiv1.VolumeMount{
				Name:      volName,
				MountPath: common.ExecutorStagingEmptyDir,
			}
			initCtr.VolumeMounts = append(initCtr.VolumeMounts, volMount)
			pod.Spec.InitContainers[i] = initCtr
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
func addInitContainers(pod *apiv1.Pod, tmpl *wfv1.Template) error {
	if len(tmpl.InitContainers) == 0 {
		return nil
	}
	mainCtr := findMainContainer(pod)
	if mainCtr == nil {
		panic("Unable to locate main container")
	}
	for _, ctr := range tmpl.InitContainers {
		log.Debugf("Adding init container %s", ctr.Name)
		if ctr.MirrorVolumeMounts != nil && *ctr.MirrorVolumeMounts {
			mirrorVolumeMounts(mainCtr, &ctr.Container)
		}
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, ctr.Container)
	}
	return nil
}

// addSidecars adds all sidecars to the pod spec of the step.
// Optionally volume mounts from the main container to the sidecar
func addSidecars(pod *apiv1.Pod, tmpl *wfv1.Template) error {
	if len(tmpl.Sidecars) == 0 {
		return nil
	}
	mainCtr := findMainContainer(pod)
	if mainCtr == nil {
		panic("Unable to locate main container")
	}
	for _, sidecar := range tmpl.Sidecars {
		log.Debugf("Adding sidecar container %s", sidecar.Name)
		if sidecar.MirrorVolumeMounts != nil && *sidecar.MirrorVolumeMounts {
			mirrorVolumeMounts(mainCtr, &sidecar.Container)
		}
		pod.Spec.Containers = append(pod.Spec.Containers, sidecar.Container)
	}
	return nil
}

// verifyResolvedVariables is a helper to ensure all {{variables}} have been resolved for a object
func verifyResolvedVariables(obj interface{}) error {
	str, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	var unresolvedErr error
	fstTmpl, err := fasttemplate.NewTemplate(string(str), "{{", "}}")
	if err != nil {
		return fmt.Errorf("unable to parse argo varaible: %w", err)
	}
	fstTmpl.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
		unresolvedErr = errors.Errorf(errors.CodeBadRequest, "failed to resolve {{%s}}", tag)
		return 0, nil
	})
	return unresolvedErr
}

// createSecretVolumes will retrieve and create Volumes and Volumemount object for Pod
func createSecretVolumes(tmpl *wfv1.Template) ([]apiv1.Volume, []apiv1.VolumeMount) {
	var allVolumesMap = make(map[string]apiv1.Volume)
	var uniqueKeyMap = make(map[string]bool)
	var secretVolumes []apiv1.Volume
	var secretVolMounts []apiv1.VolumeMount

	createArchiveLocationSecret(tmpl, allVolumesMap, uniqueKeyMap)

	for _, art := range tmpl.Outputs.Artifacts {
		createSecretVolume(allVolumesMap, art, uniqueKeyMap)
	}
	for _, art := range tmpl.Inputs.Artifacts {
		createSecretVolume(allVolumesMap, art, uniqueKeyMap)
	}

	for volMountName, val := range allVolumesMap {
		secretVolumes = append(secretVolumes, val)
		secretVolMounts = append(secretVolMounts, apiv1.VolumeMount{
			Name:      volMountName,
			MountPath: common.SecretVolMountPath + "/" + val.Name,
			ReadOnly:  true,
		})
	}

	return secretVolumes, secretVolMounts
}

func createArchiveLocationSecret(tmpl *wfv1.Template, volMap map[string]apiv1.Volume, uniqueKeyMap map[string]bool) {
	if tmpl.ArchiveLocation == nil {
		return
	}
	if s3ArtRepo := tmpl.ArchiveLocation.S3; s3ArtRepo != nil {
		createSecretVal(volMap, &s3ArtRepo.AccessKeySecret, uniqueKeyMap)
		createSecretVal(volMap, &s3ArtRepo.SecretKeySecret, uniqueKeyMap)
	} else if hdfsArtRepo := tmpl.ArchiveLocation.HDFS; hdfsArtRepo != nil {
		createSecretVal(volMap, hdfsArtRepo.KrbKeytabSecret, uniqueKeyMap)
		createSecretVal(volMap, hdfsArtRepo.KrbCCacheSecret, uniqueKeyMap)
	} else if artRepo := tmpl.ArchiveLocation.Artifactory; artRepo != nil {
		createSecretVal(volMap, artRepo.UsernameSecret, uniqueKeyMap)
		createSecretVal(volMap, artRepo.PasswordSecret, uniqueKeyMap)
	} else if gitRepo := tmpl.ArchiveLocation.Git; gitRepo != nil {
		createSecretVal(volMap, gitRepo.UsernameSecret, uniqueKeyMap)
		createSecretVal(volMap, gitRepo.PasswordSecret, uniqueKeyMap)
		createSecretVal(volMap, gitRepo.SSHPrivateKeySecret, uniqueKeyMap)
	} else if ossRepo := tmpl.ArchiveLocation.OSS; ossRepo != nil {
		createSecretVal(volMap, &ossRepo.AccessKeySecret, uniqueKeyMap)
		createSecretVal(volMap, &ossRepo.SecretKeySecret, uniqueKeyMap)
	} else if gcsRepo := tmpl.ArchiveLocation.GCS; gcsRepo != nil {
		createSecretVal(volMap, &gcsRepo.ServiceAccountKeySecret, uniqueKeyMap)
	}
}

func createSecretVolume(volMap map[string]apiv1.Volume, art wfv1.Artifact, keyMap map[string]bool) {
	if art.S3 != nil {
		createSecretVal(volMap, &art.S3.AccessKeySecret, keyMap)
		createSecretVal(volMap, &art.S3.SecretKeySecret, keyMap)
	} else if art.Git != nil {
		createSecretVal(volMap, art.Git.UsernameSecret, keyMap)
		createSecretVal(volMap, art.Git.PasswordSecret, keyMap)
		createSecretVal(volMap, art.Git.SSHPrivateKeySecret, keyMap)
	} else if art.Artifactory != nil {
		createSecretVal(volMap, art.Artifactory.UsernameSecret, keyMap)
		createSecretVal(volMap, art.Artifactory.PasswordSecret, keyMap)
	} else if art.HDFS != nil {
		createSecretVal(volMap, art.HDFS.KrbCCacheSecret, keyMap)
		createSecretVal(volMap, art.HDFS.KrbKeytabSecret, keyMap)
	} else if art.OSS != nil {
		createSecretVal(volMap, &art.OSS.AccessKeySecret, keyMap)
		createSecretVal(volMap, &art.OSS.SecretKeySecret, keyMap)
	} else if art.GCS != nil {
		createSecretVal(volMap, &art.GCS.ServiceAccountKeySecret, keyMap)
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

// findMainContainer finds main container
func findMainContainer(pod *apiv1.Pod) *apiv1.Container {
	var mainCtr *apiv1.Container
	for _, ctr := range pod.Spec.Containers {
		if ctr.Name != common.MainContainerName {
			continue
		}
		mainCtr = &ctr
		break
	}
	return mainCtr
}

// mirrorVolumeMounts mirrors volumeMounts of source container to target container
func mirrorVolumeMounts(sourceContainer, targetContainer *apiv1.Container) {
	for _, volMnt := range sourceContainer.VolumeMounts {
		if targetContainer.VolumeMounts == nil {
			targetContainer.VolumeMounts = make([]apiv1.VolumeMount, 0)
		}
		log.Debugf("Adding volume mount %v to container %v", volMnt.Name, targetContainer.Name)
		targetContainer.VolumeMounts = append(targetContainer.VolumeMounts, volMnt)

	}
}
