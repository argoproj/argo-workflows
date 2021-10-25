# V1alpha1WorkflowStatus

WorkflowStatus contains overall status information about a workflow
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**compressed_nodes** | **str** | Compressed and base64 decoded Nodes map | [optional] 
**conditions** | [**list[V1alpha1Condition]**](V1alpha1Condition.md) | Conditions is a list of conditions the Workflow may have | [optional] 
**estimated_duration** | **int** | EstimatedDuration in seconds. | [optional] 
**finished_at** | **datetime** | Time at which this workflow completed | [optional] 
**message** | **str** | A human readable message indicating details about why the workflow is in this condition. | [optional] 
**nodes** | [**dict(str, V1alpha1NodeStatus)**](V1alpha1NodeStatus.md) | Nodes is a mapping between a node ID and the node&#39;s status. | [optional] 
**offload_node_status_version** | **str** | Whether on not node status has been offloaded to a database. If exists, then Nodes and CompressedNodes will be empty. This will actually be populated with a hash of the offloaded data. | [optional] 
**outputs** | [**V1alpha1Outputs**](V1alpha1Outputs.md) |  | [optional] 
**persistent_volume_claims** | [**list[V1Volume]**](V1Volume.md) | PersistentVolumeClaims tracks all PVCs that were created as part of the io.argoproj.workflow.v1alpha1. The contents of this list are drained at the end of the workflow. | [optional] 
**phase** | **str** | Phase a simple, high-level summary of where the workflow is in its lifecycle. | [optional] 
**progress** | **str** | Progress to completion | [optional] 
**resources_duration** | **dict(str, int)** | ResourcesDuration is the total for the workflow | [optional] 
**started_at** | **datetime** | Time at which this workflow started | [optional] 
**stored_templates** | [**dict(str, V1alpha1Template)**](V1alpha1Template.md) | StoredTemplates is a mapping between a template ref and the node&#39;s status. | [optional] 
**stored_workflow_template_spec** | [**V1alpha1WorkflowSpec**](V1alpha1WorkflowSpec.md) |  | [optional] 
**synchronization** | [**V1alpha1SynchronizationStatus**](V1alpha1SynchronizationStatus.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


