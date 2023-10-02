# IoArgoprojWorkflowV1alpha1WorkflowLevelArtifactGC

WorkflowLevelArtifactGC describes how to delete artifacts from completed Workflows - this spec is used on the Workflow level

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**force_finalizer_removal** | **bool** | ForceFinalizerRemoval: if set to true, the finalizer will be removed in the case that Artifact GC fails | [optional] 
**pod_metadata** | [**IoArgoprojWorkflowV1alpha1Metadata**](IoArgoprojWorkflowV1alpha1Metadata.md) |  | [optional] 
**pod_spec_patch** | **str** | PodSpecPatch holds strategic merge patch to apply against the artgc pod spec. | [optional] 
**service_account_name** | **str** | ServiceAccountName is an optional field for specifying the Service Account that should be assigned to the Pod doing the deletion | [optional] 
**strategy** | **str** | Strategy is the strategy to use. | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


