# V1alpha1Prometheus

Prometheus is a prometheus metric to be emitted
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**counter** | [**V1alpha1Counter**](V1alpha1Counter.md) |  | [optional] 
**gauge** | [**V1alpha1Gauge**](V1alpha1Gauge.md) |  | [optional] 
**help** | **str** | Help is a string that describes the metric | 
**histogram** | [**V1alpha1Histogram**](V1alpha1Histogram.md) |  | [optional] 
**labels** | [**list[V1alpha1MetricLabel]**](V1alpha1MetricLabel.md) | Labels is a list of metric labels | [optional] 
**name** | **str** | Name is the name of the metric | 
**when** | **str** | When is a conditional statement that decides when to emit the metric | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


