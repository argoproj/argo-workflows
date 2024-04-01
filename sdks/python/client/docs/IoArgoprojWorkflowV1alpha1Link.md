# IoArgoprojWorkflowV1alpha1Link

A link to another app.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | The name of the link, E.g. \&quot;Workflow Logs\&quot; or \&quot;Pod Logs\&quot; | 
**scope** | **str** | \&quot;workflow\&quot;, \&quot;pod\&quot;, \&quot;pod-logs\&quot;, \&quot;event-source-logs\&quot;, \&quot;sensor-logs\&quot;, \&quot;workflow-list\&quot; or \&quot;chat\&quot; | 
**url** | **str** | The URL. Can contain \&quot;${metadata.namespace}\&quot;, \&quot;${metadata.name}\&quot;, \&quot;${status.startedAt}\&quot;, \&quot;${status.finishedAt}\&quot; or any other element in workflow yaml, e.g. \&quot;${io.argoproj.workflow.v1alpha1.metadata.annotations.userDefinedKey}\&quot; | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_link import IoArgoprojWorkflowV1alpha1Link

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Link from a JSON string
io_argoproj_workflow_v1alpha1_link_instance = IoArgoprojWorkflowV1alpha1Link.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Link.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_link_dict = io_argoproj_workflow_v1alpha1_link_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Link from a dict
io_argoproj_workflow_v1alpha1_link_form_dict = io_argoproj_workflow_v1alpha1_link.from_dict(io_argoproj_workflow_v1alpha1_link_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


