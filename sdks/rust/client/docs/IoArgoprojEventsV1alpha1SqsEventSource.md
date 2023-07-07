# IoArgoprojEventsV1alpha1SqsEventSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**access_key** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**dlq** | Option<**bool**> |  | [optional]
**endpoint** | Option<**String**> |  | [optional]
**filter** | Option<[**crate::models::IoArgoprojEventsV1alpha1EventSourceFilter**](io.argoproj.events.v1alpha1.EventSourceFilter.md)> |  | [optional]
**json_body** | Option<**bool**> |  | [optional]
**metadata** | Option<**::std::collections::HashMap<String, String>**> |  | [optional]
**queue** | Option<**String**> |  | [optional]
**queue_account_id** | Option<**String**> |  | [optional]
**region** | Option<**String**> |  | [optional]
**role_arn** | Option<**String**> |  | [optional]
**secret_key** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**session_token** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**wait_time_seconds** | Option<**String**> | WaitTimeSeconds is The duration (in seconds) for which the call waits for a message to arrive in the queue before returning. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


