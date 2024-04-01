# IoArgoprojEventsV1alpha1S3Artifact


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**access_key** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**bucket** | [**IoArgoprojEventsV1alpha1S3Bucket**](IoArgoprojEventsV1alpha1S3Bucket.md) |  | [optional] 
**ca_certificate** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**endpoint** | **str** |  | [optional] 
**events** | **List[str]** |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1S3Filter**](IoArgoprojEventsV1alpha1S3Filter.md) |  | [optional] 
**insecure** | **bool** |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**region** | **str** |  | [optional] 
**secret_key** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_s3_artifact import IoArgoprojEventsV1alpha1S3Artifact

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1S3Artifact from a JSON string
io_argoproj_events_v1alpha1_s3_artifact_instance = IoArgoprojEventsV1alpha1S3Artifact.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1S3Artifact.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_s3_artifact_dict = io_argoproj_events_v1alpha1_s3_artifact_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1S3Artifact from a dict
io_argoproj_events_v1alpha1_s3_artifact_form_dict = io_argoproj_events_v1alpha1_s3_artifact.from_dict(io_argoproj_events_v1alpha1_s3_artifact_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


