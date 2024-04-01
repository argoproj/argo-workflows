# IoArgoprojWorkflowV1alpha1MetricLabel

MetricLabel is a single label for a prometheus metric

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**key** | **str** |  | 
**value** | **str** |  | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_metric_label import IoArgoprojWorkflowV1alpha1MetricLabel

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1MetricLabel from a JSON string
io_argoproj_workflow_v1alpha1_metric_label_instance = IoArgoprojWorkflowV1alpha1MetricLabel.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1MetricLabel.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_metric_label_dict = io_argoproj_workflow_v1alpha1_metric_label_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1MetricLabel from a dict
io_argoproj_workflow_v1alpha1_metric_label_form_dict = io_argoproj_workflow_v1alpha1_metric_label.from_dict(io_argoproj_workflow_v1alpha1_metric_label_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


