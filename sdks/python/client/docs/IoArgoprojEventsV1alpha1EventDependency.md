# IoArgoprojEventsV1alpha1EventDependency


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**event_name** | **str** |  | [optional] 
**event_source_name** | **str** |  | [optional] 
**filters** | [**IoArgoprojEventsV1alpha1EventDependencyFilter**](IoArgoprojEventsV1alpha1EventDependencyFilter.md) |  | [optional] 
**filters_logical_operator** | **str** | FiltersLogicalOperator defines how different filters are evaluated together. Available values: and (&amp;&amp;), or (||) Is optional and if left blank treated as and (&amp;&amp;). | [optional] 
**name** | **str** |  | [optional] 
**transform** | [**IoArgoprojEventsV1alpha1EventDependencyTransformer**](IoArgoprojEventsV1alpha1EventDependencyTransformer.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


