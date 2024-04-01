# IoArgoprojWorkflowV1alpha1Data

Data is a data template

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**source** | [**IoArgoprojWorkflowV1alpha1DataSource**](IoArgoprojWorkflowV1alpha1DataSource.md) |  | 
**transformation** | [**List[IoArgoprojWorkflowV1alpha1TransformationStep]**](IoArgoprojWorkflowV1alpha1TransformationStep.md) | Transformation applies a set of transformations | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_data import IoArgoprojWorkflowV1alpha1Data

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Data from a JSON string
io_argoproj_workflow_v1alpha1_data_instance = IoArgoprojWorkflowV1alpha1Data.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Data.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_data_dict = io_argoproj_workflow_v1alpha1_data_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Data from a dict
io_argoproj_workflow_v1alpha1_data_form_dict = io_argoproj_workflow_v1alpha1_data.from_dict(io_argoproj_workflow_v1alpha1_data_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


