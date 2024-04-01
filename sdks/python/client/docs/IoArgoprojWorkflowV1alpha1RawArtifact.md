# IoArgoprojWorkflowV1alpha1RawArtifact

RawArtifact allows raw string content to be placed as an artifact in a container

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**data** | **str** | Data is the string contents of the artifact | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_raw_artifact import IoArgoprojWorkflowV1alpha1RawArtifact

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1RawArtifact from a JSON string
io_argoproj_workflow_v1alpha1_raw_artifact_instance = IoArgoprojWorkflowV1alpha1RawArtifact.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1RawArtifact.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_raw_artifact_dict = io_argoproj_workflow_v1alpha1_raw_artifact_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1RawArtifact from a dict
io_argoproj_workflow_v1alpha1_raw_artifact_form_dict = io_argoproj_workflow_v1alpha1_raw_artifact.from_dict(io_argoproj_workflow_v1alpha1_raw_artifact_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


