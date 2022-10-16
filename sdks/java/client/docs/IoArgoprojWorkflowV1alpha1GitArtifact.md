

# IoArgoprojWorkflowV1alpha1GitArtifact

GitArtifact is the location of an git artifact

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**branch** | **String** | Branch is the branch to fetch when &#x60;SingleBranch&#x60; is enabled |  [optional]
**depth** | **Integer** | Depth specifies clones/fetches should be shallow and include the given number of commits from the branch tip |  [optional]
**disableSubmodules** | **Boolean** | DisableSubmodules disables submodules during git clone |  [optional]
**fetch** | **List&lt;String&gt;** | Fetch specifies a number of refs that should be fetched before checkout |  [optional]
**insecureIgnoreHostKey** | **Boolean** | InsecureIgnoreHostKey disables SSH strict host key checking during git clone |  [optional]
**passwordSecret** | [**SecretKeySelector**](SecretKeySelector.md) |  |  [optional]
**repo** | **String** | Repo is the git repository | 
**revision** | **String** | Revision is the git commit, tag, branch to checkout |  [optional]
**singleBranch** | **Boolean** | SingleBranch enables single branch clone, using the &#x60;branch&#x60; parameter |  [optional]
**sshPrivateKeySecret** | [**SecretKeySelector**](SecretKeySelector.md) |  |  [optional]
**usernameSecret** | [**SecretKeySelector**](SecretKeySelector.md) |  |  [optional]



