# GrpcGatewayRuntimeStreamError


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**details** | [**List[GoogleProtobufAny]**](GoogleProtobufAny.md) |  | [optional] 
**grpc_code** | **int** |  | [optional] 
**http_code** | **int** |  | [optional] 
**http_status** | **str** |  | [optional] 
**message** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.grpc_gateway_runtime_stream_error import GrpcGatewayRuntimeStreamError

# TODO update the JSON string below
json = "{}"
# create an instance of GrpcGatewayRuntimeStreamError from a JSON string
grpc_gateway_runtime_stream_error_instance = GrpcGatewayRuntimeStreamError.from_json(json)
# print the JSON string representation of the object
print(GrpcGatewayRuntimeStreamError.to_json())

# convert the object into a dict
grpc_gateway_runtime_stream_error_dict = grpc_gateway_runtime_stream_error_instance.to_dict()
# create an instance of GrpcGatewayRuntimeStreamError from a dict
grpc_gateway_runtime_stream_error_form_dict = grpc_gateway_runtime_stream_error.from_dict(grpc_gateway_runtime_stream_error_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


