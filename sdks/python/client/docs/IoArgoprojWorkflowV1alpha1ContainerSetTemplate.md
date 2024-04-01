# IoArgoprojWorkflowV1alpha1ContainerSetTemplate


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**containers** | [**List[IoArgoprojWorkflowV1alpha1ContainerNode]**](IoArgoprojWorkflowV1alpha1ContainerNode.md) |  | 
**retry_strategy** | [**IoArgoprojWorkflowV1alpha1ContainerSetRetryStrategy**](IoArgoprojWorkflowV1alpha1ContainerSetRetryStrategy.md) |  | [optional] 
**volume_mounts** | [**List[VolumeMount]**](VolumeMount.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_container_set_template import IoArgoprojWorkflowV1alpha1ContainerSetTemplate

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1ContainerSetTemplate from a JSON string
io_argoproj_workflow_v1alpha1_container_set_template_instance = IoArgoprojWorkflowV1alpha1ContainerSetTemplate.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1ContainerSetTemplate.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_container_set_template_dict = io_argoproj_workflow_v1alpha1_container_set_template_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1ContainerSetTemplate from a dict
io_argoproj_workflow_v1alpha1_container_set_template_form_dict = io_argoproj_workflow_v1alpha1_container_set_template.from_dict(io_argoproj_workflow_v1alpha1_container_set_template_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


