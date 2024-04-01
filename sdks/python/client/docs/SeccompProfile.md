# SeccompProfile

SeccompProfile defines a pod/container's seccomp profile settings. Only one profile source may be set.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**localhost_profile** | **str** | localhostProfile indicates a profile defined in a file on the node should be used. The profile must be preconfigured on the node to work. Must be a descending path, relative to the kubelet&#39;s configured seccomp profile location. Must only be set if type is \&quot;Localhost\&quot;. | [optional] 
**type** | **str** | type indicates which kind of seccomp profile will be applied. Valid options are:  Localhost - a profile defined in a file on the node should be used. RuntimeDefault - the container runtime default profile should be used. Unconfined - no profile should be applied.  Possible enum values:  - &#x60;\&quot;Localhost\&quot;&#x60; indicates a profile defined in a file on the node should be used. The file&#39;s location relative to &lt;kubelet-root-dir&gt;/seccomp.  - &#x60;\&quot;RuntimeDefault\&quot;&#x60; represents the default container runtime seccomp profile.  - &#x60;\&quot;Unconfined\&quot;&#x60; indicates no seccomp profile is applied (A.K.A. unconfined). | 

## Example

```python
from argo_workflows.models.seccomp_profile import SeccompProfile

# TODO update the JSON string below
json = "{}"
# create an instance of SeccompProfile from a JSON string
seccomp_profile_instance = SeccompProfile.from_json(json)
# print the JSON string representation of the object
print(SeccompProfile.to_json())

# convert the object into a dict
seccomp_profile_dict = seccomp_profile_instance.to_dict()
# create an instance of SeccompProfile from a dict
seccomp_profile_form_dict = seccomp_profile.from_dict(seccomp_profile_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


