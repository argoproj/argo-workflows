# IoArgoprojWorkflowV1alpha1DataSource

DataSource sources external data into a data template

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artifact_paths** | [**IoArgoprojWorkflowV1alpha1ArtifactPaths**](IoArgoprojWorkflowV1alpha1ArtifactPaths.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_data_source import IoArgoprojWorkflowV1alpha1DataSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1DataSource from a JSON string
io_argoproj_workflow_v1alpha1_data_source_instance = IoArgoprojWorkflowV1alpha1DataSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1DataSource.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_data_source_dict = io_argoproj_workflow_v1alpha1_data_source_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1DataSource from a dict
io_argoproj_workflow_v1alpha1_data_source_form_dict = io_argoproj_workflow_v1alpha1_data_source.from_dict(io_argoproj_workflow_v1alpha1_data_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


