# IoArgoprojEventsV1alpha1EventDependency

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**event_name** | Option<**String**> |  | [optional]
**event_source_name** | Option<**String**> |  | [optional]
**filters** | Option<[**crate::models::IoArgoprojEventsV1alpha1EventDependencyFilter**](io.argoproj.events.v1alpha1.EventDependencyFilter.md)> |  | [optional]
**filters_logical_operator** | Option<**String**> | FiltersLogicalOperator defines how different filters are evaluated together. Available values: and (&&), or (||) Is optional and if left blank treated as and (&&). | [optional]
**name** | Option<**String**> |  | [optional]
**transform** | Option<[**crate::models::IoArgoprojEventsV1alpha1EventDependencyTransformer**](io.argoproj.events.v1alpha1.EventDependencyTransformer.md)> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


