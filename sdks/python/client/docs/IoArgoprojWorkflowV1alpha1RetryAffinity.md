# IoArgoprojWorkflowV1alpha1RetryAffinity

RetryAffinity prevents running steps on the same host.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**node_anti_affinity** | **object** | RetryNodeAntiAffinity is a placeholder for future expansion, only empty nodeAntiAffinity is allowed. In order to prevent running steps on the same host, it uses \&quot;kubernetes.io/hostname\&quot;. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_retry_affinity import IoArgoprojWorkflowV1alpha1RetryAffinity

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1RetryAffinity from a JSON string
io_argoproj_workflow_v1alpha1_retry_affinity_instance = IoArgoprojWorkflowV1alpha1RetryAffinity.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1RetryAffinity.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_retry_affinity_dict = io_argoproj_workflow_v1alpha1_retry_affinity_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1RetryAffinity from a dict
io_argoproj_workflow_v1alpha1_retry_affinity_form_dict = io_argoproj_workflow_v1alpha1_retry_affinity.from_dict(io_argoproj_workflow_v1alpha1_retry_affinity_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


