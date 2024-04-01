# IoArgoprojWorkflowV1alpha1SubmitOpts

SubmitOpts are workflow submission options

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**annotations** | **str** | Annotations adds to metadata.labels | [optional] 
**dry_run** | **bool** | DryRun validates the workflow on the client-side without creating it. This option is not supported in API | [optional] 
**entry_point** | **str** | Entrypoint overrides spec.entrypoint | [optional] 
**generate_name** | **str** | GenerateName overrides metadata.generateName | [optional] 
**labels** | **str** | Labels adds to metadata.labels | [optional] 
**name** | **str** | Name overrides metadata.name | [optional] 
**owner_reference** | [**OwnerReference**](OwnerReference.md) |  | [optional] 
**parameters** | **List[str]** | Parameters passes input parameters to workflow | [optional] 
**pod_priority_class_name** | **str** | Set the podPriorityClassName of the workflow | [optional] 
**priority** | **int** | Priority is used if controller is configured to process limited number of workflows in parallel, higher priority workflows are processed first. | [optional] 
**server_dry_run** | **bool** | ServerDryRun validates the workflow on the server-side without creating it | [optional] 
**service_account** | **str** | ServiceAccount runs all pods in the workflow using specified ServiceAccount. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_submit_opts import IoArgoprojWorkflowV1alpha1SubmitOpts

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1SubmitOpts from a JSON string
io_argoproj_workflow_v1alpha1_submit_opts_instance = IoArgoprojWorkflowV1alpha1SubmitOpts.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1SubmitOpts.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_submit_opts_dict = io_argoproj_workflow_v1alpha1_submit_opts_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1SubmitOpts from a dict
io_argoproj_workflow_v1alpha1_submit_opts_form_dict = io_argoproj_workflow_v1alpha1_submit_opts.from_dict(io_argoproj_workflow_v1alpha1_submit_opts_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


