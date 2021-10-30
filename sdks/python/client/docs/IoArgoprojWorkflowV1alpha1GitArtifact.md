# IoArgoprojWorkflowV1alpha1GitArtifact

GitArtifact is the location of an git artifact

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**repo** | **str** | Repo is the git repository | 
**depth** | **int** | Depth specifies clones/fetches should be shallow and include the given number of commits from the branch tip | [optional] 
**disable_submodules** | **bool** | DisableSubmodules disables submodules during git clone | [optional] 
**fetch** | **[str]** | Fetch specifies a number of refs that should be fetched before checkout | [optional] 
**insecure_ignore_host_key** | **bool** | InsecureIgnoreHostKey disables SSH strict host key checking during git clone | [optional] 
**password_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**revision** | **str** | Revision is the git commit, tag, branch to checkout | [optional] 
**ssh_private_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**username_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


