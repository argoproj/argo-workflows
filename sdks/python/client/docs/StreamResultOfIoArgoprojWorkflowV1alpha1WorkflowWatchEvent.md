# StreamResultOfIoArgoprojWorkflowV1alpha1WorkflowWatchEvent


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**error** | [**GrpcGatewayRuntimeStreamError**](GrpcGatewayRuntimeStreamError.md) |  | [optional] 
**result** | [**IoArgoprojWorkflowV1alpha1WorkflowWatchEvent**](IoArgoprojWorkflowV1alpha1WorkflowWatchEvent.md) |  | [optional] 

## Example

```python
from argo_workflows.models.stream_result_of_io_argoproj_workflow_v1alpha1_workflow_watch_event import StreamResultOfIoArgoprojWorkflowV1alpha1WorkflowWatchEvent

# TODO update the JSON string below
json = "{}"
# create an instance of StreamResultOfIoArgoprojWorkflowV1alpha1WorkflowWatchEvent from a JSON string
stream_result_of_io_argoproj_workflow_v1alpha1_workflow_watch_event_instance = StreamResultOfIoArgoprojWorkflowV1alpha1WorkflowWatchEvent.from_json(json)
# print the JSON string representation of the object
print(StreamResultOfIoArgoprojWorkflowV1alpha1WorkflowWatchEvent.to_json())

# convert the object into a dict
stream_result_of_io_argoproj_workflow_v1alpha1_workflow_watch_event_dict = stream_result_of_io_argoproj_workflow_v1alpha1_workflow_watch_event_instance.to_dict()
# create an instance of StreamResultOfIoArgoprojWorkflowV1alpha1WorkflowWatchEvent from a dict
stream_result_of_io_argoproj_workflow_v1alpha1_workflow_watch_event_form_dict = stream_result_of_io_argoproj_workflow_v1alpha1_workflow_watch_event.from_dict(stream_result_of_io_argoproj_workflow_v1alpha1_workflow_watch_event_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


