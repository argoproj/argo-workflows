# IoArgoprojWorkflowV1alpha1StopStrategy

StopStrategy defines if the cron workflow will stop being triggered once a certain condition has been reached, involving a number of runs of the workflow

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**condition** | **str** | Condition defines a condition that stops scheduling workflows when evaluates to true. Use the keywords &#x60;failed&#x60; or &#x60;succeeded&#x60; to access the number of failed or successful child workflows. | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_stop_strategy import IoArgoprojWorkflowV1alpha1StopStrategy

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1StopStrategy from a JSON string
io_argoproj_workflow_v1alpha1_stop_strategy_instance = IoArgoprojWorkflowV1alpha1StopStrategy.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1StopStrategy.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_stop_strategy_dict = io_argoproj_workflow_v1alpha1_stop_strategy_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1StopStrategy from a dict
io_argoproj_workflow_v1alpha1_stop_strategy_form_dict = io_argoproj_workflow_v1alpha1_stop_strategy.from_dict(io_argoproj_workflow_v1alpha1_stop_strategy_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


