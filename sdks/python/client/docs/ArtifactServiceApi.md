# argo_workflows.ArtifactServiceApi

All URIs are relative to *http://localhost:2746*

Method | HTTP request | Description
------------- | ------------- | -------------
[**get_input_artifact**](ArtifactServiceApi.md#get_input_artifact) | **GET** /input-artifacts/{namespace}/{name}/{podName}/{artifactName} | Get an input artifact.
[**get_input_artifact_by_uid**](ArtifactServiceApi.md#get_input_artifact_by_uid) | **GET** /input-artifacts-by-uid/{uid}/{podName}/{artifactName} | Get an input artifact by UID.
[**get_output_artifact**](ArtifactServiceApi.md#get_output_artifact) | **GET** /artifacts/{namespace}/{name}/{podName}/{artifactName} | Get an output artifact.
[**get_output_artifact_by_uid**](ArtifactServiceApi.md#get_output_artifact_by_uid) | **GET** /artifacts-by-uid/{uid}/{podName}/{artifactName} | Get an output artifact by UID.


# **get_input_artifact**
> get_input_artifact(namespace, name, pod_name, artifact_name)

Get an input artifact.

### Example

```python
import time
import argo_workflows
from argo_workflows.api import artifact_service_api
from argo_workflows.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = argo_workflows.Configuration(
    host = "http://localhost:2746"
)


# Enter a context with an instance of the API client
with argo_workflows.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = artifact_service_api.ArtifactServiceApi(api_client)
    namespace = "namespace_example" # str | 
    name = "name_example" # str | 
    pod_name = "podName_example" # str | 
    artifact_name = "artifactName_example" # str | 

    # example passing only required values which don't have defaults set
    try:
        # Get an input artifact.
        api_instance.get_input_artifact(namespace, name, pod_name, artifact_name)
    except argo_workflows.ApiException as e:
        print("Exception when calling ArtifactServiceApi->get_input_artifact: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  |
 **name** | **str**|  |
 **pod_name** | **str**|  |
 **artifact_name** | **str**|  |

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | An artifact file. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_input_artifact_by_uid**
> get_input_artifact_by_uid(namespace, uid, pod_name, artifact_name)

Get an input artifact by UID.

### Example

```python
import time
import argo_workflows
from argo_workflows.api import artifact_service_api
from argo_workflows.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = argo_workflows.Configuration(
    host = "http://localhost:2746"
)


# Enter a context with an instance of the API client
with argo_workflows.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = artifact_service_api.ArtifactServiceApi(api_client)
    namespace = "namespace_example" # str | 
    uid = "uid_example" # str | 
    pod_name = "podName_example" # str | 
    artifact_name = "artifactName_example" # str | 

    # example passing only required values which don't have defaults set
    try:
        # Get an input artifact by UID.
        api_instance.get_input_artifact_by_uid(namespace, uid, pod_name, artifact_name)
    except argo_workflows.ApiException as e:
        print("Exception when calling ArtifactServiceApi->get_input_artifact_by_uid: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  |
 **uid** | **str**|  |
 **pod_name** | **str**|  |
 **artifact_name** | **str**|  |

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | An artifact file. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_output_artifact**
> get_output_artifact(namespace, name, pod_name, artifact_name)

Get an output artifact.

### Example

```python
import time
import argo_workflows
from argo_workflows.api import artifact_service_api
from argo_workflows.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = argo_workflows.Configuration(
    host = "http://localhost:2746"
)


# Enter a context with an instance of the API client
with argo_workflows.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = artifact_service_api.ArtifactServiceApi(api_client)
    namespace = "namespace_example" # str | 
    name = "name_example" # str | 
    pod_name = "podName_example" # str | 
    artifact_name = "artifactName_example" # str | 

    # example passing only required values which don't have defaults set
    try:
        # Get an output artifact.
        api_instance.get_output_artifact(namespace, name, pod_name, artifact_name)
    except argo_workflows.ApiException as e:
        print("Exception when calling ArtifactServiceApi->get_output_artifact: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  |
 **name** | **str**|  |
 **pod_name** | **str**|  |
 **artifact_name** | **str**|  |

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | An artifact file. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_output_artifact_by_uid**
> get_output_artifact_by_uid(uid, pod_name, artifact_name)

Get an output artifact by UID.

### Example

```python
import time
import argo_workflows
from argo_workflows.api import artifact_service_api
from argo_workflows.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = argo_workflows.Configuration(
    host = "http://localhost:2746"
)


# Enter a context with an instance of the API client
with argo_workflows.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = artifact_service_api.ArtifactServiceApi(api_client)
    uid = "uid_example" # str | 
    pod_name = "podName_example" # str | 
    artifact_name = "artifactName_example" # str | 

    # example passing only required values which don't have defaults set
    try:
        # Get an output artifact by UID.
        api_instance.get_output_artifact_by_uid(uid, pod_name, artifact_name)
    except argo_workflows.ApiException as e:
        print("Exception when calling ArtifactServiceApi->get_output_artifact_by_uid: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **uid** | **str**|  |
 **pod_name** | **str**|  |
 **artifact_name** | **str**|  |

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | An artifact file. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

