# IoArgoprojWorkflowV1alpha1VolumeClaimGC

VolumeClaimGC describes how to delete volumes from completed Workflows

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**strategy** | **str** | Strategy is the strategy to use. One of \&quot;OnWorkflowCompletion\&quot;, \&quot;OnWorkflowSuccess\&quot;. Defaults to \&quot;OnWorkflowSuccess\&quot; | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_volume_claim_gc import IoArgoprojWorkflowV1alpha1VolumeClaimGC

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1VolumeClaimGC from a JSON string
io_argoproj_workflow_v1alpha1_volume_claim_gc_instance = IoArgoprojWorkflowV1alpha1VolumeClaimGC.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1VolumeClaimGC.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_volume_claim_gc_dict = io_argoproj_workflow_v1alpha1_volume_claim_gc_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1VolumeClaimGC from a dict
io_argoproj_workflow_v1alpha1_volume_claim_gc_form_dict = io_argoproj_workflow_v1alpha1_volume_claim_gc.from_dict(io_argoproj_workflow_v1alpha1_volume_claim_gc_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


