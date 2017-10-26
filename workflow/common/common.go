package common

import (
	wfv1 "github.com/argoproj/argo/api/workflow/v1"
)

const (
	// Container names used in the workflow pod
	MainContainerName = "main"
	InitContainerName = "init"
	WaitContainerName = "wait"

	// PodMetadataVolumeName is the volume name defined in a workflow pod spec to expose pod metadata via downward API
	PodMetadataVolumeName = "podmetadata"

	// PodMetadataAnnotationsVolumePath is volume path for metadata.annotations in the downward API
	PodMetadataAnnotationsVolumePath = "annotations"
	// PodMetadataMountPath is the directory mount location for DownwardAPI volume containing pod metadata
	PodMetadataMountPath = "/argo/" + PodMetadataVolumeName
	// PodMetadataAnnotationsPath is the file path containing pod metadata annotations. Examined by argoexec
	PodMetadataAnnotationsPath = PodMetadataMountPath + "/" + PodMetadataAnnotationsVolumePath
	//
	PodStatusVolumePath = "podstatus"

	// DockerLibVolumeName is the volume name for the /var/lib/docker host path volume
	DockerLibVolumeName = "docker-lib"
	// DockerLibHostPath is the host directory path containing docker runtime state
	DockerLibHostPath = "/var/lib/docker"

	// AnnotationKeyNodeName is the pod metadata annotation key containing the workflow node name
	AnnotationKeyNodeName = wfv1.CRDFullName + "/node-name"
	// AnnotationKeyTemplate is the pod metadata annotation key containing the container template as JSON
	AnnotationKeyTemplate = wfv1.CRDFullName + "/template"

	// LabelKeyArgoWorkflow is the pod metadata label to indidcate this pod is part of a workflow
	LabelKeyArgoWorkflow = wfv1.CRDFullName + "/argo-workflow"
	// LabelKeyWorkflow is the pod metadata label to indidcate the associated workflow name
	LabelKeyWorkflow = wfv1.CRDFullName + "/workflow"

	// ExecutorArtifactBaseDir is the base directory in the init container in which artifacts will be copied to.
	// Each artifact will be named according to its input name (e.g: /argo/inputs/artifacts/CODE)
	ExecutorArtifactBaseDir = "/argo/inputs/artifacts"

	// Various environment variables containing pod information exposed to the executor container(s)

	// EnvVarHostIP contains the host IP which the container is executing on.
	// Used to communicate with kubelet directly (rather than API server).
	// Kubelet enables the wait sidekick to determine
	EnvVarHostIP = "ARGO_HOST_IP"
	// EnvVarPodIP contains the IP of the pod (currently unused)
	EnvVarPodIP = "ARGO_POD_IP"
	// EnvVarPodName contains the name of the pod (currently unused)
	EnvVarPodName = "ARGO_POD_NAME"
	// EnvVarNamespace contains the namespace of the pod (currently unused)
	EnvVarNamespace = "ARGO_NAMESPACE"
)
