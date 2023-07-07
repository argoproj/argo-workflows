# IoArgoprojEventsV1alpha1SensorSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**dependencies** | Option<[**Vec<crate::models::IoArgoprojEventsV1alpha1EventDependency>**](io.argoproj.events.v1alpha1.EventDependency.md)> | Dependencies is a list of the events that this sensor is dependent on. | [optional]
**error_on_failed_round** | Option<**bool**> | ErrorOnFailedRound if set to true, marks sensor state as `error` if the previous trigger round fails. Once sensor state is set to `error`, no further triggers will be processed. | [optional]
**event_bus_name** | Option<**String**> |  | [optional]
**replicas** | Option<**i32**> |  | [optional]
**template** | Option<[**crate::models::IoArgoprojEventsV1alpha1Template**](io.argoproj.events.v1alpha1.Template.md)> |  | [optional]
**triggers** | Option<[**Vec<crate::models::IoArgoprojEventsV1alpha1Trigger>**](io.argoproj.events.v1alpha1.Trigger.md)> | Triggers is a list of the things that this sensor evokes. These are the outputs from this sensor. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


