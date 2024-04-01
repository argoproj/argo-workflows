# IoArgoprojWorkflowV1alpha1Metrics

Metrics are a list of metrics emitted from a Workflow/Template

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**prometheus** | [**List[IoArgoprojWorkflowV1alpha1Prometheus]**](IoArgoprojWorkflowV1alpha1Prometheus.md) | Prometheus is a list of prometheus metrics to be emitted | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_metrics import IoArgoprojWorkflowV1alpha1Metrics

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Metrics from a JSON string
io_argoproj_workflow_v1alpha1_metrics_instance = IoArgoprojWorkflowV1alpha1Metrics.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Metrics.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_metrics_dict = io_argoproj_workflow_v1alpha1_metrics_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Metrics from a dict
io_argoproj_workflow_v1alpha1_metrics_form_dict = io_argoproj_workflow_v1alpha1_metrics.from_dict(io_argoproj_workflow_v1alpha1_metrics_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


