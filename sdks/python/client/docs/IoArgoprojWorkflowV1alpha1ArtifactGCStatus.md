# IoArgoprojWorkflowV1alpha1ArtifactGCStatus

ArtifactGCStatus describes the result of the deletion

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artifact_results_by_node** | [**Dict[str, IoArgoprojWorkflowV1alpha1ArtifactResultNodeStatus]**](IoArgoprojWorkflowV1alpha1ArtifactResultNodeStatus.md) | ArtifactResultsByNode maps Node name to result | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_artifact_gc_status import IoArgoprojWorkflowV1alpha1ArtifactGCStatus

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1ArtifactGCStatus from a JSON string
io_argoproj_workflow_v1alpha1_artifact_gc_status_instance = IoArgoprojWorkflowV1alpha1ArtifactGCStatus.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1ArtifactGCStatus.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_artifact_gc_status_dict = io_argoproj_workflow_v1alpha1_artifact_gc_status_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1ArtifactGCStatus from a dict
io_argoproj_workflow_v1alpha1_artifact_gc_status_form_dict = io_argoproj_workflow_v1alpha1_artifact_gc_status.from_dict(io_argoproj_workflow_v1alpha1_artifact_gc_status_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


