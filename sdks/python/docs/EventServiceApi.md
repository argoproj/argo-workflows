# openapi_client.EventServiceApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**receive_event**](EventServiceApi.md#receive_event) | **POST** /api/v1/events/{namespace}/{discriminator} | 


# **receive_event**
> object receive_event(namespace, discriminator, body)



### Example

```python
from __future__ import print_function
import time
import openapi_client
from openapi_client.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = openapi_client.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with openapi_client.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = openapi_client.EventServiceApi(api_client)
    namespace = 'namespace_example' # str | The namespace for the io.argoproj.workflow.v1alpha1. This can be empty if the client has cluster scoped permissions. If empty, then the event is \"broadcast\" to workflow event binding in all namespaces.
discriminator = 'discriminator_example' # str | Optional discriminator for the io.argoproj.workflow.v1alpha1. This should almost always be empty. Used for edge-cases where the event payload alone is not provide enough information to discriminate the event. This MUST NOT be used as security mechanism, e.g. to allow two clients to use the same access token, or to support webhooks on unsecured server. Instead, use access tokens. This is made available as `discriminator` in the event binding selector (`/spec/event/selector)`
body = None # object | The event itself can be any data.

    try:
        api_response = api_instance.receive_event(namespace, discriminator, body)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling EventServiceApi->receive_event: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**| The namespace for the io.argoproj.workflow.v1alpha1. This can be empty if the client has cluster scoped permissions. If empty, then the event is \&quot;broadcast\&quot; to workflow event binding in all namespaces. | 
 **discriminator** | **str**| Optional discriminator for the io.argoproj.workflow.v1alpha1. This should almost always be empty. Used for edge-cases where the event payload alone is not provide enough information to discriminate the event. This MUST NOT be used as security mechanism, e.g. to allow two clients to use the same access token, or to support webhooks on unsecured server. Instead, use access tokens. This is made available as &#x60;discriminator&#x60; in the event binding selector (&#x60;/spec/event/selector)&#x60; | 
 **body** | **object**| The event itself can be any data. | 

### Return type

**object**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

