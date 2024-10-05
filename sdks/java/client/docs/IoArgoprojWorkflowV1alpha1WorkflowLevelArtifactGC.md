

# IoArgoprojWorkflowV1alpha1WorkflowLevelArtifactGC

WorkflowLevelArtifactGC describes how to delete artifacts from completed Workflows - this spec is used on the Workflow level

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**forceFinalizerRemoval** | **Boolean** | ForceFinalizerRemoval: if set to true, the finalizer will be removed in the case that Artifact GC fails |  [optional]
**podMetadata** | [**IoArgoprojWorkflowV1alpha1Metadata**](IoArgoprojWorkflowV1alpha1Metadata.md) |  |  [optional]
**podSpecPatch** | **String** | PodSpecPatch holds strategic merge patch to apply against the artgc pod spec. |  [optional]
**serviceAccountName** | **String** | ServiceAccountName is an optional field for specifying the Service Account that should be assigned to the Pod doing the deletion |  [optional]
**strategy** | **String** | Strategy is the strategy to use. |  [optional]



