# IoArgoprojWorkflowV1alpha1OSSLifecycleRule

OSSLifecycleRule specifies how to manage bucket's lifecycle

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**mark_deletion_after_days** | **int** | MarkDeletionAfterDays is the number of days before we delete objects in the bucket | [optional] 
**mark_infrequent_access_after_days** | **int** | MarkInfrequentAccessAfterDays is the number of days before we convert the objects in the bucket to Infrequent Access (IA) storage type | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_oss_lifecycle_rule import IoArgoprojWorkflowV1alpha1OSSLifecycleRule

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1OSSLifecycleRule from a JSON string
io_argoproj_workflow_v1alpha1_oss_lifecycle_rule_instance = IoArgoprojWorkflowV1alpha1OSSLifecycleRule.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1OSSLifecycleRule.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_oss_lifecycle_rule_dict = io_argoproj_workflow_v1alpha1_oss_lifecycle_rule_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1OSSLifecycleRule from a dict
io_argoproj_workflow_v1alpha1_oss_lifecycle_rule_form_dict = io_argoproj_workflow_v1alpha1_oss_lifecycle_rule.from_dict(io_argoproj_workflow_v1alpha1_oss_lifecycle_rule_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


