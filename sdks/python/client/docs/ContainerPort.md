# ContainerPort

ContainerPort represents a network port in a single container.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**container_port** | **int** | Number of port to expose on the pod&#39;s IP address. This must be a valid port number, 0 &lt; x &lt; 65536. | 
**host_ip** | **str** | What host IP to bind the external port to. | [optional] 
**host_port** | **int** | Number of port to expose on the host. If specified, this must be a valid port number, 0 &lt; x &lt; 65536. If HostNetwork is specified, this must match ContainerPort. Most containers do not need this. | [optional] 
**name** | **str** | If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services. | [optional] 
**protocol** | **str** | Protocol for port. Must be UDP, TCP, or SCTP. Defaults to \&quot;TCP\&quot;.  Possible enum values:  - &#x60;\&quot;SCTP\&quot;&#x60; is the SCTP protocol.  - &#x60;\&quot;TCP\&quot;&#x60; is the TCP protocol.  - &#x60;\&quot;UDP\&quot;&#x60; is the UDP protocol. | [optional] 

## Example

```python
from argo_workflows.models.container_port import ContainerPort

# TODO update the JSON string below
json = "{}"
# create an instance of ContainerPort from a JSON string
container_port_instance = ContainerPort.from_json(json)
# print the JSON string representation of the object
print(ContainerPort.to_json())

# convert the object into a dict
container_port_dict = container_port_instance.to_dict()
# create an instance of ContainerPort from a dict
container_port_form_dict = container_port.from_dict(container_port_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


