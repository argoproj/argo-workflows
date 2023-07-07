# IoArgoprojEventsV1alpha1EventContext

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**datacontenttype** | Option<**String**> | DataContentType - A MIME (RFC2046) string describing the media type of `data`. | [optional]
**id** | Option<**String**> | ID of the event; must be non-empty and unique within the scope of the producer. | [optional]
**source** | Option<**String**> | Source - A URI describing the event producer. | [optional]
**specversion** | Option<**String**> | SpecVersion - The version of the CloudEvents specification used by the io.argoproj.workflow.v1alpha1. | [optional]
**subject** | Option<**String**> |  | [optional]
**time** | Option<**String**> | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional]
**_type** | Option<**String**> | Type - The type of the occurrence which has happened. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


