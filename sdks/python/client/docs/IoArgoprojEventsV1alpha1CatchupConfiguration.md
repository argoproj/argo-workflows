# IoArgoprojEventsV1alpha1CatchupConfiguration


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**enabled** | **bool** |  | [optional] 
**max_duration** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_catchup_configuration import IoArgoprojEventsV1alpha1CatchupConfiguration

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1CatchupConfiguration from a JSON string
io_argoproj_events_v1alpha1_catchup_configuration_instance = IoArgoprojEventsV1alpha1CatchupConfiguration.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1CatchupConfiguration.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_catchup_configuration_dict = io_argoproj_events_v1alpha1_catchup_configuration_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1CatchupConfiguration from a dict
io_argoproj_events_v1alpha1_catchup_configuration_form_dict = io_argoproj_events_v1alpha1_catchup_configuration.from_dict(io_argoproj_events_v1alpha1_catchup_configuration_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


