# IoArgoprojWorkflowV1alpha1ArchiveStrategy

ArchiveStrategy describes how to archive files/directory when saving artifacts

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**var_none** | **object** | NoneStrategy indicates to skip tar process and upload the files or directory tree as independent files. Note that if the artifact is a directory, the artifact driver must support the ability to save/load the directory appropriately. | [optional] 
**tar** | [**IoArgoprojWorkflowV1alpha1TarStrategy**](IoArgoprojWorkflowV1alpha1TarStrategy.md) |  | [optional] 
**zip** | **object** | ZipStrategy will unzip zipped input artifacts | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_archive_strategy import IoArgoprojWorkflowV1alpha1ArchiveStrategy

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1ArchiveStrategy from a JSON string
io_argoproj_workflow_v1alpha1_archive_strategy_instance = IoArgoprojWorkflowV1alpha1ArchiveStrategy.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1ArchiveStrategy.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_archive_strategy_dict = io_argoproj_workflow_v1alpha1_archive_strategy_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1ArchiveStrategy from a dict
io_argoproj_workflow_v1alpha1_archive_strategy_form_dict = io_argoproj_workflow_v1alpha1_archive_strategy.from_dict(io_argoproj_workflow_v1alpha1_archive_strategy_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


