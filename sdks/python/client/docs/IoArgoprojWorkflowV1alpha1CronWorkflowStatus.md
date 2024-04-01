# IoArgoprojWorkflowV1alpha1CronWorkflowStatus

CronWorkflowStatus is the status of a CronWorkflow

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**active** | [**List[ObjectReference]**](ObjectReference.md) | Active is a list of active workflows stemming from this CronWorkflow | 
**conditions** | [**List[IoArgoprojWorkflowV1alpha1Condition]**](IoArgoprojWorkflowV1alpha1Condition.md) | Conditions is a list of conditions the CronWorkflow may have | 
**failed** | **int** | Failed is a counter of how many times a child workflow terminated in failed or errored state | 
**last_scheduled_time** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | 
**phase** | **str** | Phase defines the cron workflow phase. It is changed to Stopped when the stopping condition is achieved which stops new CronWorkflows from running | 
**succeeded** | **int** | Succeeded is a counter of how many times the child workflows had success | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_cron_workflow_status import IoArgoprojWorkflowV1alpha1CronWorkflowStatus

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1CronWorkflowStatus from a JSON string
io_argoproj_workflow_v1alpha1_cron_workflow_status_instance = IoArgoprojWorkflowV1alpha1CronWorkflowStatus.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1CronWorkflowStatus.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_cron_workflow_status_dict = io_argoproj_workflow_v1alpha1_cron_workflow_status_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1CronWorkflowStatus from a dict
io_argoproj_workflow_v1alpha1_cron_workflow_status_form_dict = io_argoproj_workflow_v1alpha1_cron_workflow_status.from_dict(io_argoproj_workflow_v1alpha1_cron_workflow_status_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


