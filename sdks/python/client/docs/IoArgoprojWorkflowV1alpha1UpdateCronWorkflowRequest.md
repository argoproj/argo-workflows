# IoArgoprojWorkflowV1alpha1UpdateCronWorkflowRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**cron_workflow** | [**IoArgoprojWorkflowV1alpha1CronWorkflow**](IoArgoprojWorkflowV1alpha1CronWorkflow.md) |  | [optional] 
**name** | **str** | DEPRECATED: This field is ignored. | [optional] 
**namespace** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_update_cron_workflow_request import IoArgoprojWorkflowV1alpha1UpdateCronWorkflowRequest

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1UpdateCronWorkflowRequest from a JSON string
io_argoproj_workflow_v1alpha1_update_cron_workflow_request_instance = IoArgoprojWorkflowV1alpha1UpdateCronWorkflowRequest.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1UpdateCronWorkflowRequest.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_update_cron_workflow_request_dict = io_argoproj_workflow_v1alpha1_update_cron_workflow_request_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1UpdateCronWorkflowRequest from a dict
io_argoproj_workflow_v1alpha1_update_cron_workflow_request_form_dict = io_argoproj_workflow_v1alpha1_update_cron_workflow_request.from_dict(io_argoproj_workflow_v1alpha1_update_cron_workflow_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


