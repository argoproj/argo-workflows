# IoArgoprojEventsV1alpha1GitArtifact

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**branch** | **str** |  | [optional] 
**clone_directory** | **str** | Directory to clone the repository. We clone complete directory because GitArtifact is not limited to any specific Git service providers. Hence we don&#39;t use any specific git provider client. | [optional] 
**creds** | [**IoArgoprojEventsV1alpha1GitCreds**](IoArgoprojEventsV1alpha1GitCreds.md) |  | [optional] 
**file_path** | **str** |  | [optional] 
**ref** | **str** |  | [optional] 
**remote** | [**IoArgoprojEventsV1alpha1GitRemoteConfig**](IoArgoprojEventsV1alpha1GitRemoteConfig.md) |  | [optional] 
**ssh_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**tag** | **str** |  | [optional] 
**url** | **str** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


