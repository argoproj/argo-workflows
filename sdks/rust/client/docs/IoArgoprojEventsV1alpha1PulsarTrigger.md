# IoArgoprojEventsV1alpha1PulsarTrigger

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth_token_secret** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**connection_backoff** | Option<[**crate::models::IoArgoprojEventsV1alpha1Backoff**](io.argoproj.events.v1alpha1.Backoff.md)> |  | [optional]
**parameters** | Option<[**Vec<crate::models::IoArgoprojEventsV1alpha1TriggerParameter>**](io.argoproj.events.v1alpha1.TriggerParameter.md)> | Parameters is the list of parameters that is applied to resolved Kafka trigger object. | [optional]
**payload** | Option<[**Vec<crate::models::IoArgoprojEventsV1alpha1TriggerParameter>**](io.argoproj.events.v1alpha1.TriggerParameter.md)> | Payload is the list of key-value extracted from an event payload to construct the request payload. | [optional]
**tls** | Option<[**crate::models::IoArgoprojEventsV1alpha1TlsConfig**](io.argoproj.events.v1alpha1.TLSConfig.md)> |  | [optional]
**tls_allow_insecure_connection** | Option<**bool**> |  | [optional]
**tls_trust_certs_secret** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**tls_validate_hostname** | Option<**bool**> |  | [optional]
**topic** | Option<**String**> |  | [optional]
**url** | Option<**String**> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


