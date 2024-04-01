# IoArgoprojWorkflowV1alpha1DAGTemplate

DAGTemplate is a template subtype for directed acyclic graph templates

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fail_fast** | **bool** | This flag is for DAG logic. The DAG logic has a built-in \&quot;fail fast\&quot; feature to stop scheduling new steps, as soon as it detects that one of the DAG nodes is failed. Then it waits until all DAG nodes are completed before failing the DAG itself. The FailFast flag default is true,  if set to false, it will allow a DAG to run all branches of the DAG to completion (either success or failure), regardless of the failed outcomes of branches in the DAG. More info and example about this feature at https://github.com/argoproj/argo-workflows/issues/1442 | [optional] 
**target** | **str** | Target are one or more names of targets to execute in a DAG | [optional] 
**tasks** | [**List[IoArgoprojWorkflowV1alpha1DAGTask]**](IoArgoprojWorkflowV1alpha1DAGTask.md) | Tasks are a list of DAG tasks | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_dag_template import IoArgoprojWorkflowV1alpha1DAGTemplate

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1DAGTemplate from a JSON string
io_argoproj_workflow_v1alpha1_dag_template_instance = IoArgoprojWorkflowV1alpha1DAGTemplate.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1DAGTemplate.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_dag_template_dict = io_argoproj_workflow_v1alpha1_dag_template_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1DAGTemplate from a dict
io_argoproj_workflow_v1alpha1_dag_template_form_dict = io_argoproj_workflow_v1alpha1_dag_template.from_dict(io_argoproj_workflow_v1alpha1_dag_template_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


