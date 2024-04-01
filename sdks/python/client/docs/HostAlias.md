# HostAlias

HostAlias holds the mapping between IP and hostnames that will be injected as an entry in the pod's hosts file.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**hostnames** | **List[str]** | Hostnames for the above IP address. | [optional] 
**ip** | **str** | IP address of the host file entry. | [optional] 

## Example

```python
from argo_workflows.models.host_alias import HostAlias

# TODO update the JSON string below
json = "{}"
# create an instance of HostAlias from a JSON string
host_alias_instance = HostAlias.from_json(json)
# print the JSON string representation of the object
print(HostAlias.to_json())

# convert the object into a dict
host_alias_dict = host_alias_instance.to_dict()
# create an instance of HostAlias from a dict
host_alias_form_dict = host_alias.from_dict(host_alias_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


