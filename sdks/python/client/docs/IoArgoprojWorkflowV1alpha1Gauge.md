# IoArgoprojWorkflowV1alpha1Gauge

Gauge is a Gauge prometheus metric

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**operation** | **str** | Operation defines the operation to apply with value and the metrics&#39; current value | [optional] 
**realtime** | **bool** | Realtime emits this metric in real time if applicable | 
**value** | **str** | Value is the value to be used in the operation with the metric&#39;s current value. If no operation is set, value is the value of the metric | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_gauge import IoArgoprojWorkflowV1alpha1Gauge

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Gauge from a JSON string
io_argoproj_workflow_v1alpha1_gauge_instance = IoArgoprojWorkflowV1alpha1Gauge.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Gauge.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_gauge_dict = io_argoproj_workflow_v1alpha1_gauge_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Gauge from a dict
io_argoproj_workflow_v1alpha1_gauge_form_dict = io_argoproj_workflow_v1alpha1_gauge.from_dict(io_argoproj_workflow_v1alpha1_gauge_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


