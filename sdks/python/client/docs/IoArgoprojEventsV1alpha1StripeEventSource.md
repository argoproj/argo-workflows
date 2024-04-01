# IoArgoprojEventsV1alpha1StripeEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_key** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**create_webhook** | **bool** |  | [optional] 
**event_filter** | **List[str]** |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**webhook** | [**IoArgoprojEventsV1alpha1WebhookContext**](IoArgoprojEventsV1alpha1WebhookContext.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_stripe_event_source import IoArgoprojEventsV1alpha1StripeEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1StripeEventSource from a JSON string
io_argoproj_events_v1alpha1_stripe_event_source_instance = IoArgoprojEventsV1alpha1StripeEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1StripeEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_stripe_event_source_dict = io_argoproj_events_v1alpha1_stripe_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1StripeEventSource from a dict
io_argoproj_events_v1alpha1_stripe_event_source_form_dict = io_argoproj_events_v1alpha1_stripe_event_source.from_dict(io_argoproj_events_v1alpha1_stripe_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


