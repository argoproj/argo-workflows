# IoArgoprojEventsV1alpha1ResourceEventSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**event_types** | Option<**Vec<String>**> | EventTypes is the list of event type to watch. Possible values are - ADD, UPDATE and DELETE. | [optional]
**filter** | Option<[**crate::models::IoArgoprojEventsV1alpha1ResourceFilter**](io.argoproj.events.v1alpha1.ResourceFilter.md)> |  | [optional]
**group_version_resource** | Option<[**crate::models::GroupVersionResource**](GroupVersionResource.md)> |  | [optional]
**metadata** | Option<**::std::collections::HashMap<String, String>**> |  | [optional]
**namespace** | Option<**String**> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


