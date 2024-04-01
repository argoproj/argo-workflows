# IoArgoprojWorkflowV1alpha1ExecutorConfig

ExecutorConfig holds configurations of an executor container.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**service_account_name** | **str** | ServiceAccountName specifies the service account name of the executor container. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_executor_config import IoArgoprojWorkflowV1alpha1ExecutorConfig

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1ExecutorConfig from a JSON string
io_argoproj_workflow_v1alpha1_executor_config_instance = IoArgoprojWorkflowV1alpha1ExecutorConfig.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1ExecutorConfig.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_executor_config_dict = io_argoproj_workflow_v1alpha1_executor_config_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1ExecutorConfig from a dict
io_argoproj_workflow_v1alpha1_executor_config_form_dict = io_argoproj_workflow_v1alpha1_executor_config.from_dict(io_argoproj_workflow_v1alpha1_executor_config_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


