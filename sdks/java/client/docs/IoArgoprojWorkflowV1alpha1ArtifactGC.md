

# IoArgoprojWorkflowV1alpha1ArtifactGC

ArtifactGC describes how to delete artifacts from completed Workflows - this is embedded into the WorkflowLevelArtifactGC, and also used for individual Artifacts to override that as needed

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**env** | [**List&lt;io.kubernetes.client.openapi.models.V1EnvVar&gt;**](io.kubernetes.client.openapi.models.V1EnvVar.md) | Env is an optional field for specifying environment variables that should be assigned to the Pod doing the deletion |  [optional]
**podMetadata** | [**IoArgoprojWorkflowV1alpha1Metadata**](IoArgoprojWorkflowV1alpha1Metadata.md) |  |  [optional]
**serviceAccountName** | **String** | ServiceAccountName is an optional field for specifying the Service Account that should be assigned to the Pod doing the deletion |  [optional]
**strategy** | **String** | Strategy is the strategy to use. |  [optional]
**volumeMounts** | [**List&lt;io.kubernetes.client.openapi.models.V1VolumeMount&gt;**](io.kubernetes.client.openapi.models.V1VolumeMount.md) | VolumeMounts is an optional field for specifying volume mounts that should be assigned to the Pod doing the deletion |  [optional]
**volumes** | [**List&lt;io.kubernetes.client.openapi.models.V1Volume&gt;**](io.kubernetes.client.openapi.models.V1Volume.md) | Volumes is an optional field for specifying volumes that should be assigned to the Pod doing the deletion |  [optional]



