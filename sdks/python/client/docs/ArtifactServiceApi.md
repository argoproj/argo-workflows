# argo_workflows.ArtifactServiceApi

All URIs are relative to *http://localhost:2746*

Method | HTTP request | Description
------------- | ------------- | -------------
[**get_artifact_file**](ArtifactServiceApi.md#get_artifact_file) | **GET** /artifact-files/{namespace}/{idDiscriminator}/{id}/{nodeId}/{artifactDiscriminator}/{artifactName} | Get an artifact.
[**get_input_artifact**](ArtifactServiceApi.md#get_input_artifact) | **GET** /input-artifacts/{namespace}/{name}/{nodeId}/{artifactName} | Get an input artifact.
[**get_input_artifact_by_uid**](ArtifactServiceApi.md#get_input_artifact_by_uid) | **GET** /input-artifacts-by-uid/{uid}/{nodeId}/{artifactName} | Get an input artifact by UID.
[**get_output_artifact**](ArtifactServiceApi.md#get_output_artifact) | **GET** /artifacts/{namespace}/{name}/{nodeId}/{artifactName} | Get an output artifact.
[**get_output_artifact_by_uid**](ArtifactServiceApi.md#get_output_artifact_by_uid) | **GET** /artifacts-by-uid/{uid}/{nodeId}/{artifactName} | Get an output artifact by UID.


# **get_artifact_file**
> file_type get_artifact_file(namespace, id_discriminator, id, node_id, artifact_name, )

Get an artifact.

### Example

* Api Key Authentication (BearerToken):

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
    api_instance = artifact_service_api.ArtifactServiceApi(api_client)
    namespace = "namespace_example" # str | 
    id_discriminator = "workflow" # str | 
    id = "id_example" # str | 
    node_id = "nodeId_example" # str | 
    artifact_name = "artifactName_example" # str | 

    # example passing only required values which don't have defaults set
    try:
        # Get an artifact.
        api_response = api_instance.get_artifact_file(namespace, id_discriminator, id, node_id, artifact_name, )
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling ArtifactServiceApi->get_artifact_file: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  |
 **id_discriminator** | **str**|  |
 **id** | **str**|  |
 **node_id** | **str**|  |
 **artifact_name** | **str**|  |
 **artifact_discriminator** | **str**|  | defaults to "outputs"

### Return type

**file_type**

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | An artifact file. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_input_artifact**
> file_type get_input_artifact(namespace, name, node_id, artifact_name)

Get an input artifact.

### Example

* Api Key Authentication (BearerToken):

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
    api_instance = artifact_service_api.ArtifactServiceApi(api_client)
    namespace = "namespace_example" # str | 
    name = "name_example" # str | 
    node_id = "nodeId_example" # str | 
    artifact_name = "artifactName_example" # str | 

    # example passing only required values which don't have defaults set
    try:
        # Get an input artifact.
        api_response = api_instance.get_input_artifact(namespace, name, node_id, artifact_name)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling ArtifactServiceApi->get_input_artifact: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  |
 **name** | **str**|  |
 **node_id** | **str**|  |
 **artifact_name** | **str**|  |

### Return type

**file_type**

### Authorization

[BearerToken](../README.md#BearerToken)

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
> file_type get_input_artifact_by_uid(uid, node_id, artifact_name)

Get an input artifact by UID.

### Example

* Api Key Authentication (BearerToken):

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
    api_instance = artifact_service_api.ArtifactServiceApi(api_client)
    uid = "uid_example" # str | 
    node_id = "nodeId_example" # str | 
    artifact_name = "artifactName_example" # str | 

    # example passing only required values which don't have defaults set
    try:
        # Get an input artifact by UID.
        api_response = api_instance.get_input_artifact_by_uid(uid, node_id, artifact_name)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling ArtifactServiceApi->get_input_artifact_by_uid: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **uid** | **str**|  |
 **node_id** | **str**|  |
 **artifact_name** | **str**|  |

### Return type

**file_type**

### Authorization

[BearerToken](../README.md#BearerToken)

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
> file_type get_output_artifact(namespace, name, node_id, artifact_name)

Get an output artifact.

### Example

* Api Key Authentication (BearerToken):

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
    api_instance = artifact_service_api.ArtifactServiceApi(api_client)
    namespace = "namespace_example" # str | 
    name = "name_example" # str | 
    node_id = "nodeId_example" # str | 
    artifact_name = "artifactName_example" # str | 

    # example passing only required values which don't have defaults set
    try:
        # Get an output artifact.
        api_response = api_instance.get_output_artifact(namespace, name, node_id, artifact_name)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling ArtifactServiceApi->get_output_artifact: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  |
 **name** | **str**|  |
 **node_id** | **str**|  |
 **artifact_name** | **str**|  |

### Return type

**file_type**

### Authorization

[BearerToken](../README.md#BearerToken)

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
> file_type get_output_artifact_by_uid(uid, node_id, artifact_name)

Get an output artifact by UID.

### Example

* Api Key Authentication (BearerToken):

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
    api_instance = artifact_service_api.ArtifactServiceApi(api_client)
    uid = "uid_example" # str | 
    node_id = "nodeId_example" # str | 
    artifact_name = "artifactName_example" # str | 

    # example passing only required values which don't have defaults set
    try:
        # Get an output artifact by UID.
        api_response = api_instance.get_output_artifact_by_uid(uid, node_id, artifact_name)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling ArtifactServiceApi->get_output_artifact_by_uid: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **uid** | **str**|  |
 **node_id** | **str**|  |
 **artifact_name** | **str**|  |

### Return type

**file_type**

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | An artifact file. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

