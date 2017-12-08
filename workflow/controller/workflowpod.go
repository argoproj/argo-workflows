package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"path"

	wfv1 "github.com/argoproj/argo/api/workflow/v1alpha1"
	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/workflow/common"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasttemplate"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
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
					apiv1.DownwardAPIVolumeFile{
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

	hostPathDir = apiv1.HostPathDirectory

	// volumeDockerLib provides the wait container access to the minion's host docker containers
	// runtime files (e.g. /var/lib/docker/container). This is used by the executor to access
	// the main container's logs (and potentially storage to upload output artifacts)
	volumeDockerLib = apiv1.Volume{
		Name: common.DockerLibVolumeName,
		VolumeSource: apiv1.VolumeSource{
			HostPath: &apiv1.HostPathVolumeSource{
				Path: common.DockerLibHostPath,
				Type: &hostPathDir,
			},
		},
	}
	volumeMountDockerLib = apiv1.VolumeMount{
		Name:      volumeDockerLib.Name,
		MountPath: volumeDockerLib.VolumeSource.HostPath.Path,
		ReadOnly:  true,
	}

	// volumeDockerSock provides the wait container direct access to the minion's host docker daemon.
	// The primary purpose of this is to make available `docker cp` to collect an output artifact
	// from a container. Alternatively, we could use `kubectl cp`, but `docker cp` avoids the extra
	// hop to the kube api server.
	volumeDockerSock = apiv1.Volume{
		Name: common.DockerSockVolumeName,
		VolumeSource: apiv1.VolumeSource{
			HostPath: &apiv1.HostPathVolumeSource{
				Path: "/var/run",
				Type: &hostPathDir,
			},
		},
	}
	volumeMountDockerSock = apiv1.VolumeMount{
		Name:      volumeDockerSock.Name,
		MountPath: "/var/run/docker.sock",
		ReadOnly:  true,
		SubPath:   "docker.sock",
	}

	// execEnvVars exposes various pod information as environment variables to the exec container
	execEnvVars = []apiv1.EnvVar{
		envFromField(common.EnvVarHostIP, "status.hostIP"),
		envFromField(common.EnvVarPodIP, "status.podIP"),
		envFromField(common.EnvVarPodName, "metadata.name"),
		envFromField(common.EnvVarNamespace, "metadata.namespace"),
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

func (woc *wfOperationCtx) createWorkflowPod(nodeName string, tmpl *wfv1.Template) error {
	woc.log.Debugf("Creating Pod: %s", nodeName)
	tmpl = tmpl.DeepCopy()
	waitCtr, err := woc.newWaitContainer(tmpl)
	if err != nil {
		return err
	}
	var mainCtr apiv1.Container
	if tmpl.Container != nil {
		mainCtr = *tmpl.Container
	} else if tmpl.Script != nil {
		// script case
		mainCtr = apiv1.Container{
			Image:   tmpl.Script.Image,
			Command: tmpl.Script.Command,
			Args:    []string{common.ScriptTemplateSourcePath},
		}
	} else {
		return errors.InternalError("Cannot create container from non-container/script template")
	}
	mainCtr.Name = common.MainContainerName
	t := true

	pod := apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: woc.wf.NodeID(nodeName),
			Labels: map[string]string{
				common.LabelKeyWorkflow:  woc.wf.ObjectMeta.Name, // Allows filtering by pods related to specific workflow
				common.LabelKeyCompleted: "false",                // Allows filtering by incomplete workflow pods
			},
			Annotations: map[string]string{
				common.AnnotationKeyNodeName: nodeName,
			},
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion:         wfv1.CRDFullName,
					Kind:               wfv1.CRDKind,
					Name:               woc.wf.ObjectMeta.Name,
					UID:                woc.wf.ObjectMeta.UID,
					BlockOwnerDeletion: &t,
				},
			},
		},
		Spec: apiv1.PodSpec{
			RestartPolicy: apiv1.RestartPolicyNever,
			Containers: []apiv1.Container{
				*waitCtr,
				mainCtr,
			},
			Volumes: []apiv1.Volume{
				volumePodMetadata,
				volumeDockerLib,
				volumeDockerSock,
			},
		},
	}

	// Add init container only if it needs input artifacts
	// or if it is a script template (which needs to populate the script)
	if len(tmpl.Inputs.Artifacts) > 0 || tmpl.Script != nil {
		initCtr := woc.newInitContainer(tmpl)
		pod.Spec.InitContainers = []apiv1.Container{initCtr}
	}

	woc.addNodeSelectors(&pod, tmpl)

	err = woc.addVolumeReferences(&pod, tmpl)
	if err != nil {
		return err
	}

	err = woc.addInputArtifactsVolumes(&pod, tmpl)
	if err != nil {
		return err
	}
	err = woc.addArchiveLocation(&pod, tmpl)
	if err != nil {
		return err
	}

	if tmpl.Script != nil {
		addScriptVolume(&pod)
	}

	// addSidecars should be called after all volumes have been manipulated
	// in the main container (in case sidecar requires volume mount mirroring)
	err = addSidecars(&pod, tmpl)
	if err != nil {
		return err
	}

	// Set the container template JSON in pod annotations, which executor
	// will examine for things like artifact location/path. Also ensures
	// that all variables have been resolved. Do this last, after all
	// template manipulations have been performed.
	tmplBytes, err := json.Marshal(tmpl)
	if err != nil {
		return err
	}
	err = verifyResolvedVariables(string(tmplBytes))
	if err != nil {
		return err
	}
	pod.ObjectMeta.Annotations[common.AnnotationKeyTemplate] = string(tmplBytes)

	created, err := woc.controller.clientset.CoreV1().Pods(woc.wf.ObjectMeta.Namespace).Create(&pod)
	if err != nil {
		if apierr.IsAlreadyExists(err) {
			// workflow pod names are deterministic. We can get here if
			// the controller fails to persist the workflow after creating the pod.
			woc.log.Infof("Skipped pod %s creation: already exists", nodeName)
			return nil
		}
		woc.log.Infof("Failed to create pod %s: %v", nodeName, err)
		return errors.InternalWrapError(err)
	}
	woc.log.Infof("Created pod: %s", created.Name)
	return nil
}

func (woc *wfOperationCtx) newInitContainer(tmpl *wfv1.Template) apiv1.Container {
	ctr := woc.newExecContainer(common.InitContainerName, false)
	ctr.Command = []string{"argoexec"}
	argoExecCmd := fmt.Sprintf("init")
	ctr.Args = []string{argoExecCmd}
	ctr.VolumeMounts = []apiv1.VolumeMount{
		volumeMountPodMetadata,
	}
	return *ctr
}

func (woc *wfOperationCtx) newWaitContainer(tmpl *wfv1.Template) (*apiv1.Container, error) {
	ctr := woc.newExecContainer(common.WaitContainerName, false)
	ctr.Command = []string{"argoexec"}
	argoExecCmd := fmt.Sprintf("wait")
	ctr.Args = []string{argoExecCmd}
	ctr.VolumeMounts = []apiv1.VolumeMount{
		volumeMountPodMetadata,
		volumeMountDockerLib,
		volumeMountDockerSock,
	}
	return ctr, nil
}

func (woc *wfOperationCtx) newExecContainer(name string, privileged bool) *apiv1.Container {
	exec := apiv1.Container{
		Name:  name,
		Image: woc.controller.Config.ExecutorImage,
		Env:   execEnvVars,
		Resources: apiv1.ResourceRequirements{
			Limits: apiv1.ResourceList{
				apiv1.ResourceCPU:    resource.MustParse("0.5"),
				apiv1.ResourceMemory: resource.MustParse("512Mi"),
			},
			Requests: apiv1.ResourceList{
				apiv1.ResourceCPU:    resource.MustParse("0.1"),
				apiv1.ResourceMemory: resource.MustParse("64Mi"),
			},
		},
		SecurityContext: &apiv1.SecurityContext{
			Privileged: &privileged,
		},
	}
	return &exec
}

// addNodeSelectors applies any node selectors, either set in the workflow or the template, to the pod
func (woc *wfOperationCtx) addNodeSelectors(pod *apiv1.Pod, tmpl *wfv1.Template) {
	if len(tmpl.NodeSelector) > 0 {
		pod.Spec.NodeSelector = tmpl.NodeSelector
		return
	}
	if len(woc.wf.Spec.NodeSelector) > 0 {
		pod.Spec.NodeSelector = woc.wf.Spec.NodeSelector
		return
	}
}

// addVolumeReferences adds any volumeMounts that a container is referencing, to the pod.spec.volumes
// These are either specified in the workflow.spec.volumes or the workflow.spec.volumeClaimTemplate section
func (woc *wfOperationCtx) addVolumeReferences(pod *apiv1.Pod, tmpl *wfv1.Template) error {
	if tmpl.Container == nil {
		return nil
	}
	for _, volMnt := range tmpl.Container.VolumeMounts {
		vol := getVolByName(volMnt.Name, woc.wf)
		if vol == nil {
			return errors.Errorf(errors.CodeBadRequest, "volume '%s' not found in workflow spec", volMnt.Name)
		}
		if len(pod.Spec.Volumes) == 0 {
			pod.Spec.Volumes = make([]apiv1.Volume, 0)
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, *vol)
	}
	return nil
}

// getVolByName is a helper to retreive a volume by its name, either from the volumes or claims section
func getVolByName(name string, wf *wfv1.Workflow) *apiv1.Volume {
	for _, vol := range wf.Spec.Volumes {
		if vol.Name == name {
			return &vol
		}
	}
	for _, pvc := range wf.Status.PersistentVolumeClaims {
		if pvc.Name == name {
			return &pvc
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
			for _, mnt := range tmpl.Container.VolumeMounts {
				mnt.MountPath = path.Join(common.InitContainerMainFilesystemDir, mnt.MountPath)
				initCtr.VolumeMounts = append(initCtr.VolumeMounts, mnt)
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
			mainCtr = &ctr
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

// addArchiveLocation updates the template with artifact repository information configured in the controller.
// This is skipped for templates which have explicitly set an archive location in the template
func (woc *wfOperationCtx) addArchiveLocation(pod *apiv1.Pod, tmpl *wfv1.Template) error {
	if tmpl.ArchiveLocation != nil {
		return nil
	}
	tmpl.ArchiveLocation = &wfv1.ArtifactLocation{}
	// artifacts are stored in using the following formula:
	// <repo_key_prefix>/<worflow_name>/<node_id>/<artifact_name>.tgz
	// (e.g. myworkflowartifacts/argo-wf-fhljp/argo-wf-fhljp-123291312382/src.tgz)
	// TODO: will need to support more advanced organization of artifacts such as dated
	// (e.g. myworkflowartifacts/2017/10/31/... )
	if woc.controller.Config.ArtifactRepository.S3 != nil {
		log.Debugf("Setting s3 artifact repository information")
		keyPrefix := ""
		if woc.controller.Config.ArtifactRepository.S3.KeyPrefix != "" {
			keyPrefix = woc.controller.Config.ArtifactRepository.S3.KeyPrefix + "/"
		}
		artLocationKey := fmt.Sprintf("%s%s/%s", keyPrefix, woc.wf.ObjectMeta.Name, pod.ObjectMeta.Name)
		tmpl.ArchiveLocation.S3 = &wfv1.S3Artifact{
			S3Bucket: woc.controller.Config.ArtifactRepository.S3.S3Bucket,
			Key:      artLocationKey,
		}
	} else {
		for _, art := range tmpl.Outputs.Artifacts {
			if !art.HasLocation() {
				return errors.Errorf(errors.CodeBadRequest, "controller is not configured with a default archive location")
			}
		}
	}
	return nil
}

// addScriptVolume sets up the shared volume between init container and main container
// containing the template script source code
func addScriptVolume(pod *apiv1.Pod) {
	volName := "script"
	scriptVol := apiv1.Volume{
		Name: volName,
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{},
		},
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, scriptVol)

	for i, initCtr := range pod.Spec.InitContainers {
		if initCtr.Name == common.InitContainerName {
			volMount := apiv1.VolumeMount{
				Name:      volName,
				MountPath: common.ScriptTemplateEmptyDir,
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
				MountPath: common.ScriptTemplateEmptyDir,
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

// addSidecars adds all sidecars to the pod spec of the step.
// Optionally volume mounts from the main container to the sidecar
func addSidecars(pod *apiv1.Pod, tmpl *wfv1.Template) error {
	if len(tmpl.Sidecars) == 0 {
		return nil
	}
	var mainCtr *apiv1.Container
	for _, ctr := range pod.Spec.Containers {
		if ctr.Name != common.MainContainerName {
			continue
		}
		mainCtr = &ctr
		break
	}
	if mainCtr == nil {
		panic("Unable to locate main container")
	}
	for _, sidecar := range tmpl.Sidecars {
		if sidecar.MirrorVolumeMounts != nil && *sidecar.MirrorVolumeMounts {
			for _, volMnt := range mainCtr.VolumeMounts {
				if sidecar.VolumeMounts == nil {
					sidecar.VolumeMounts = make([]apiv1.VolumeMount, 0)
				}
				sidecar.VolumeMounts = append(sidecar.VolumeMounts, volMnt)
			}
		}
		pod.Spec.Containers = append(pod.Spec.Containers, sidecar.Container)
	}
	return nil
}

// verifyResolvedVariables is a helper to ensure all {{variables}} have been resolved
func verifyResolvedVariables(tmplStr string) error {
	var unresolvedErr error
	fstTmpl := fasttemplate.New(tmplStr, "{{", "}}")
	fstTmpl.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
		unresolvedErr = errors.Errorf(errors.CodeBadRequest, "failed to resolve {{%s}} tmplStr: %s", tag, tmplStr)
		return 0, nil
	})
	return unresolvedErr
}
