# IoArgoprojEventsV1alpha1WebhookContext


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**endpoint** | **str** |  | [optional] 
**max_payload_size** | **str** |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**method** | **str** |  | [optional] 
**port** | **str** | Port on which HTTP server is listening for incoming events. | [optional] 
**server_cert_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**server_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**url** | **str** | URL is the url of the server. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_webhook_context import IoArgoprojEventsV1alpha1WebhookContext

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1WebhookContext from a JSON string
io_argoproj_events_v1alpha1_webhook_context_instance = IoArgoprojEventsV1alpha1WebhookContext.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1WebhookContext.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_webhook_context_dict = io_argoproj_events_v1alpha1_webhook_context_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1WebhookContext from a dict
io_argoproj_events_v1alpha1_webhook_context_form_dict = io_argoproj_events_v1alpha1_webhook_context.from_dict(io_argoproj_events_v1alpha1_webhook_context_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


