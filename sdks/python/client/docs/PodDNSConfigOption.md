# PodDNSConfigOption

PodDNSConfigOption defines DNS resolver options of a pod.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | Required. | [optional] 
**value** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.pod_dns_config_option import PodDNSConfigOption

# TODO update the JSON string below
json = "{}"
# create an instance of PodDNSConfigOption from a JSON string
pod_dns_config_option_instance = PodDNSConfigOption.from_json(json)
# print the JSON string representation of the object
print(PodDNSConfigOption.to_json())

# convert the object into a dict
pod_dns_config_option_dict = pod_dns_config_option_instance.to_dict()
# create an instance of PodDNSConfigOption from a dict
pod_dns_config_option_form_dict = pod_dns_config_option.from_dict(pod_dns_config_option_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


