# V1alpha1GitArtifact

GitArtifact is the location of an git artifact
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**depth** | **int** | Depth specifies clones/fetches should be shallow and include the given number of commits from the branch tip | [optional] 
**fetch** | **list[str]** | Fetch specifies a number of refs that should be fetched before checkout | [optional] 
**insecure_ignore_host_key** | **bool** | InsecureIgnoreHostKey disables SSH strict host key checking during git clone | [optional] 
**password_secret** | [**V1SecretKeySelector**](V1SecretKeySelector.md) |  | [optional] 
**repo** | **str** | Repo is the git repository | 
**revision** | **str** | Revision is the git commit, tag, branch to checkout | [optional] 
**ssh_private_key_secret** | [**V1SecretKeySelector**](V1SecretKeySelector.md) |  | [optional] 
**username_secret** | [**V1SecretKeySelector**](V1SecretKeySelector.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


