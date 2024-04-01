# IoArgoprojEventsV1alpha1Template


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**affinity** | [**Affinity**](Affinity.md) |  | [optional] 
**container** | [**Container**](Container.md) |  | [optional] 
**image_pull_secrets** | [**List[LocalObjectReference]**](LocalObjectReference.md) |  | [optional] 
**metadata** | [**IoArgoprojEventsV1alpha1Metadata**](IoArgoprojEventsV1alpha1Metadata.md) |  | [optional] 
**node_selector** | **Dict[str, str]** |  | [optional] 
**priority** | **int** |  | [optional] 
**priority_class_name** | **str** |  | [optional] 
**security_context** | [**PodSecurityContext**](PodSecurityContext.md) |  | [optional] 
**service_account_name** | **str** |  | [optional] 
**tolerations** | [**List[Toleration]**](Toleration.md) |  | [optional] 
**volumes** | [**List[Volume]**](Volume.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_template import IoArgoprojEventsV1alpha1Template

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1Template from a JSON string
io_argoproj_events_v1alpha1_template_instance = IoArgoprojEventsV1alpha1Template.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1Template.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_template_dict = io_argoproj_events_v1alpha1_template_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1Template from a dict
io_argoproj_events_v1alpha1_template_form_dict = io_argoproj_events_v1alpha1_template.from_dict(io_argoproj_events_v1alpha1_template_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


