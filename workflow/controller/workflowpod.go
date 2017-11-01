package controller

import (
	"encoding/json"
	"fmt"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/workflow/common"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Reusable k8s pod spec portions used in workflow pods
var (
	// volumePodMetadata makes available the pod metadata available as a file
	// to the argoexec init and sidekick containers. Specifically, the template
	// of the pod is stored as an annotation
	volumePodMetadata = corev1.Volume{
		Name: common.PodMetadataVolumeName,
		VolumeSource: corev1.VolumeSource{
			DownwardAPI: &corev1.DownwardAPIVolumeSource{
				Items: []corev1.DownwardAPIVolumeFile{
					corev1.DownwardAPIVolumeFile{
						Path: common.PodMetadataAnnotationsVolumePath,
						FieldRef: &corev1.ObjectFieldSelector{
							APIVersion: "v1",
							FieldPath:  "metadata.annotations",
						},
					},
				},
			},
		},
	}
	volumeMountPodMetadata = corev1.VolumeMount{
		Name:      volumePodMetadata.Name,
		MountPath: common.PodMetadataMountPath,
	}
	// volumeDockerLib provides the argoexec sidekick container access to the minion's
	// docker containers runtime files (e.g. /var/lib/docker/container). This is required
	// for argoexec to access the main container's logs and storage to upload output artifacts
	hostPathDir     = corev1.HostPathDirectory
	volumeDockerLib = corev1.Volume{
		Name: common.DockerLibVolumeName,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: common.DockerLibHostPath,
				Type: &hostPathDir,
			},
		},
	}
	volumeMountDockerLib = corev1.VolumeMount{
		Name:      volumeDockerLib.Name,
		MountPath: volumeDockerLib.VolumeSource.HostPath.Path,
		ReadOnly:  true,
	}

	// execEnvVars exposes various pod information as environment variables to the exec container
	execEnvVars = []corev1.EnvVar{
		envFromField(common.EnvVarHostIP, "status.hostIP"),
		envFromField(common.EnvVarPodIP, "status.podIP"),
		envFromField(common.EnvVarPodName, "metadata.name"),
		envFromField(common.EnvVarNamespace, "metadata.namespace"),
	}
)

// envFromField is a helper to return a EnvVar with the name and field
func envFromField(envVarName, fieldPath string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: envVarName,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath:  fieldPath,
			},
		},
	}
}

func (woc *wfOperationCtx) createWorkflowPod(nodeName string, tmpl *wfv1.Template) error {
	woc.log.Infof("Creating Pod: %s", nodeName)
	waitCtr, err := woc.newWaitContainer(tmpl)
	if err != nil {
		return err
	}
	mainCtrTmpl := tmpl.DeepCopy()
	var mainCtr corev1.Container
	if mainCtrTmpl.Container != nil {
		mainCtr = *mainCtrTmpl.Container
	} else {
		// script case
		mainCtr = corev1.Container{
			Image:   mainCtrTmpl.Script.Image,
			Command: mainCtrTmpl.Script.Command,
			Args:    []string{common.ScriptTemplateSourcePath},
		}
	}
	mainCtr.Name = common.MainContainerName
	t := true

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: woc.wf.NodeID(nodeName),
			Labels: map[string]string{
				common.LabelKeyWorkflow:     woc.wf.ObjectMeta.Name, // Allow filtering by pods related to specific workflow
				common.LabelKeyArgoWorkflow: "true",                 // Allow filtering by only argo workflow related pods
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
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				*waitCtr,
				mainCtr,
			},
			Volumes: []corev1.Volume{
				volumePodMetadata,
				volumeDockerLib,
			},
		},
	}

	// Add init container only if it needs input artifacts
	if len(mainCtrTmpl.Inputs.Artifacts) > 0 {
		initCtr := woc.newInitContainer(tmpl)
		pod.Spec.InitContainers = []corev1.Container{initCtr}
	}

	err = woc.addVolumeReferences(&pod, mainCtrTmpl)
	if err != nil {
		return err
	}

	err = addInputArtifactsVolumes(&pod, mainCtrTmpl)
	if err != nil {
		return err
	}
	woc.addOutputArtifactsRepoMetaData(&pod, mainCtrTmpl)

	if tmpl.Script != nil {
		addScriptVolume(&pod)
	}

	// Set the container template JSON in pod annotations, which executor will look to for artifact
	tmplBytes, err := json.Marshal(mainCtrTmpl)
	if err != nil {
		return err
	}
	pod.ObjectMeta.Annotations[common.AnnotationKeyTemplate] = string(tmplBytes)

	created, err := woc.controller.podCl.Create(&pod)
	if err != nil {
		if apierr.IsAlreadyExists(err) {
			// workflow pod names are deterministic. We can get here if
			// the controller crashes after creating the pod, but fails
			// to store the update to etc, and controller retries creation
			woc.log.Infof("pod %s already exists\n", nodeName)
			return nil
		}
		woc.log.Infof("Failed to create pod %s: %v\n", nodeName, err)
		return errors.InternalWrapError(err)
	}
	woc.log.Infof("Created pod: %s", created.Name)
	return nil
}

func (woc *wfOperationCtx) newInitContainer(tmpl *wfv1.Template) corev1.Container {
	ctr := woc.newExecContainer(common.InitContainerName, false)
	ctr.Command = []string{"sh", "-c"}
	argoExecCmd := fmt.Sprintf("echo sleeping; cat %s; sleep 10; find /argo; echo done", common.PodMetadataAnnotationsPath)
	ctr.Args = []string{argoExecCmd}
	ctr.VolumeMounts = []corev1.VolumeMount{
		volumeMountPodMetadata,
	}
	return *ctr
}

func (woc *wfOperationCtx) newWaitContainer(tmpl *wfv1.Template) (*corev1.Container, error) {
	ctr := woc.newExecContainer(common.WaitContainerName, false)
	ctr.Command = []string{"sh", "-c"}
	argoExecCmd := fmt.Sprintf("echo sleeping; cat %s; sleep 10; echo done", common.PodMetadataAnnotationsPath)
	ctr.Args = []string{argoExecCmd}
	ctr.VolumeMounts = []corev1.VolumeMount{
		volumeMountPodMetadata,
		volumeMountDockerLib,
	}
	return ctr, nil
}

func (woc *wfOperationCtx) newExecContainer(name string, privileged bool) *corev1.Container {
	exec := corev1.Container{
		Name:  name,
		Image: woc.controller.Config.ExecutorImage,
		Env:   execEnvVars,
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("0.5"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("0.1"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
		},
		SecurityContext: &corev1.SecurityContext{
			Privileged: &privileged,
		},
	}
	return &exec
}

// addVolumeReferences adds any volumeMounts that a container referencing, to the pod spec
func (woc *wfOperationCtx) addVolumeReferences(pod *corev1.Pod, tmpl *wfv1.Template) error {
	for _, volMnt := range tmpl.Container.VolumeMounts {
		vol := getVolByName(volMnt.Name, woc.wf.Spec.Volumes)
		if vol == nil {
			return errors.Errorf(errors.CodeBadRequest, "volume '%s' not found in workflow spec", volMnt.Name)
		}
		if len(pod.Spec.Volumes) == 0 {
			pod.Spec.Volumes = make([]corev1.Volume, 0)
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, *vol)
	}
	return nil
}

func getVolByName(name string, vols []corev1.Volume) *corev1.Volume {
	for _, vol := range vols {
		if vol.Name == name {
			return &vol
		}
	}
	return nil
}

// addInputArtifactVolumes sets up the artifacts volume to the pod if the user's container requires input artifacts.
// To support input artifacts, the init container shares a empty dir volume with the main container.
// It is the responsibility of the init container to load all artifacts to the mounted emptydir location.
// (e.g. /inputs/artifacts/CODE). The shared emptydir is mapped to the correspoding location in the main container.
func addInputArtifactsVolumes(pod *corev1.Pod, tmpl *wfv1.Template) error {
	if len(tmpl.Inputs.Artifacts) == 0 {
		return nil
	}
	volName := "input-artifacts"
	artVol := corev1.Volume{
		Name: volName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, artVol)

	for i, initCtr := range pod.Spec.InitContainers {
		if initCtr.Name == common.InitContainerName {
			volMount := corev1.VolumeMount{
				Name:      volName,
				MountPath: common.ExecutorArtifactBaseDir,
			}
			initCtr.VolumeMounts = append(initCtr.VolumeMounts, volMount)

			// HACK: debug purposes. sleep to experiment with init container artifacts
			initCtr.Command = []string{"sh", "-c"}
			initCtr.Args = []string{"sleep 999999; echo done"}

			pod.Spec.InitContainers[i] = initCtr
			break
		}
	}

	mainCtrIndex := 0
	var mainCtr *corev1.Container
	for i, ctr := range pod.Spec.Containers {
		if ctr.Name == common.MainContainerName {
			mainCtrIndex = i
			mainCtr = &ctr
			break
		}
		if ctr.Name == common.WaitContainerName {
			// HACK: debug purposes. sleep to experiment with wait container artifacts
			ctr.Command = []string{"sh", "-c"}
			ctr.Args = []string{"sleep 999999; echo done"}
			pod.Spec.Containers[i] = ctr
		}
	}
	if mainCtr == nil {
		errors.InternalError("Could not find main container in pod spec")
	}
	if mainCtr.VolumeMounts == nil {
		mainCtr.VolumeMounts = []corev1.VolumeMount{}
	}
	// TODO: the order in which we construct the volume mounts may matter,
	// especially if they are overlapping.
	for artName, art := range tmpl.Inputs.Artifacts {
		if art == nil {
			return errors.Errorf(errors.CodeBadRequest, "inputs.artifacts.%s did not specify a path", artName)
		}
		volMount := corev1.VolumeMount{
			Name:      volName,
			MountPath: art.Path,
			SubPath:   artName,
		}
		mainCtr.VolumeMounts = append(mainCtr.VolumeMounts, volMount)
	}
	pod.Spec.Containers[mainCtrIndex] = *mainCtr
	return nil
}

// addOutputArtifactsRepoMetaData updates the template with artifact repository information configured in the controller.
// This is skipped for artifacts which have explicitly set an output artifact location in the template
func (woc *wfOperationCtx) addOutputArtifactsRepoMetaData(pod *corev1.Pod, tmpl *wfv1.Template) {
	for artName, art := range tmpl.Outputs.Artifacts {
		if art.Destination != nil {
			// The artifact destination was explicitly set in the template. Skip
			continue
		}
		if woc.controller.Config.ArtifactRepository.S3 != nil {
			// artifacts are stored in S3 using the following formula:
			// <repo_key_prefix>/<worflow_name>/<node_id>/<artifact_name>
			// (e.g. myworkflowartifacts/argo-wf-fhljp/argo-wf-fhljp-123291312382/CODE)
			// TODO: will need to support more advanced organization of artifacts such as dated
			// (e.g. myworkflowartifacts/2017/10/31/... )
			keyPrefix := ""
			if woc.controller.Config.ArtifactRepository.S3.KeyPrefix != "" {
				keyPrefix = woc.controller.Config.ArtifactRepository.S3.KeyPrefix + "/"
			}
			artLocationKey := fmt.Sprintf("%s%s/%s/%s", keyPrefix, pod.Labels[common.LabelKeyWorkflow], pod.ObjectMeta.Name, artName)
			art.Destination = &wfv1.ArtifactDestination{
				S3: &wfv1.S3ArtifactDestination{
					S3Bucket: woc.controller.Config.ArtifactRepository.S3.S3Bucket,
					Key:      artLocationKey,
				},
			}
		}
		tmpl.Outputs.Artifacts[artName] = art
	}
}

// addScriptVolume sets up the shared volume between init container and main container
// containing the template script source code
func addScriptVolume(pod *corev1.Pod) {
	volName := "script"
	scriptVol := corev1.Volume{
		Name: volName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, scriptVol)

	for i, initCtr := range pod.Spec.InitContainers {
		if initCtr.Name == common.InitContainerName {
			volMount := corev1.VolumeMount{
				Name:      volName,
				MountPath: common.ScriptTemplateEmptyDir,
			}
			initCtr.VolumeMounts = append(initCtr.VolumeMounts, volMount)

			// HACK: debug purposes. sleep to experiment with init container artifacts
			initCtr.Command = []string{"sh", "-c"}
			initCtr.Args = []string{"sleep 999999; echo done"}

			pod.Spec.InitContainers[i] = initCtr
			break
		}
	}
	found := false
	for i, ctr := range pod.Spec.Containers {
		if ctr.Name == common.MainContainerName {
			volMount := corev1.VolumeMount{
				Name:      volName,
				MountPath: common.ScriptTemplateEmptyDir,
			}
			if ctr.VolumeMounts == nil {
				ctr.VolumeMounts = []corev1.VolumeMount{volMount}
			} else {
				ctr.VolumeMounts = append(ctr.VolumeMounts, volMount)
			}
			pod.Spec.Containers[i] = ctr
			found = true
			break
		}
		if ctr.Name == common.WaitContainerName {
			// HACK: debug purposes. sleep to experiment with wait container artifacts
			ctr.Command = []string{"sh", "-c"}
			ctr.Args = []string{"sleep 999999; echo done"}
			pod.Spec.Containers[i] = ctr
		}
	}
	if !found {
		panic("asdf")
	}
}
