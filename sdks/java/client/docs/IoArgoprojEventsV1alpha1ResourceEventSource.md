

# IoArgoprojEventsV1alpha1ResourceEventSource

ResourceEventSource refers to a event-source for K8s resource related events.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**eventTypes** | **List&lt;String&gt;** | EventTypes is the list of event type to watch. Possible values are - ADD, UPDATE and DELETE. |  [optional]
**filter** | [**IoArgoprojEventsV1alpha1ResourceFilter**](IoArgoprojEventsV1alpha1ResourceFilter.md) |  |  [optional]
**groupVersionResource** | [**GroupVersionResource**](GroupVersionResource.md) |  |  [optional]
**metadata** | **Map&lt;String, String&gt;** |  |  [optional]
**namespace** | **String** |  |  [optional]



