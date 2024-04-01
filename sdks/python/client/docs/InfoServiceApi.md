# argo_workflows.InfoServiceApi

All URIs are relative to *http://localhost:2746*

Method | HTTP request | Description
------------- | ------------- | -------------
[**collect_event**](InfoServiceApi.md#collect_event) | **POST** /api/v1/tracking/event | 
[**get_info**](InfoServiceApi.md#get_info) | **GET** /api/v1/info | 
[**get_user_info**](InfoServiceApi.md#get_user_info) | **GET** /api/v1/userinfo | 
[**get_version**](InfoServiceApi.md#get_version) | **GET** /api/v1/version | 


# **collect_event**
> object collect_event(body)



### Example

* Api Key Authentication (BearerToken):

```python
import argo_workflows
from argo_workflows.models.io_argoproj_workflow_v1alpha1_collect_event_request import IoArgoprojWorkflowV1alpha1CollectEventRequest
from argo_workflows.rest import ApiException
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
configuration.api_key['BearerToken'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['BearerToken'] = 'Bearer'

# Enter a context with an instance of the API client
with argo_workflows.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = argo_workflows.InfoServiceApi(api_client)
    body = argo_workflows.IoArgoprojWorkflowV1alpha1CollectEventRequest() # IoArgoprojWorkflowV1alpha1CollectEventRequest | 

    try:
        api_response = api_instance.collect_event(body)
        print("The response of InfoServiceApi->collect_event:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling InfoServiceApi->collect_event: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**IoArgoprojWorkflowV1alpha1CollectEventRequest**](IoArgoprojWorkflowV1alpha1CollectEventRequest.md)|  | 

### Return type

**object**

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

# **get_info**
> IoArgoprojWorkflowV1alpha1InfoResponse get_info()



### Example

* Api Key Authentication (BearerToken):

```python
import argo_workflows
from argo_workflows.models.io_argoproj_workflow_v1alpha1_info_response import IoArgoprojWorkflowV1alpha1InfoResponse
from argo_workflows.rest import ApiException
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
configuration.api_key['BearerToken'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['BearerToken'] = 'Bearer'

# Enter a context with an instance of the API client
with argo_workflows.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = argo_workflows.InfoServiceApi(api_client)

    try:
        api_response = api_instance.get_info()
        print("The response of InfoServiceApi->get_info:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling InfoServiceApi->get_info: %s\n" % e)
```



### Parameters

This endpoint does not need any parameter.

### Return type

[**IoArgoprojWorkflowV1alpha1InfoResponse**](IoArgoprojWorkflowV1alpha1InfoResponse.md)

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

# **get_user_info**
> IoArgoprojWorkflowV1alpha1GetUserInfoResponse get_user_info()



### Example

* Api Key Authentication (BearerToken):

```python
import argo_workflows
from argo_workflows.models.io_argoproj_workflow_v1alpha1_get_user_info_response import IoArgoprojWorkflowV1alpha1GetUserInfoResponse
from argo_workflows.rest import ApiException
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
configuration.api_key['BearerToken'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['BearerToken'] = 'Bearer'

# Enter a context with an instance of the API client
with argo_workflows.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = argo_workflows.InfoServiceApi(api_client)

    try:
        api_response = api_instance.get_user_info()
        print("The response of InfoServiceApi->get_user_info:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling InfoServiceApi->get_user_info: %s\n" % e)
```



### Parameters

This endpoint does not need any parameter.

### Return type

[**IoArgoprojWorkflowV1alpha1GetUserInfoResponse**](IoArgoprojWorkflowV1alpha1GetUserInfoResponse.md)

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

# **get_version**
> IoArgoprojWorkflowV1alpha1Version get_version()



### Example

* Api Key Authentication (BearerToken):

```python
import argo_workflows
from argo_workflows.models.io_argoproj_workflow_v1alpha1_version import IoArgoprojWorkflowV1alpha1Version
from argo_workflows.rest import ApiException
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
configuration.api_key['BearerToken'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['BearerToken'] = 'Bearer'

# Enter a context with an instance of the API client
with argo_workflows.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = argo_workflows.InfoServiceApi(api_client)

    try:
        api_response = api_instance.get_version()
        print("The response of InfoServiceApi->get_version:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling InfoServiceApi->get_version: %s\n" % e)
```



### Parameters

This endpoint does not need any parameter.

### Return type

[**IoArgoprojWorkflowV1alpha1Version**](IoArgoprojWorkflowV1alpha1Version.md)

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

