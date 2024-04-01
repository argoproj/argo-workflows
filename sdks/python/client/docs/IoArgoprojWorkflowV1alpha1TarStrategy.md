# IoArgoprojWorkflowV1alpha1TarStrategy

TarStrategy will tar and gzip the file or directory when saving

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**compression_level** | **int** | CompressionLevel specifies the gzip compression level to use for the artifact. Defaults to gzip.DefaultCompression. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_tar_strategy import IoArgoprojWorkflowV1alpha1TarStrategy

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1TarStrategy from a JSON string
io_argoproj_workflow_v1alpha1_tar_strategy_instance = IoArgoprojWorkflowV1alpha1TarStrategy.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1TarStrategy.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_tar_strategy_dict = io_argoproj_workflow_v1alpha1_tar_strategy_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1TarStrategy from a dict
io_argoproj_workflow_v1alpha1_tar_strategy_form_dict = io_argoproj_workflow_v1alpha1_tar_strategy.from_dict(io_argoproj_workflow_v1alpha1_tar_strategy_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


