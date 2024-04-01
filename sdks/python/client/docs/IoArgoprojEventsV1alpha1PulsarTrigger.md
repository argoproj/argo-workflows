# IoArgoprojEventsV1alpha1PulsarTrigger

PulsarTrigger refers to the specification of the Pulsar trigger.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth_athenz_params** | **Dict[str, str]** |  | [optional] 
**auth_athenz_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**auth_token_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**connection_backoff** | [**IoArgoprojEventsV1alpha1Backoff**](IoArgoprojEventsV1alpha1Backoff.md) |  | [optional] 
**parameters** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) | Parameters is the list of parameters that is applied to resolved Kafka trigger object. | [optional] 
**payload** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) | Payload is the list of key-value extracted from an event payload to construct the request payload. | [optional] 
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  | [optional] 
**tls_allow_insecure_connection** | **bool** |  | [optional] 
**tls_trust_certs_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**tls_validate_hostname** | **bool** |  | [optional] 
**topic** | **str** |  | [optional] 
**url** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_pulsar_trigger import IoArgoprojEventsV1alpha1PulsarTrigger

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1PulsarTrigger from a JSON string
io_argoproj_events_v1alpha1_pulsar_trigger_instance = IoArgoprojEventsV1alpha1PulsarTrigger.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1PulsarTrigger.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_pulsar_trigger_dict = io_argoproj_events_v1alpha1_pulsar_trigger_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1PulsarTrigger from a dict
io_argoproj_events_v1alpha1_pulsar_trigger_form_dict = io_argoproj_events_v1alpha1_pulsar_trigger.from_dict(io_argoproj_events_v1alpha1_pulsar_trigger_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


