# IoArgoprojEventsV1alpha1URLArtifact

URLArtifact contains information about an artifact at an http endpoint.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**path** | **str** |  | [optional] 
**verify_cert** | **bool** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_url_artifact import IoArgoprojEventsV1alpha1URLArtifact

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1URLArtifact from a JSON string
io_argoproj_events_v1alpha1_url_artifact_instance = IoArgoprojEventsV1alpha1URLArtifact.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1URLArtifact.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_url_artifact_dict = io_argoproj_events_v1alpha1_url_artifact_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1URLArtifact from a dict
io_argoproj_events_v1alpha1_url_artifact_form_dict = io_argoproj_events_v1alpha1_url_artifact.from_dict(io_argoproj_events_v1alpha1_url_artifact_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


