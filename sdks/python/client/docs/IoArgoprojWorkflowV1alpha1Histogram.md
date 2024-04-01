# IoArgoprojWorkflowV1alpha1Histogram

Histogram is a Histogram prometheus metric

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**buckets** | **List[float]** | Buckets is a list of bucket divisors for the histogram | 
**value** | **str** | Value is the value of the metric | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_histogram import IoArgoprojWorkflowV1alpha1Histogram

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Histogram from a JSON string
io_argoproj_workflow_v1alpha1_histogram_instance = IoArgoprojWorkflowV1alpha1Histogram.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Histogram.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_histogram_dict = io_argoproj_workflow_v1alpha1_histogram_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Histogram from a dict
io_argoproj_workflow_v1alpha1_histogram_form_dict = io_argoproj_workflow_v1alpha1_histogram.from_dict(io_argoproj_workflow_v1alpha1_histogram_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


