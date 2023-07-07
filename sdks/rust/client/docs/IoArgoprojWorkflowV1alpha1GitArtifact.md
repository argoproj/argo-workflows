# IoArgoprojWorkflowV1alpha1GitArtifact

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**branch** | Option<**String**> | Branch is the branch to fetch when `SingleBranch` is enabled | [optional]
**depth** | Option<**i32**> | Depth specifies clones/fetches should be shallow and include the given number of commits from the branch tip | [optional]
**disable_submodules** | Option<**bool**> | DisableSubmodules disables submodules during git clone | [optional]
**fetch** | Option<**Vec<String>**> | Fetch specifies a number of refs that should be fetched before checkout | [optional]
**insecure_ignore_host_key** | Option<**bool**> | InsecureIgnoreHostKey disables SSH strict host key checking during git clone | [optional]
**password_secret** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**repo** | **String** | Repo is the git repository | 
**revision** | Option<**String**> | Revision is the git commit, tag, branch to checkout | [optional]
**single_branch** | Option<**bool**> | SingleBranch enables single branch clone, using the `branch` parameter | [optional]
**ssh_private_key_secret** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**username_secret** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


