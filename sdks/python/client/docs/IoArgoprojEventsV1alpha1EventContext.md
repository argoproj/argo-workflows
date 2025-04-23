# IoArgoprojEventsV1alpha1EventContext


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**datacontenttype** | **str** | DataContentType - A MIME (RFC2046) string describing the media type of &#x60;data&#x60;. | [optional] 
**id** | **str** | ID of the event; must be non-empty and unique within the scope of the producer. | [optional] 
**source** | **str** | Source - A URI describing the event producer. | [optional] 
**specversion** | **str** | SpecVersion - The version of the CloudEvents specification used by the io.argoproj.workflow.v1alpha1. | [optional] 
**subject** | **str** |  | [optional] 
**time** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional] 
**type** | **str** | Type - The type of the occurrence which has happened. | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


