# IoArgoprojEventsV1alpha1K8SResourcePolicy


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**backoff** | [**IoArgoprojEventsV1alpha1Backoff**](IoArgoprojEventsV1alpha1Backoff.md) |  | [optional] 
**error_on_backoff_timeout** | **bool** |  | [optional] 
**labels** | **Dict[str, str]** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_k8_s_resource_policy import IoArgoprojEventsV1alpha1K8SResourcePolicy

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1K8SResourcePolicy from a JSON string
io_argoproj_events_v1alpha1_k8_s_resource_policy_instance = IoArgoprojEventsV1alpha1K8SResourcePolicy.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1K8SResourcePolicy.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_k8_s_resource_policy_dict = io_argoproj_events_v1alpha1_k8_s_resource_policy_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1K8SResourcePolicy from a dict
io_argoproj_events_v1alpha1_k8_s_resource_policy_form_dict = io_argoproj_events_v1alpha1_k8_s_resource_policy.from_dict(io_argoproj_events_v1alpha1_k8_s_resource_policy_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


