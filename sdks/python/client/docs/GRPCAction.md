# GRPCAction


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**port** | **int** | Port number of the gRPC service. Number must be in the range 1 to 65535. | 
**service** | **str** | Service is the name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).  If this is not specified, the default behavior is defined by gRPC. | [optional] 

## Example

```python
from argo_workflows.models.grpc_action import GRPCAction

# TODO update the JSON string below
json = "{}"
# create an instance of GRPCAction from a JSON string
grpc_action_instance = GRPCAction.from_json(json)
# print the JSON string representation of the object
print(GRPCAction.to_json())

# convert the object into a dict
grpc_action_dict = grpc_action_instance.to_dict()
# create an instance of GRPCAction from a dict
grpc_action_form_dict = grpc_action.from_dict(grpc_action_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


