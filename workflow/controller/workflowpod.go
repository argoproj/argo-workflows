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

func (wfc *WorkflowController) createWorkflowPod(wf *wfv1.Workflow, nodeName string, tmpl *wfv1.Template, args *wfv1.Arguments) error {
	fmt.Printf("Creating Pod: %s\n", nodeName)
	initCtr, err := wfc.newInitContainer(tmpl)
	if err != nil {
		return err
	}
	waitCtr, err := wfc.newWaitContainer(tmpl)
	if err != nil {
		return err
	}
	mainCtr := tmpl.Container.DeepCopy()
	mainCtr.Name = "main"
	t := true

	tmplBytes, err := json.Marshal(tmpl)
	if err != nil {
		return err
	}

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: wf.NodeID(nodeName),
			Labels: map[string]string{
				common.LabelKeyWorkflow:     wf.ObjectMeta.Name, // Allow filtering by pods related to specific workflow
				common.LabelKeyArgoWorkflow: "true",             // Allow filtering by only argo workflow related pods
			},
			Annotations: map[string]string{
				common.AnnotationKeyNodeName: nodeName,
				common.AnnotationKeyTemplate: string(tmplBytes),
			},
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion:         wfv1.CRDFullName,
					Kind:               wfv1.CRDKind,
					Name:               wf.ObjectMeta.Name,
					UID:                wf.ObjectMeta.UID,
					BlockOwnerDeletion: &t,
				},
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			InitContainers: []corev1.Container{
				*initCtr,
			},
			Containers: []corev1.Container{
				*waitCtr,
				*mainCtr,
			},
			Volumes: []corev1.Volume{
				volumePodMetadata,
				volumeDockerLib,
			},
		},
	}

	created, err := wfc.podCl.Create(&pod)
	if err != nil {
		if apierr.IsAlreadyExists(err) {
			// workflow pod names are deterministic. We can get here if
			// the controller crashes after creating the pod, but fails
			// to store the update to etc, and controller retries creation
			fmt.Printf("pod %s already exists\n", nodeName)
			return nil
		}
		fmt.Printf("Failed to create pod %s: %v\n", nodeName, err)
		return errors.InternalWrapError(err)
	}
	fmt.Printf("Created pod: %v\n", created)
	return nil
}

func (wfc *WorkflowController) newInitContainer(tmpl *wfv1.Template) (*corev1.Container, error) {
	ctr := wfc.newExecContainer("init", false)
	ctr.Command = []string{"sh", "-c"}
	argoExecCmd := fmt.Sprintf("echo sleeping; cat %s; sleep 10; echo done", common.PodMetadataAnnotationsPath)
	ctr.Args = []string{argoExecCmd}
	ctr.VolumeMounts = []corev1.VolumeMount{
		volumeMountPodMetadata,
	}
	return ctr, nil
}

func (wfc *WorkflowController) newWaitContainer(tmpl *wfv1.Template) (*corev1.Container, error) {
	ctr := wfc.newExecContainer("wait", true)
	ctr.Command = []string{"sh", "-c"}
	argoExecCmd := fmt.Sprintf("echo sleeping; cat %s; sleep 999999; echo done", common.PodMetadataAnnotationsPath)
	ctr.Args = []string{argoExecCmd}
	ctr.VolumeMounts = []corev1.VolumeMount{
		volumeMountPodMetadata,
		volumeMountDockerLib,
	}
	return ctr, nil
}

func (wfc *WorkflowController) newExecContainer(name string, privileged bool) *corev1.Container {
	exec := corev1.Container{
		Name:  name,
		Image: wfc.ArgoExecImage,
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
