# IoArgoprojWorkflowV1alpha1WorkflowStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artifact_gc_status** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1ArtGcStatus**](io.argoproj.workflow.v1alpha1.ArtGCStatus.md)> |  | [optional]
**artifact_repository_ref** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1ArtifactRepositoryRefStatus**](io.argoproj.workflow.v1alpha1.ArtifactRepositoryRefStatus.md)> |  | [optional]
**compressed_nodes** | Option<**String**> | Compressed and base64 decoded Nodes map | [optional]
**conditions** | Option<[**Vec<crate::models::IoArgoprojWorkflowV1alpha1Condition>**](io.argoproj.workflow.v1alpha1.Condition.md)> | Conditions is a list of conditions the Workflow may have | [optional]
**estimated_duration** | Option<**i32**> | EstimatedDuration in seconds. | [optional]
**finished_at** | Option<**String**> | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional]
**message** | Option<**String**> | A human readable message indicating details about why the workflow is in this condition. | [optional]
**nodes** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojWorkflowV1alpha1NodeStatus>**](io.argoproj.workflow.v1alpha1.NodeStatus.md)> | Nodes is a mapping between a node ID and the node's status. | [optional]
**offload_node_status_version** | Option<**String**> | Whether on not node status has been offloaded to a database. If exists, then Nodes and CompressedNodes will be empty. This will actually be populated with a hash of the offloaded data. | [optional]
**outputs** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1Outputs**](io.argoproj.workflow.v1alpha1.Outputs.md)> |  | [optional]
**persistent_volume_claims** | Option<[**Vec<crate::models::Volume>**](Volume.md)> | PersistentVolumeClaims tracks all PVCs that were created as part of the io.argoproj.workflow.v1alpha1. The contents of this list are drained at the end of the workflow. | [optional]
**phase** | Option<**String**> | Phase a simple, high-level summary of where the workflow is in its lifecycle. Will be \"\" (Unknown), \"Pending\", or \"Running\" before the workflow is completed, and \"Succeeded\", \"Failed\" or \"Error\" once the workflow has completed. | [optional]
**progress** | Option<**String**> | Progress to completion | [optional]
**resources_duration** | Option<**::std::collections::HashMap<String, i64>**> | ResourcesDuration is the total for the workflow | [optional]
**started_at** | Option<**String**> | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional]
**stored_templates** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojWorkflowV1alpha1Template>**](io.argoproj.workflow.v1alpha1.Template.md)> | StoredTemplates is a mapping between a template ref and the node's status. | [optional]
**stored_workflow_template_spec** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1WorkflowSpec**](io.argoproj.workflow.v1alpha1.WorkflowSpec.md)> |  | [optional]
**synchronization** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1SynchronizationStatus**](io.argoproj.workflow.v1alpha1.SynchronizationStatus.md)> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


