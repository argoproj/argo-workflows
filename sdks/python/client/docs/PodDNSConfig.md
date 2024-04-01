# PodDNSConfig

PodDNSConfig defines the DNS parameters of a pod in addition to those generated from DNSPolicy.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**nameservers** | **List[str]** | A list of DNS name server IP addresses. This will be appended to the base nameservers generated from DNSPolicy. Duplicated nameservers will be removed. | [optional] 
**options** | [**List[PodDNSConfigOption]**](PodDNSConfigOption.md) | A list of DNS resolver options. This will be merged with the base options generated from DNSPolicy. Duplicated entries will be removed. Resolution options given in Options will override those that appear in the base DNSPolicy. | [optional] 
**searches** | **List[str]** | A list of DNS search domains for host-name lookup. This will be appended to the base search paths generated from DNSPolicy. Duplicated search paths will be removed. | [optional] 

## Example

```python
from argo_workflows.models.pod_dns_config import PodDNSConfig

# TODO update the JSON string below
json = "{}"
# create an instance of PodDNSConfig from a JSON string
pod_dns_config_instance = PodDNSConfig.from_json(json)
# print the JSON string representation of the object
print(PodDNSConfig.to_json())

# convert the object into a dict
pod_dns_config_dict = pod_dns_config_instance.to_dict()
# create an instance of PodDNSConfig from a dict
pod_dns_config_form_dict = pod_dns_config.from_dict(pod_dns_config_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


