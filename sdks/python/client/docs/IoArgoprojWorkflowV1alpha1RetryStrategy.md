# IoArgoprojWorkflowV1alpha1RetryStrategy

RetryStrategy provides controls on how to retry a workflow step

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**affinity** | [**IoArgoprojWorkflowV1alpha1RetryAffinity**](IoArgoprojWorkflowV1alpha1RetryAffinity.md) |  | [optional] 
**backoff** | [**IoArgoprojWorkflowV1alpha1Backoff**](IoArgoprojWorkflowV1alpha1Backoff.md) |  | [optional] 
**expression** | **str** | Expression is a condition expression for when a node will be retried. If it evaluates to false, the node will not be retried and the retry strategy will be ignored | [optional] 
**limit** | **str** |  | [optional] 
**retry_policy** | **str** | RetryPolicy is a policy of NodePhase statuses that will be retried | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_retry_strategy import IoArgoprojWorkflowV1alpha1RetryStrategy

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1RetryStrategy from a JSON string
io_argoproj_workflow_v1alpha1_retry_strategy_instance = IoArgoprojWorkflowV1alpha1RetryStrategy.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1RetryStrategy.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_retry_strategy_dict = io_argoproj_workflow_v1alpha1_retry_strategy_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1RetryStrategy from a dict
io_argoproj_workflow_v1alpha1_retry_strategy_form_dict = io_argoproj_workflow_v1alpha1_retry_strategy.from_dict(io_argoproj_workflow_v1alpha1_retry_strategy_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


