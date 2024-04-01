# IoArgoprojWorkflowV1alpha1SemaphoreRef

SemaphoreRef is a reference of Semaphore

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**config_map_key_ref** | [**ConfigMapKeySelector**](ConfigMapKeySelector.md) |  | [optional] 
**namespace** | **str** | Namespace is the namespace of the configmap, default: [namespace of workflow] | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_semaphore_ref import IoArgoprojWorkflowV1alpha1SemaphoreRef

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1SemaphoreRef from a JSON string
io_argoproj_workflow_v1alpha1_semaphore_ref_instance = IoArgoprojWorkflowV1alpha1SemaphoreRef.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1SemaphoreRef.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_semaphore_ref_dict = io_argoproj_workflow_v1alpha1_semaphore_ref_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1SemaphoreRef from a dict
io_argoproj_workflow_v1alpha1_semaphore_ref_form_dict = io_argoproj_workflow_v1alpha1_semaphore_ref.from_dict(io_argoproj_workflow_v1alpha1_semaphore_ref_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


