# IoArgoprojWorkflowV1alpha1SemaphoreHolding


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**holders** | **List[str]** | Holders stores the list of current holder names in the io.argoproj.workflow.v1alpha1. | [optional] 
**semaphore** | **str** | Semaphore stores the semaphore name. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_semaphore_holding import IoArgoprojWorkflowV1alpha1SemaphoreHolding

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1SemaphoreHolding from a JSON string
io_argoproj_workflow_v1alpha1_semaphore_holding_instance = IoArgoprojWorkflowV1alpha1SemaphoreHolding.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1SemaphoreHolding.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_semaphore_holding_dict = io_argoproj_workflow_v1alpha1_semaphore_holding_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1SemaphoreHolding from a dict
io_argoproj_workflow_v1alpha1_semaphore_holding_form_dict = io_argoproj_workflow_v1alpha1_semaphore_holding.from_dict(io_argoproj_workflow_v1alpha1_semaphore_holding_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


