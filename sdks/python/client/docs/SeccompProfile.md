# SeccompProfile

SeccompProfile defines a pod/container's seccomp profile settings. Only one profile source may be set.

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**type** | **str** | type indicates which kind of seccomp profile will be applied. Valid options are:  Localhost - a profile defined in a file on the node should be used. RuntimeDefault - the container runtime default profile should be used. Unconfined - no profile should be applied.  Possible enum values:  - &#x60;\&quot;Localhost\&quot;&#x60; indicates a profile defined in a file on the node should be used. The file&#39;s location relative to &lt;kubelet-root-dir&gt;/seccomp.  - &#x60;\&quot;RuntimeDefault\&quot;&#x60; represents the default container runtime seccomp profile.  - &#x60;\&quot;Unconfined\&quot;&#x60; indicates no seccomp profile is applied (A.K.A. unconfined). | 
**localhost_profile** | **str** | localhostProfile indicates a profile defined in a file on the node should be used. The profile must be preconfigured on the node to work. Must be a descending path, relative to the kubelet&#39;s configured seccomp profile location. Must only be set if type is \&quot;Localhost\&quot;. | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


