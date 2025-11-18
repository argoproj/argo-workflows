# IoArgoprojWorkflowV1alpha1Gauge

Gauge is a Gauge prometheus metric

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**realtime** | **bool** | Realtime emits this metric in real time if applicable | 
**value** | **str** | Value is the value to be used in the operation with the metric&#39;s current value. If no operation is set, value is the value of the metric MaxLength is an artificial limit to limit CEL validation costs - see note at top of file | 
**operation** | **str** | Operation defines the operation to apply with value and the metrics&#39; current value | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


