

# Event

Event is a report of an event somewhere in the cluster.  Events have a limited retention time and triggers and messages may evolve with time.  Event consumers should not rely on the timing of an event with a given Reason reflecting a consistent underlying trigger, or the continued existence of events with that Reason.  Events should be treated as informative, best-effort, supplemental data.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**action** | **String** | What action was taken/failed regarding to the Regarding object. |  [optional]
**apiVersion** | **String** | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  [optional]
**count** | **Integer** | The number of times this event has occurred. |  [optional]
**eventTime** | **OffsetDateTime** | MicroTime is version of Time with microsecond level precision. |  [optional]
**firstTimestamp** | **java.time.Instant** |  |  [optional]
**involvedObject** | [**io.kubernetes.client.openapi.models.V1ObjectReference**](io.kubernetes.client.openapi.models.V1ObjectReference.md) |  | 
**kind** | **String** | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  [optional]
**lastTimestamp** | **java.time.Instant** |  |  [optional]
**message** | **String** | A human-readable description of the status of this operation. |  [optional]
**metadata** | [**io.kubernetes.client.openapi.models.V1ObjectMeta**](io.kubernetes.client.openapi.models.V1ObjectMeta.md) |  | 
**reason** | **String** | This should be a short, machine understandable string that gives the reason for the transition into the object&#39;s current status. |  [optional]
**related** | [**io.kubernetes.client.openapi.models.V1ObjectReference**](io.kubernetes.client.openapi.models.V1ObjectReference.md) |  |  [optional]
**reportingComponent** | **String** | Name of the controller that emitted this Event, e.g. &#x60;kubernetes.io/kubelet&#x60;. |  [optional]
**reportingInstance** | **String** | ID of the controller instance, e.g. &#x60;kubelet-xyzf&#x60;. |  [optional]
**series** | [**EventSeries**](EventSeries.md) |  |  [optional]
**source** | [**EventSource**](EventSource.md) |  |  [optional]
**type** | **String** | Type of this event (Normal, Warning), new types could be added in the future |  [optional]



