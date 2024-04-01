# IoArgoprojWorkflowV1alpha1WorkflowMetadata


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**annotations** | **Dict[str, str]** |  | [optional] 
**labels** | **Dict[str, str]** |  | [optional] 
**labels_from** | [**Dict[str, IoArgoprojWorkflowV1alpha1LabelValueFrom]**](IoArgoprojWorkflowV1alpha1LabelValueFrom.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_workflow_metadata import IoArgoprojWorkflowV1alpha1WorkflowMetadata

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowMetadata from a JSON string
io_argoproj_workflow_v1alpha1_workflow_metadata_instance = IoArgoprojWorkflowV1alpha1WorkflowMetadata.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1WorkflowMetadata.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_workflow_metadata_dict = io_argoproj_workflow_v1alpha1_workflow_metadata_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowMetadata from a dict
io_argoproj_workflow_v1alpha1_workflow_metadata_form_dict = io_argoproj_workflow_v1alpha1_workflow_metadata.from_dict(io_argoproj_workflow_v1alpha1_workflow_metadata_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


