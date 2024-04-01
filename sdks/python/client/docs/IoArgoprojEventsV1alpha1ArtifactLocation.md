# IoArgoprojEventsV1alpha1ArtifactLocation


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**configmap** | [**ConfigMapKeySelector**](ConfigMapKeySelector.md) |  | [optional] 
**file** | [**IoArgoprojEventsV1alpha1FileArtifact**](IoArgoprojEventsV1alpha1FileArtifact.md) |  | [optional] 
**git** | [**IoArgoprojEventsV1alpha1GitArtifact**](IoArgoprojEventsV1alpha1GitArtifact.md) |  | [optional] 
**inline** | **str** |  | [optional] 
**resource** | [**IoArgoprojEventsV1alpha1Resource**](IoArgoprojEventsV1alpha1Resource.md) |  | [optional] 
**s3** | [**IoArgoprojEventsV1alpha1S3Artifact**](IoArgoprojEventsV1alpha1S3Artifact.md) |  | [optional] 
**url** | [**IoArgoprojEventsV1alpha1URLArtifact**](IoArgoprojEventsV1alpha1URLArtifact.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_artifact_location import IoArgoprojEventsV1alpha1ArtifactLocation

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1ArtifactLocation from a JSON string
io_argoproj_events_v1alpha1_artifact_location_instance = IoArgoprojEventsV1alpha1ArtifactLocation.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1ArtifactLocation.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_artifact_location_dict = io_argoproj_events_v1alpha1_artifact_location_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1ArtifactLocation from a dict
io_argoproj_events_v1alpha1_artifact_location_form_dict = io_argoproj_events_v1alpha1_artifact_location.from_dict(io_argoproj_events_v1alpha1_artifact_location_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


