# IoArgoprojWorkflowV1alpha1HTTPArtifact

HTTPArtifact allows a file served on HTTP to be placed as an input artifact in a container

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth** | [**IoArgoprojWorkflowV1alpha1HTTPAuth**](IoArgoprojWorkflowV1alpha1HTTPAuth.md) |  | [optional] 
**headers** | [**List[IoArgoprojWorkflowV1alpha1Header]**](IoArgoprojWorkflowV1alpha1Header.md) | Headers are an optional list of headers to send with HTTP requests for artifacts | [optional] 
**url** | **str** | URL of the artifact | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_http_artifact import IoArgoprojWorkflowV1alpha1HTTPArtifact

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1HTTPArtifact from a JSON string
io_argoproj_workflow_v1alpha1_http_artifact_instance = IoArgoprojWorkflowV1alpha1HTTPArtifact.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1HTTPArtifact.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_http_artifact_dict = io_argoproj_workflow_v1alpha1_http_artifact_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1HTTPArtifact from a dict
io_argoproj_workflow_v1alpha1_http_artifact_form_dict = io_argoproj_workflow_v1alpha1_http_artifact.from_dict(io_argoproj_workflow_v1alpha1_http_artifact_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


