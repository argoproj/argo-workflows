# IoArgoprojEventsV1alpha1PulsarEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth_athenz_params** | **Dict[str, str]** |  | [optional] 
**auth_athenz_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**auth_token_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**connection_backoff** | [**IoArgoprojEventsV1alpha1Backoff**](IoArgoprojEventsV1alpha1Backoff.md) |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**json_body** | **bool** |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  | [optional] 
**tls_allow_insecure_connection** | **bool** |  | [optional] 
**tls_trust_certs_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**tls_validate_hostname** | **bool** |  | [optional] 
**topics** | **List[str]** |  | [optional] 
**type** | **str** |  | [optional] 
**url** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_pulsar_event_source import IoArgoprojEventsV1alpha1PulsarEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1PulsarEventSource from a JSON string
io_argoproj_events_v1alpha1_pulsar_event_source_instance = IoArgoprojEventsV1alpha1PulsarEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1PulsarEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_pulsar_event_source_dict = io_argoproj_events_v1alpha1_pulsar_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1PulsarEventSource from a dict
io_argoproj_events_v1alpha1_pulsar_event_source_form_dict = io_argoproj_events_v1alpha1_pulsar_event_source.from_dict(io_argoproj_events_v1alpha1_pulsar_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


