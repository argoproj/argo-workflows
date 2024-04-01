# IoArgoprojEventsV1alpha1Service


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**cluster_ip** | **str** |  | [optional] 
**ports** | [**List[ServicePort]**](ServicePort.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_service import IoArgoprojEventsV1alpha1Service

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1Service from a JSON string
io_argoproj_events_v1alpha1_service_instance = IoArgoprojEventsV1alpha1Service.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1Service.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_service_dict = io_argoproj_events_v1alpha1_service_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1Service from a dict
io_argoproj_events_v1alpha1_service_form_dict = io_argoproj_events_v1alpha1_service.from_dict(io_argoproj_events_v1alpha1_service_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


