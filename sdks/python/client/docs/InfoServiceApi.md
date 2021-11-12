# openapi_client.InfoServiceApi

All URIs are relative to *http://localhost:2746*

Method | HTTP request | Description
------------- | ------------- | -------------
[**info_service_get_info**](InfoServiceApi.md#info_service_get_info) | **GET** /api/v1/info | 
[**info_service_get_user_info**](InfoServiceApi.md#info_service_get_user_info) | **GET** /api/v1/userinfo | 
[**info_service_get_version**](InfoServiceApi.md#info_service_get_version) | **GET** /api/v1/version | 


# **info_service_get_info**
> IoArgoprojWorkflowV1alpha1InfoResponse info_service_get_info()



### Example

```python
import time
import openapi_client
from openapi_client.api import info_service_api
from openapi_client.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from openapi_client.model.io_argoproj_workflow_v1alpha1_info_response import IoArgoprojWorkflowV1alpha1InfoResponse
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = openapi_client.Configuration(
    host = "http://localhost:2746"
)


# Enter a context with an instance of the API client
with openapi_client.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = info_service_api.InfoServiceApi(api_client)

    # example, this endpoint has no required or optional parameters
    try:
        api_response = api_instance.info_service_get_info()
        pprint(api_response)
    except openapi_client.ApiException as e:
        print("Exception when calling InfoServiceApi->info_service_get_info: %s\n" % e)
```


### Parameters
This endpoint does not need any parameter.

### Return type

[**IoArgoprojWorkflowV1alpha1InfoResponse**](IoArgoprojWorkflowV1alpha1InfoResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **info_service_get_user_info**
> IoArgoprojWorkflowV1alpha1GetUserInfoResponse info_service_get_user_info()



### Example

```python
import time
import openapi_client
from openapi_client.api import info_service_api
from openapi_client.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from openapi_client.model.io_argoproj_workflow_v1alpha1_get_user_info_response import IoArgoprojWorkflowV1alpha1GetUserInfoResponse
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = openapi_client.Configuration(
    host = "http://localhost:2746"
)


# Enter a context with an instance of the API client
with openapi_client.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = info_service_api.InfoServiceApi(api_client)

    # example, this endpoint has no required or optional parameters
    try:
        api_response = api_instance.info_service_get_user_info()
        pprint(api_response)
    except openapi_client.ApiException as e:
        print("Exception when calling InfoServiceApi->info_service_get_user_info: %s\n" % e)
```


### Parameters
This endpoint does not need any parameter.

### Return type

[**IoArgoprojWorkflowV1alpha1GetUserInfoResponse**](IoArgoprojWorkflowV1alpha1GetUserInfoResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **info_service_get_version**
> IoArgoprojWorkflowV1alpha1Version info_service_get_version()



### Example

```python
import time
import openapi_client
from openapi_client.api import info_service_api
from openapi_client.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from openapi_client.model.io_argoproj_workflow_v1alpha1_version import IoArgoprojWorkflowV1alpha1Version
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = openapi_client.Configuration(
    host = "http://localhost:2746"
)


# Enter a context with an instance of the API client
with openapi_client.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = info_service_api.InfoServiceApi(api_client)

    # example, this endpoint has no required or optional parameters
    try:
        api_response = api_instance.info_service_get_version()
        pprint(api_response)
    except openapi_client.ApiException as e:
        print("Exception when calling InfoServiceApi->info_service_get_version: %s\n" % e)
```


### Parameters
This endpoint does not need any parameter.

### Return type

[**IoArgoprojWorkflowV1alpha1Version**](IoArgoprojWorkflowV1alpha1Version.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

