# TCPSocketAction

TCPSocketAction describes an action based on opening a socket

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**host** | **str** | Optional: Host name to connect to, defaults to the pod IP. | [optional] 
**port** | **str** |  | 

## Example

```python
from argo_workflows.models.tcp_socket_action import TCPSocketAction

# TODO update the JSON string below
json = "{}"
# create an instance of TCPSocketAction from a JSON string
tcp_socket_action_instance = TCPSocketAction.from_json(json)
# print the JSON string representation of the object
print(TCPSocketAction.to_json())

# convert the object into a dict
tcp_socket_action_dict = tcp_socket_action_instance.to_dict()
# create an instance of TCPSocketAction from a dict
tcp_socket_action_form_dict = tcp_socket_action.from_dict(tcp_socket_action_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


