# Event

Event is a report of an event somewhere in the cluster.  Events have a limited retention time and triggers and messages may evolve with time.  Event consumers should not rely on the timing of an event with a given Reason reflecting a consistent underlying trigger, or the continued existence of events with that Reason.  Events should be treated as informative, best-effort, supplemental data.

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**involved_object** | [**ObjectReference**](ObjectReference.md) |  | 
**metadata** | [**ObjectMeta**](ObjectMeta.md) |  | 
**action** | **str** | What action was taken/failed regarding to the Regarding object. | [optional] 
**api_version** | **str** | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources | [optional] 
**count** | **int** | The number of times this event has occurred. | [optional] 
**event_time** | **datetime** | MicroTime is version of Time with microsecond level precision. | [optional] 
**first_timestamp** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional] 
**kind** | **str** | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds | [optional] 
**last_timestamp** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional] 
**message** | **str** | A human-readable description of the status of this operation. | [optional] 
**reason** | **str** | This should be a short, machine understandable string that gives the reason for the transition into the object&#39;s current status. | [optional] 
**related** | [**ObjectReference**](ObjectReference.md) |  | [optional] 
**reporting_component** | **str** | Name of the controller that emitted this Event, e.g. &#x60;kubernetes.io/kubelet&#x60;. | [optional] 
**reporting_instance** | **str** | ID of the controller instance, e.g. &#x60;kubelet-xyzf&#x60;. | [optional] 
**series** | [**EventSeries**](EventSeries.md) |  | [optional] 
**source** | [**EventSource**](EventSource.md) |  | [optional] 
**type** | **str** | Type of this event (Normal, Warning), new types could be added in the future | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


