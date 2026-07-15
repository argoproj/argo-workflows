

# IoArgoprojWorkflowV1alpha1Gauge

Gauge is a Gauge prometheus metric

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**operation** | **String** | Operation defines the operation to apply with value and the metrics&#39; current value |  [optional]
**realtime** | **Boolean** | Realtime emits this metric in real time if applicable | 
**value** | **String** | Value is the value to be used in the operation with the metric&#39;s current value. If no operation is set, value is the value of the metric MaxLength is an artificial limit to limit CEL validation costs - see note at top of file | 



