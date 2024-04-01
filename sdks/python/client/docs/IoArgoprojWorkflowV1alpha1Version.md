# IoArgoprojWorkflowV1alpha1Version


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**build_date** | **str** |  | 
**compiler** | **str** |  | 
**git_commit** | **str** |  | 
**git_tag** | **str** |  | 
**git_tree_state** | **str** |  | 
**go_version** | **str** |  | 
**platform** | **str** |  | 
**version** | **str** |  | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_version import IoArgoprojWorkflowV1alpha1Version

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Version from a JSON string
io_argoproj_workflow_v1alpha1_version_instance = IoArgoprojWorkflowV1alpha1Version.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Version.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_version_dict = io_argoproj_workflow_v1alpha1_version_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Version from a dict
io_argoproj_workflow_v1alpha1_version_form_dict = io_argoproj_workflow_v1alpha1_version.from_dict(io_argoproj_workflow_v1alpha1_version_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


