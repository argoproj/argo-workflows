# IoArgoprojWorkflowV1alpha1Prometheus

Prometheus is a prometheus metric to be emitted

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**counter** | [**IoArgoprojWorkflowV1alpha1Counter**](IoArgoprojWorkflowV1alpha1Counter.md) |  | [optional] 
**gauge** | [**IoArgoprojWorkflowV1alpha1Gauge**](IoArgoprojWorkflowV1alpha1Gauge.md) |  | [optional] 
**help** | **str** | Help is a string that describes the metric | 
**histogram** | [**IoArgoprojWorkflowV1alpha1Histogram**](IoArgoprojWorkflowV1alpha1Histogram.md) |  | [optional] 
**labels** | [**List[IoArgoprojWorkflowV1alpha1MetricLabel]**](IoArgoprojWorkflowV1alpha1MetricLabel.md) | Labels is a list of metric labels | [optional] 
**name** | **str** | Name is the name of the metric | 
**when** | **str** | When is a conditional statement that decides when to emit the metric | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_prometheus import IoArgoprojWorkflowV1alpha1Prometheus

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Prometheus from a JSON string
io_argoproj_workflow_v1alpha1_prometheus_instance = IoArgoprojWorkflowV1alpha1Prometheus.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Prometheus.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_prometheus_dict = io_argoproj_workflow_v1alpha1_prometheus_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Prometheus from a dict
io_argoproj_workflow_v1alpha1_prometheus_form_dict = io_argoproj_workflow_v1alpha1_prometheus.from_dict(io_argoproj_workflow_v1alpha1_prometheus_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


