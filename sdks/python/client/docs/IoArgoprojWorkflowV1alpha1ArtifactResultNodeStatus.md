# IoArgoprojWorkflowV1alpha1ArtifactResultNodeStatus

ArtifactResultNodeStatus describes the result of the deletion on a given node

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artifact_results** | [**Dict[str, IoArgoprojWorkflowV1alpha1ArtifactResult]**](IoArgoprojWorkflowV1alpha1ArtifactResult.md) | ArtifactResults maps Artifact name to result of the deletion | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_artifact_result_node_status import IoArgoprojWorkflowV1alpha1ArtifactResultNodeStatus

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1ArtifactResultNodeStatus from a JSON string
io_argoproj_workflow_v1alpha1_artifact_result_node_status_instance = IoArgoprojWorkflowV1alpha1ArtifactResultNodeStatus.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1ArtifactResultNodeStatus.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_artifact_result_node_status_dict = io_argoproj_workflow_v1alpha1_artifact_result_node_status_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1ArtifactResultNodeStatus from a dict
io_argoproj_workflow_v1alpha1_artifact_result_node_status_form_dict = io_argoproj_workflow_v1alpha1_artifact_result_node_status.from_dict(io_argoproj_workflow_v1alpha1_artifact_result_node_status_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


