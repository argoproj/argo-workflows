# IoArgoprojEventsV1alpha1HTTPTrigger


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**basic_auth** | [**IoArgoprojEventsV1alpha1BasicAuth**](IoArgoprojEventsV1alpha1BasicAuth.md) |  | [optional] 
**headers** | **Dict[str, str]** |  | [optional] 
**method** | **str** |  | [optional] 
**parameters** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) | Parameters is the list of key-value extracted from event&#39;s payload that are applied to the HTTP trigger resource. | [optional] 
**payload** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) |  | [optional] 
**secure_headers** | [**List[IoArgoprojEventsV1alpha1SecureHeader]**](IoArgoprojEventsV1alpha1SecureHeader.md) |  | [optional] 
**timeout** | **str** |  | [optional] 
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  | [optional] 
**url** | **str** | URL refers to the URL to send HTTP request to. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_http_trigger import IoArgoprojEventsV1alpha1HTTPTrigger

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1HTTPTrigger from a JSON string
io_argoproj_events_v1alpha1_http_trigger_instance = IoArgoprojEventsV1alpha1HTTPTrigger.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1HTTPTrigger.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_http_trigger_dict = io_argoproj_events_v1alpha1_http_trigger_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1HTTPTrigger from a dict
io_argoproj_events_v1alpha1_http_trigger_form_dict = io_argoproj_events_v1alpha1_http_trigger.from_dict(io_argoproj_events_v1alpha1_http_trigger_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


