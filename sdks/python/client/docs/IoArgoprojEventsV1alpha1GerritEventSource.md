# IoArgoprojEventsV1alpha1GerritEventSource


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth** | [**IoArgoprojEventsV1alpha1BasicAuth**](IoArgoprojEventsV1alpha1BasicAuth.md) |  | [optional] 
**delete_hook_on_finish** | **bool** |  | [optional] 
**events** | **[str]** |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**gerrit_base_url** | **str** |  | [optional] 
**hook_name** | **str** |  | [optional] 
**metadata** | **{str: (str,)}** |  | [optional] 
**projects** | **[str]** | List of project namespace paths like \&quot;whynowy/test\&quot;. | [optional] 
**ssl_verify** | **bool** |  | [optional] 
**webhook** | [**IoArgoprojEventsV1alpha1WebhookContext**](IoArgoprojEventsV1alpha1WebhookContext.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


