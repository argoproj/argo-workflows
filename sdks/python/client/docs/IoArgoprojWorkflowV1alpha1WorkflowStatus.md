# IoArgoprojWorkflowV1alpha1WorkflowStatus

WorkflowStatus contains overall status information about a workflow

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artifact_gc_status** | [**IoArgoprojWorkflowV1alpha1ArtGCStatus**](IoArgoprojWorkflowV1alpha1ArtGCStatus.md) |  | [optional] 
**artifact_repository_ref** | [**IoArgoprojWorkflowV1alpha1ArtifactRepositoryRefStatus**](IoArgoprojWorkflowV1alpha1ArtifactRepositoryRefStatus.md) |  | [optional] 
**compressed_nodes** | **str** | Compressed and base64 decoded Nodes map | [optional] 
**conditions** | [**List[IoArgoprojWorkflowV1alpha1Condition]**](IoArgoprojWorkflowV1alpha1Condition.md) | Conditions is a list of conditions the Workflow may have | [optional] 
**estimated_duration** | **int** | EstimatedDuration in seconds. | [optional] 
**finished_at** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional] 
**message** | **str** | A human readable message indicating details about why the workflow is in this condition. | [optional] 
**nodes** | [**Dict[str, IoArgoprojWorkflowV1alpha1NodeStatus]**](IoArgoprojWorkflowV1alpha1NodeStatus.md) | Nodes is a mapping between a node ID and the node&#39;s status. | [optional] 
**offload_node_status_version** | **str** | Whether on not node status has been offloaded to a database. If exists, then Nodes and CompressedNodes will be empty. This will actually be populated with a hash of the offloaded data. | [optional] 
**outputs** | [**IoArgoprojWorkflowV1alpha1Outputs**](IoArgoprojWorkflowV1alpha1Outputs.md) |  | [optional] 
**persistent_volume_claims** | [**List[Volume]**](Volume.md) | PersistentVolumeClaims tracks all PVCs that were created as part of the io.argoproj.workflow.v1alpha1. The contents of this list are drained at the end of the workflow. | [optional] 
**phase** | **str** | Phase a simple, high-level summary of where the workflow is in its lifecycle. Will be \&quot;\&quot; (Unknown), \&quot;Pending\&quot;, or \&quot;Running\&quot; before the workflow is completed, and \&quot;Succeeded\&quot;, \&quot;Failed\&quot; or \&quot;Error\&quot; once the workflow has completed. | [optional] 
**progress** | **str** | Progress to completion | [optional] 
**resources_duration** | **Dict[str, int]** | ResourcesDuration is the total for the workflow | [optional] 
**started_at** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional] 
**stored_templates** | [**Dict[str, IoArgoprojWorkflowV1alpha1Template]**](IoArgoprojWorkflowV1alpha1Template.md) | StoredTemplates is a mapping between a template ref and the node&#39;s status. | [optional] 
**stored_workflow_template_spec** | [**IoArgoprojWorkflowV1alpha1WorkflowSpec**](IoArgoprojWorkflowV1alpha1WorkflowSpec.md) |  | [optional] 
**synchronization** | [**IoArgoprojWorkflowV1alpha1SynchronizationStatus**](IoArgoprojWorkflowV1alpha1SynchronizationStatus.md) |  | [optional] 
**task_results_completion_status** | **Dict[str, bool]** | TaskResultsCompletionStatus tracks task result completion status (mapped by node ID). Used to prevent premature archiving and garbage collection. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_workflow_status import IoArgoprojWorkflowV1alpha1WorkflowStatus

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowStatus from a JSON string
io_argoproj_workflow_v1alpha1_workflow_status_instance = IoArgoprojWorkflowV1alpha1WorkflowStatus.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1WorkflowStatus.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_workflow_status_dict = io_argoproj_workflow_v1alpha1_workflow_status_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowStatus from a dict
io_argoproj_workflow_v1alpha1_workflow_status_form_dict = io_argoproj_workflow_v1alpha1_workflow_status.from_dict(io_argoproj_workflow_v1alpha1_workflow_status_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


