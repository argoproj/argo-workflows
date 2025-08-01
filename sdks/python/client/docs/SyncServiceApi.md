# argo_workflows.SyncServiceApi

All URIs are relative to *http://localhost:2746*

Method | HTTP request | Description
------------- | ------------- | -------------
[**create_sync_limit**](SyncServiceApi.md#create_sync_limit) | **POST** /api/v1/sync/{namespace} | 
[**delete_sync_limit**](SyncServiceApi.md#delete_sync_limit) | **DELETE** /api/v1/sync/{namespace}/{name} | 
[**get_sync_limit**](SyncServiceApi.md#get_sync_limit) | **GET** /api/v1/sync/{namespace}/{name} | 
[**update_sync_limit**](SyncServiceApi.md#update_sync_limit) | **PUT** /api/v1/sync/{namespace}/{name} | 


# **create_sync_limit**
> SyncSyncLimitResponse create_sync_limit(namespace, body)



### Example

* Api Key Authentication (BearerToken):

```python
import time
import argo_workflows
from argo_workflows.api import sync_service_api
from argo_workflows.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from argo_workflows.model.sync_create_sync_limit_request import SyncCreateSyncLimitRequest
from argo_workflows.model.sync_sync_limit_response import SyncSyncLimitResponse
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = argo_workflows.Configuration(
    host = "http://localhost:2746"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: BearerToken
configuration.api_key['BearerToken'] = 'YOUR_API_KEY'

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['BearerToken'] = 'Bearer'

# Enter a context with an instance of the API client
with argo_workflows.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = sync_service_api.SyncServiceApi(api_client)
    namespace = "namespace_example" # str | 
    body = SyncCreateSyncLimitRequest(
        key="key_example",
        name="name_example",
        namespace="namespace_example",
        size_limit=1,
        type=SyncSyncConfigType("CONFIG_MAP"),
    ) # SyncCreateSyncLimitRequest | 

    # example passing only required values which don't have defaults set
    try:
        api_response = api_instance.create_sync_limit(namespace, body)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling SyncServiceApi->create_sync_limit: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  |
 **body** | [**SyncCreateSyncLimitRequest**](SyncCreateSyncLimitRequest.md)|  |

### Return type

[**SyncSyncLimitResponse**](SyncSyncLimitResponse.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **delete_sync_limit**
> bool, date, datetime, dict, float, int, list, str, none_type delete_sync_limit(namespace, name)



### Example

* Api Key Authentication (BearerToken):

```python
import time
import argo_workflows
from argo_workflows.api import sync_service_api
from argo_workflows.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = argo_workflows.Configuration(
    host = "http://localhost:2746"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: BearerToken
configuration.api_key['BearerToken'] = 'YOUR_API_KEY'

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['BearerToken'] = 'Bearer'

# Enter a context with an instance of the API client
with argo_workflows.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = sync_service_api.SyncServiceApi(api_client)
    namespace = "namespace_example" # str | 
    name = "name_example" # str | 
    type = "CONFIG_MAP" # str |  (optional) if omitted the server will use the default value of "CONFIG_MAP"
    key = "key_example" # str |  (optional)

    # example passing only required values which don't have defaults set
    try:
        api_response = api_instance.delete_sync_limit(namespace, name)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling SyncServiceApi->delete_sync_limit: %s\n" % e)

    # example passing only required values which don't have defaults set
    # and optional values
    try:
        api_response = api_instance.delete_sync_limit(namespace, name, type=type, key=key)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling SyncServiceApi->delete_sync_limit: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  |
 **name** | **str**|  |
 **type** | **str**|  | [optional] if omitted the server will use the default value of "CONFIG_MAP"
 **key** | **str**|  | [optional]

### Return type

**bool, date, datetime, dict, float, int, list, str, none_type**

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_sync_limit**
> SyncSyncLimitResponse get_sync_limit(namespace, name)



### Example

* Api Key Authentication (BearerToken):

```python
import time
import argo_workflows
from argo_workflows.api import sync_service_api
from argo_workflows.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from argo_workflows.model.sync_sync_limit_response import SyncSyncLimitResponse
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = argo_workflows.Configuration(
    host = "http://localhost:2746"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: BearerToken
configuration.api_key['BearerToken'] = 'YOUR_API_KEY'

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['BearerToken'] = 'Bearer'

# Enter a context with an instance of the API client
with argo_workflows.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = sync_service_api.SyncServiceApi(api_client)
    namespace = "namespace_example" # str | 
    name = "name_example" # str | 
    type = "CONFIG_MAP" # str |  (optional) if omitted the server will use the default value of "CONFIG_MAP"
    key = "key_example" # str |  (optional)

    # example passing only required values which don't have defaults set
    try:
        api_response = api_instance.get_sync_limit(namespace, name)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling SyncServiceApi->get_sync_limit: %s\n" % e)

    # example passing only required values which don't have defaults set
    # and optional values
    try:
        api_response = api_instance.get_sync_limit(namespace, name, type=type, key=key)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling SyncServiceApi->get_sync_limit: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  |
 **name** | **str**|  |
 **type** | **str**|  | [optional] if omitted the server will use the default value of "CONFIG_MAP"
 **key** | **str**|  | [optional]

### Return type

[**SyncSyncLimitResponse**](SyncSyncLimitResponse.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **update_sync_limit**
> SyncSyncLimitResponse update_sync_limit(namespace, name)



### Example

* Api Key Authentication (BearerToken):

```python
import time
import argo_workflows
from argo_workflows.api import sync_service_api
from argo_workflows.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from argo_workflows.model.sync_sync_limit_response import SyncSyncLimitResponse
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = argo_workflows.Configuration(
    host = "http://localhost:2746"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: BearerToken
configuration.api_key['BearerToken'] = 'YOUR_API_KEY'

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['BearerToken'] = 'Bearer'

# Enter a context with an instance of the API client
with argo_workflows.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = sync_service_api.SyncServiceApi(api_client)
    namespace = "namespace_example" # str | 
    name = "name_example" # str | 

    # example passing only required values which don't have defaults set
    try:
        api_response = api_instance.update_sync_limit(namespace, name)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling SyncServiceApi->update_sync_limit: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  |
 **name** | **str**|  |

### Return type

[**SyncSyncLimitResponse**](SyncSyncLimitResponse.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

