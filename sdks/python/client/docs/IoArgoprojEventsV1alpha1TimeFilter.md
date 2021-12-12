# IoArgoprojEventsV1alpha1TimeFilter

TimeFilter describes a window in time. It filters out events that occur outside the time limits. In other words, only events that occur after Start and before Stop will pass this filter.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**start** | **str** | Start is the beginning of a time window in UTC. Before this time, events for this dependency are ignored. Format is hh:mm:ss. | [optional] 
**stop** | **str** | Stop is the end of a time window in UTC. After or equal to this time, events for this dependency are ignored and Format is hh:mm:ss. If it is smaller than Start, it is treated as next day of Start (e.g.: 22:00:00-01:00:00 means 22:00:00-25:00:00). | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


