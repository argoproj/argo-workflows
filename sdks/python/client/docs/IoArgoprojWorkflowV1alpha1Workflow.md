# IoArgoprojWorkflowV1alpha1Workflow

Workflow is the definition of a workflow resource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_version** | **str** | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#resources | [optional] 
**kind** | **str** | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#types-kinds | [optional] 
**metadata** | [**ObjectMeta**](ObjectMeta.md) |  | 
**spec** | [**IoArgoprojWorkflowV1alpha1WorkflowSpec**](IoArgoprojWorkflowV1alpha1WorkflowSpec.md) |  | 
**status** | [**IoArgoprojWorkflowV1alpha1WorkflowStatus**](IoArgoprojWorkflowV1alpha1WorkflowStatus.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_workflow import IoArgoprojWorkflowV1alpha1Workflow

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Workflow from a JSON string
io_argoproj_workflow_v1alpha1_workflow_instance = IoArgoprojWorkflowV1alpha1Workflow.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Workflow.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_workflow_dict = io_argoproj_workflow_v1alpha1_workflow_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Workflow from a dict
io_argoproj_workflow_v1alpha1_workflow_form_dict = io_argoproj_workflow_v1alpha1_workflow.from_dict(io_argoproj_workflow_v1alpha1_workflow_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


