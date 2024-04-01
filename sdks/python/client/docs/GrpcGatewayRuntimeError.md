# GrpcGatewayRuntimeError


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**code** | **int** |  | [optional] 
**details** | [**List[GoogleProtobufAny]**](GoogleProtobufAny.md) |  | [optional] 
**error** | **str** |  | [optional] 
**message** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.grpc_gateway_runtime_error import GrpcGatewayRuntimeError

# TODO update the JSON string below
json = "{}"
# create an instance of GrpcGatewayRuntimeError from a JSON string
grpc_gateway_runtime_error_instance = GrpcGatewayRuntimeError.from_json(json)
# print the JSON string representation of the object
print(GrpcGatewayRuntimeError.to_json())

# convert the object into a dict
grpc_gateway_runtime_error_dict = grpc_gateway_runtime_error_instance.to_dict()
# create an instance of GrpcGatewayRuntimeError from a dict
grpc_gateway_runtime_error_form_dict = grpc_gateway_runtime_error.from_dict(grpc_gateway_runtime_error_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


