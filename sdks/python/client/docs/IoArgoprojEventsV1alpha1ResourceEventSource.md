# IoArgoprojEventsV1alpha1ResourceEventSource

ResourceEventSource refers to a event-source for K8s resource related events.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**event_types** | **list[str]** | EventTypes is the list of event type to watch. Possible values are - ADD, UPDATE and DELETE. | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1ResourceFilter**](IoArgoprojEventsV1alpha1ResourceFilter.md) |  | [optional] 
**group_version_resource** | [**GroupVersionResource**](GroupVersionResource.md) |  | [optional] 
**metadata** | **dict(str, str)** |  | [optional] 
**namespace** | **str** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


