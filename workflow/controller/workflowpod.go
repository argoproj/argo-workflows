package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strconv"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasttemplate"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// volumeDockerSock provides the wait container direct access to the minion's host docker daemon.
	// The primary purpose of this is to make available `docker cp` to collect an output artifact
	// from a container. Alternatively, we could use `kubectl cp`, but `docker cp` avoids the extra
	// hop to the kube api server.
	volumeDockerSock = apiv1.Volume{
		Name: common.DockerSockVolumeName,
		VolumeSource: apiv1.VolumeSource{
			HostPath: &apiv1.HostPathVolumeSource{
				Path: "/var/run/docker.sock",
				Type: &hostPathSocket,
			},
		},
	}
	volumeMountDockerSock = apiv1.VolumeMount{
		Name:      volumeDockerSock.Name,
		MountPath: "/var/run/docker.sock",
		ReadOnly:  true,
	}

	// execEnvVars exposes various pod information as environment variables to the exec container
	execEnvVars = []apiv1.EnvVar{
		envFromField(common.EnvVarPodName, "metadata.name"),
	}
)

// envFromField is a helper to return a EnvVar with the name and field
func envFromField(envVarName, fieldPath string) apiv1.EnvVar {
	return apiv1.EnvVar{
		Name: envVarName,
		ValueFrom: &apiv1.EnvVarSource{
			FieldRef: &apiv1.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath:  fieldPath,
			},
		},
	}
}

func (woc *wfOperationCtx) createWorkflowPod(nodeName string, mainCtr apiv1.Container, tmpl *wfv1.Template) (*apiv1.Pod, error) {
	nodeID := woc.wf.NodeID(nodeName)
	woc.log.Debugf("Creating Pod: %s (%s)", nodeName, nodeID)
	tmpl = tmpl.DeepCopy()
	wfSpec := woc.wf.Spec.DeepCopy()
	mainCtr.Name = common.MainContainerName
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
				*metav1.NewControllerRef(woc.wf, wfv1.SchemaGroupVersionKind),
			},
		},
		Spec: apiv1.PodSpec{
			RestartPolicy: apiv1.RestartPolicyNever,
			Containers: []apiv1.Container{
				mainCtr,
			},
			Volumes:               woc.createVolumes(),
			ActiveDeadlineSeconds: tmpl.ActiveDeadlineSeconds,
			ServiceAccountName:    woc.wf.Spec.ServiceAccountName,
			ImagePullSecrets:      woc.wf.Spec.ImagePullSecrets,
		},
	}

	if woc.wf.Spec.HostNetwork != nil {
		pod.Spec.HostNetwork = *woc.wf.Spec.HostNetwork
	}

	if woc.wf.Spec.DNSPolicy != nil {
		pod.Spec.DNSPolicy = *woc.wf.Spec.DNSPolicy
	}

	if woc.wf.Spec.DNSConfig != nil {
		pod.Spec.DNSConfig = woc.wf.Spec.DNSConfig
	}

	if woc.controller.Config.InstanceID != "" {
		pod.ObjectMeta.Labels[common.LabelKeyControllerInstanceID] = woc.controller.Config.InstanceID
	}

	err := woc.addArchiveLocation(pod, tmpl)
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

	// Add init container only if it needs input artifacts. This is also true for
	// script templates (which needs to populate the script)
	if len(tmpl.Inputs.Artifacts) > 0 || tmpl.GetType() == wfv1.TemplateTypeScript {
		initCtr := woc.newInitContainer(tmpl)
		pod.Spec.InitContainers = []apiv1.Container{initCtr}
	}

	addSchedulingConstraints(pod, wfSpec, tmpl)
	woc.addMetadata(pod, tmpl)

	err = addVolumeReferences(pod, wfSpec, tmpl, woc.wf.Status.PersistentVolumeClaims)
	if err != nil {
		return nil, err
	}

	err = woc.addInputArtifactsVolumes(pod, tmpl)
	if err != nil {
		return nil, err
	}

	if tmpl.GetType() == wfv1.TemplateTypeScript {
		addExecutorStagingVolume(pod)
	}

	// addInitContainers should be called after all volumes have been manipulated
	// in the main container (in case sidecar requires volume mount mirroring)
	err = addInitContainers(pod, tmpl)
	if err != nil {
		return nil, err
	}

	// addSidecars should be called after all volumes have been manipulated
	// in the main container (in case sidecar requires volume mount mirroring)
	err = addSidecars(pod, tmpl)
	if err != nil {
		return nil, err
	}

	// Set the container template JSON in pod annotations, which executor examines for things like
	// artifact location/path.
	tmplBytes, err := json.Marshal(tmpl)
	if err != nil {
		return nil, err
	}
	pod.ObjectMeta.Annotations[common.AnnotationKeyTemplate] = string(tmplBytes)

	// Perform one last variable substitution here. Some variables come from the from workflow
	// configmap (e.g. archive location), and were not substituted in executeTemplate.
	pod, err = substituteGlobals(pod, woc.globalParams)
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

	created, err := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.ObjectMeta.Namespace).Create(pod)
	if err != nil {
		if apierr.IsAlreadyExists(err) {
			// workflow pod names are deterministic. We can get here if the
			// controller fails to persist the workflow after creating the pod.
			woc.log.Infof("Skipped pod %s (%s) creation: already exists", nodeName, nodeID)
			return created, nil
		}
		woc.log.Infof("Failed to create pod %s (%s): %v", nodeName, nodeID, err)
		return nil, errors.InternalWrapError(err)
	}
	woc.log.Infof("Created pod: %s (%s)", nodeName, created.Name)
	woc.activePods++
	return created, nil
}

// substituteGlobals returns a pod spec with global parameter references substituted as well as pod.name
func substituteGlobals(pod *apiv1.Pod, globalParams map[string]string) (*apiv1.Pod, error) {
	newGlobalParams := make(map[string]string)
	for k, v := range globalParams {
		newGlobalParams[k] = v
	}
	newGlobalParams[common.LocalVarPodName] = pod.Name
	globalParams = newGlobalParams
	specBytes, err := json.Marshal(pod)
	if err != nil {
		return nil, err
	}
	fstTmpl := fasttemplate.New(string(specBytes), "{{", "}}")
	newSpecBytes, err := common.Replace(fstTmpl, globalParams, true)
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
	ctr := woc.newExecContainer(common.InitContainerName, false, "init")
	ctr.VolumeMounts = append([]apiv1.VolumeMount{volumeMountPodMetadata}, ctr.VolumeMounts...)
	return *ctr
}

func (woc *wfOperationCtx) newWaitContainer(tmpl *wfv1.Template) (*apiv1.Container, error) {
	ctr := woc.newExecContainer(common.WaitContainerName, false, "wait")
	ctr.VolumeMounts = append(woc.createVolumeMounts(), ctr.VolumeMounts...)
	return ctr, nil
}

func (woc *wfOperationCtx) createEnvVars() []apiv1.EnvVar {
	switch woc.controller.Config.ContainerRuntimeExecutor {
	case common.ContainerRuntimeExecutorK8sAPI:
		return append(execEnvVars,
			apiv1.EnvVar{
				Name:  common.EnvVarContainerRuntimeExecutor,
				Value: woc.controller.Config.ContainerRuntimeExecutor,
			},
		)
	case common.ContainerRuntimeExecutorKubelet:
		return append(execEnvVars,
			apiv1.EnvVar{
				Name:  common.EnvVarContainerRuntimeExecutor,
				Value: woc.controller.Config.ContainerRuntimeExecutor,
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
	default:
		return execEnvVars
	}
}

func (woc *wfOperationCtx) createVolumeMounts() []apiv1.VolumeMount {
	volumeMounts := []apiv1.VolumeMount{
		volumeMountPodMetadata,
	}
	switch woc.controller.Config.ContainerRuntimeExecutor {
	case common.ContainerRuntimeExecutorKubelet, common.ContainerRuntimeExecutorK8sAPI:
		return volumeMounts
	default:
		return append(volumeMounts, volumeMountDockerSock)
	}
}

func (woc *wfOperationCtx) createVolumes() []apiv1.Volume {
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
	switch woc.controller.Config.ContainerRuntimeExecutor {
	case common.ContainerRuntimeExecutorKubelet, common.ContainerRuntimeExecutorK8sAPI:
		return volumes
	default:
		return append(volumes, volumeDockerSock)
	}
}

func (woc *wfOperationCtx) newExecContainer(name string, privileged bool, subCommand string) *apiv1.Container {
	exec := apiv1.Container{
		Name:            name,
		Image:           woc.controller.executorImage(),
		ImagePullPolicy: woc.controller.executorImagePullPolicy(),
		Env:             woc.createEnvVars(),
		SecurityContext: &apiv1.SecurityContext{
			Privileged: &privileged,
		},
		Command: []string{"argoexec"},
		Args:    []string{subCommand},
	}
	if woc.controller.Config.ExecutorResources != nil {
		exec.Resources = *woc.controller.Config.ExecutorResources
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
		exec.VolumeMounts = []apiv1.VolumeMount{{
			Name:      name,
			MountPath: path,
			ReadOnly:  true,
			SubPath:   woc.controller.Config.KubeConfig.SecretKey,
		},
		}
		exec.Args = append(exec.Args, "--kubeconfig="+path)
	}
	return &exec
}

// addMetadata applies metadata specified in the template
func (woc *wfOperationCtx) addMetadata(pod *apiv1.Pod, tmpl *wfv1.Template) {
	for k, v := range tmpl.Metadata.Annotations {
		pod.ObjectMeta.Annotations[k] = v
	}
	for k, v := range tmpl.Metadata.Labels {
		pod.ObjectMeta.Labels[k] = v
	}
	if woc.workflowDeadline != nil {
		execCtl := common.ExecutionControl{
			Deadline: woc.workflowDeadline,
		}
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
	// Set schedulerName (if specified)
	if tmpl.SchedulerName != "" {
		pod.Spec.SchedulerName = tmpl.SchedulerName
	} else if wfSpec.SchedulerName != "" {
		pod.Spec.SchedulerName = wfSpec.SchedulerName
	}
}

// addVolumeReferences adds any volumeMounts that a container/sidecar is referencing, to the pod.spec.volumes
// These are either specified in the workflow.spec.volumes or the workflow.spec.volumeClaimTemplate section
func addVolumeReferences(pod *apiv1.Pod, wfSpec *wfv1.WorkflowSpec, tmpl *wfv1.Template, pvcs []apiv1.Volume) error {
	switch tmpl.GetType() {
	case wfv1.TemplateTypeContainer, wfv1.TemplateTypeScript:
	default:
		return nil
	}

	// getVolByName is a helper to retrieve a volume by its name, either from the volumes or claims section
	getVolByName := func(name string) *apiv1.Volume {
		for _, vol := range wfSpec.Volumes {
			if vol.Name == name {
				return &vol
			}
		}
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
			if tmpl.Container != nil {
				for _, mnt := range tmpl.Container.VolumeMounts {
					mnt.MountPath = path.Join(common.InitContainerMainFilesystemDir, mnt.MountPath)
					initCtr.VolumeMounts = append(initCtr.VolumeMounts, mnt)
				}
			}
			pod.Spec.InitContainers[i] = initCtr
			break
		}
	}

	mainCtrIndex := 0
	var mainCtr *apiv1.Container
	for i, ctr := range pod.Spec.Containers {
		if ctr.Name == common.MainContainerName {
			mainCtrIndex = i
			mainCtr = &pod.Spec.Containers[i]
		}
	}
	if mainCtr == nil {
		panic("Could not find main container in pod spec")
	}
	// TODO: the order in which we construct the volume mounts may matter,
	// especially if they are overlapping.
	for _, art := range tmpl.Inputs.Artifacts {
		if art.Path == "" {
			return errors.Errorf(errors.CodeBadRequest, "inputs.artifacts.%s did not specify a path", art.Name)
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

// addArchiveLocation updates the template with the default artifact repository information
// configured in the controller. This is skipped for templates which have explicitly set an archive
// location in the template.
func (woc *wfOperationCtx) addArchiveLocation(pod *apiv1.Pod, tmpl *wfv1.Template) error {
	if tmpl.ArchiveLocation == nil {
		tmpl.ArchiveLocation = &wfv1.ArtifactLocation{
			ArchiveLogs: woc.controller.Config.ArtifactRepository.ArchiveLogs,
		}
	}
	if tmpl.ArchiveLocation.S3 != nil || tmpl.ArchiveLocation.Artifactory != nil || tmpl.ArchiveLocation.HDFS != nil {
		// User explicitly set the location. nothing else to do.
		return nil
	}
	// needLocation keeps track if the workflow needs to have an archive location set.
	// If so, and one was not supplied (or defaulted), we will return error
	var needLocation bool
	if tmpl.ArchiveLocation.ArchiveLogs != nil && *tmpl.ArchiveLocation.ArchiveLogs {
		needLocation = true
	}

	// artifact location is defaulted using the following formula:
	// <worflow_name>/<pod_name>/<artifact_name>.tgz
	// (e.g. myworkflowartifacts/argo-wf-fhljp/argo-wf-fhljp-123291312382/src.tgz)
	if s3Location := woc.controller.Config.ArtifactRepository.S3; s3Location != nil {
		log.Debugf("Setting s3 artifact repository information")
		artLocationKey := s3Location.KeyFormat
		// NOTE: we use unresolved variables, will get substituted later
		if artLocationKey == "" {
			artLocationKey = path.Join(s3Location.KeyPrefix, common.DefaultArchivePattern)
		}
		tmpl.ArchiveLocation.S3 = &wfv1.S3Artifact{
			S3Bucket: s3Location.S3Bucket,
			Key:      artLocationKey,
		}
	} else if woc.controller.Config.ArtifactRepository.Artifactory != nil {
		log.Debugf("Setting artifactory artifact repository information")
		repoURL := ""
		if woc.controller.Config.ArtifactRepository.Artifactory.RepoURL != "" {
			repoURL = woc.controller.Config.ArtifactRepository.Artifactory.RepoURL + "/"
		}
		artURL := fmt.Sprintf("%s%s", repoURL, common.DefaultArchivePattern)
		tmpl.ArchiveLocation.Artifactory = &wfv1.ArtifactoryArtifact{
			ArtifactoryAuth: woc.controller.Config.ArtifactRepository.Artifactory.ArtifactoryAuth,
			URL:             artURL,
		}
	} else if hdfsLocation := woc.controller.Config.ArtifactRepository.HDFS; hdfsLocation != nil {
		log.Debugf("Setting HDFS artifact repository information")
		tmpl.ArchiveLocation.HDFS = &wfv1.HDFSArtifact{
			HDFSConfig: hdfsLocation.HDFSConfig,
			Path:       hdfsLocation.PathFormat,
			Force:      hdfsLocation.Force,
		}
	} else {
		for _, art := range tmpl.Outputs.Artifacts {
			if !art.HasLocation() {
				needLocation = true
				break
			}
		}
		if needLocation {
			return errors.Errorf(errors.CodeBadRequest, "controller is not configured with a default archive location")
		}
	}
	return nil
}

// addExecutorStagingVolume sets up a shared staging volume between the init container
// and main container for the purpose of holding the script source code for script templates
func addExecutorStagingVolume(pod *apiv1.Pod) {
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
			if ctr.VolumeMounts == nil {
				ctr.VolumeMounts = []apiv1.VolumeMount{volMount}
			} else {
				ctr.VolumeMounts = append(ctr.VolumeMounts, volMount)
			}
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
	fstTmpl := fasttemplate.New(string(str), "{{", "}}")
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

	createArgoArtifactsRepoSecret(tmpl, allVolumesMap, uniqueKeyMap)

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
			MountPath: common.SecretVolMountPath,
			ReadOnly:  true,
		})
	}

	return secretVolumes, secretVolMounts
}

func createArgoArtifactsRepoSecret(tmpl *wfv1.Template, volMap map[string]apiv1.Volume, uniqueKeyMap map[string]bool) {
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
	}
}

func createSecretVal(volMap map[string]apiv1.Volume, secret *apiv1.SecretKeySelector, keyMap map[string]bool) {
	if secret == nil {
		return
	}
	if vol, ok := volMap[secret.Name]; ok {
		key := apiv1.KeyToPath{
			Key:  secret.Key,
			Path: secret.Name + "/" + secret.Key,
		}
		if val, _ := keyMap[secret.Name+"-"+secret.Key]; !val {
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
							Path: secret.Name + "/" + secret.Key,
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
