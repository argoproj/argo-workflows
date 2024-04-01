# IoArgoprojWorkflowV1alpha1Column

Column is a custom column that will be exposed in the Workflow List View.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**key** | **str** | The key of the label or annotation, e.g., \&quot;workflows.argoproj.io/completed\&quot;. | 
**name** | **str** | The name of this column, e.g., \&quot;Workflow Completed\&quot;. | 
**type** | **str** | The type of this column, \&quot;label\&quot; or \&quot;annotation\&quot;. | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_column import IoArgoprojWorkflowV1alpha1Column

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Column from a JSON string
io_argoproj_workflow_v1alpha1_column_instance = IoArgoprojWorkflowV1alpha1Column.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Column.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_column_dict = io_argoproj_workflow_v1alpha1_column_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Column from a dict
io_argoproj_workflow_v1alpha1_column_form_dict = io_argoproj_workflow_v1alpha1_column.from_dict(io_argoproj_workflow_v1alpha1_column_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


