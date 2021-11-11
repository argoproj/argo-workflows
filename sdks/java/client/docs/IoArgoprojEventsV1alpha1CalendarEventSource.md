

# IoArgoprojEventsV1alpha1CalendarEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**exclusionDates** | **List&lt;String&gt;** |  |  [optional]
**interval** | **String** | Interval is a string that describes an interval duration, e.g. 1s, 30m, 2h... |  [optional]
**metadata** | **Map&lt;String, String&gt;** |  |  [optional]
**persistence** | [**IoArgoprojEventsV1alpha1EventPersistence**](IoArgoprojEventsV1alpha1EventPersistence.md) |  |  [optional]
**schedule** | **String** |  |  [optional]
**timezone** | **String** |  |  [optional]
**userPayload** | **byte[]** | UserPayload will be sent to sensor as extra data once the event is triggered +optional Deprecated: will be removed in v1.5. Please use Metadata instead. |  [optional]



