# argo_workflows.SensorServiceApi

All URIs are relative to *http://localhost:2746*

Method | HTTP request | Description
------------- | ------------- | -------------
[**create_sensor**](SensorServiceApi.md#create_sensor) | **POST** /api/v1/sensors/{namespace} | 
[**delete_sensor**](SensorServiceApi.md#delete_sensor) | **DELETE** /api/v1/sensors/{namespace}/{name} | 
[**get_sensor**](SensorServiceApi.md#get_sensor) | **GET** /api/v1/sensors/{namespace}/{name} | 
[**list_sensors**](SensorServiceApi.md#list_sensors) | **GET** /api/v1/sensors/{namespace} | 
[**sensors_logs**](SensorServiceApi.md#sensors_logs) | **GET** /api/v1/stream/sensors/{namespace}/logs | 
[**update_sensor**](SensorServiceApi.md#update_sensor) | **PUT** /api/v1/sensors/{namespace}/{name} | 
[**watch_sensors**](SensorServiceApi.md#watch_sensors) | **GET** /api/v1/stream/sensors/{namespace} | 


# **create_sensor**
> IoArgoprojEventsV1alpha1Sensor create_sensor(namespace, body)



### Example

* Api Key Authentication (BearerToken):

```python
import argo_workflows
from argo_workflows.models.io_argoproj_events_v1alpha1_sensor import IoArgoprojEventsV1alpha1Sensor
from argo_workflows.models.sensor_create_sensor_request import SensorCreateSensorRequest
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
    api_instance = argo_workflows.SensorServiceApi(api_client)
    namespace = 'namespace_example' # str | 
    body = argo_workflows.SensorCreateSensorRequest() # SensorCreateSensorRequest | 

    try:
        api_response = api_instance.create_sensor(namespace, body)
        print("The response of SensorServiceApi->create_sensor:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SensorServiceApi->create_sensor: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  | 
 **body** | [**SensorCreateSensorRequest**](SensorCreateSensorRequest.md)|  | 

### Return type

[**IoArgoprojEventsV1alpha1Sensor**](IoArgoprojEventsV1alpha1Sensor.md)

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

# **delete_sensor**
> object delete_sensor(namespace, name, delete_options_grace_period_seconds=delete_options_grace_period_seconds, delete_options_preconditions_uid=delete_options_preconditions_uid, delete_options_preconditions_resource_version=delete_options_preconditions_resource_version, delete_options_orphan_dependents=delete_options_orphan_dependents, delete_options_propagation_policy=delete_options_propagation_policy, delete_options_dry_run=delete_options_dry_run)



### Example

* Api Key Authentication (BearerToken):

```python
import argo_workflows
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
    api_instance = argo_workflows.SensorServiceApi(api_client)
    namespace = 'namespace_example' # str | 
    name = 'name_example' # str | 
    delete_options_grace_period_seconds = 'delete_options_grace_period_seconds_example' # str | The duration in seconds before the object should be deleted. Value must be non-negative integer. The value zero indicates delete immediately. If this value is nil, the default grace period for the specified type will be used. Defaults to a per object value if not specified. zero means delete immediately. +optional. (optional)
    delete_options_preconditions_uid = 'delete_options_preconditions_uid_example' # str | Specifies the target UID. +optional. (optional)
    delete_options_preconditions_resource_version = 'delete_options_preconditions_resource_version_example' # str | Specifies the target ResourceVersion +optional. (optional)
    delete_options_orphan_dependents = True # bool | Deprecated: please use the PropagationPolicy, this field will be deprecated in 1.7. Should the dependent objects be orphaned. If true/false, the \"orphan\" finalizer will be added to/removed from the object's finalizers list. Either this field or PropagationPolicy may be set, but not both. +optional. (optional)
    delete_options_propagation_policy = 'delete_options_propagation_policy_example' # str | Whether and how garbage collection will be performed. Either this field or OrphanDependents may be set, but not both. The default policy is decided by the existing finalizer set in the metadata.finalizers and the resource-specific default policy. Acceptable values are: 'Orphan' - orphan the dependents; 'Background' - allow the garbage collector to delete the dependents in the background; 'Foreground' - a cascading policy that deletes all dependents in the foreground. +optional. (optional)
    delete_options_dry_run = ['delete_options_dry_run_example'] # List[str] | When present, indicates that modifications should not be persisted. An invalid or unrecognized dryRun directive will result in an error response and no further processing of the request. Valid values are: - All: all dry run stages will be processed +optional. (optional)

    try:
        api_response = api_instance.delete_sensor(namespace, name, delete_options_grace_period_seconds=delete_options_grace_period_seconds, delete_options_preconditions_uid=delete_options_preconditions_uid, delete_options_preconditions_resource_version=delete_options_preconditions_resource_version, delete_options_orphan_dependents=delete_options_orphan_dependents, delete_options_propagation_policy=delete_options_propagation_policy, delete_options_dry_run=delete_options_dry_run)
        print("The response of SensorServiceApi->delete_sensor:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SensorServiceApi->delete_sensor: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  | 
 **name** | **str**|  | 
 **delete_options_grace_period_seconds** | **str**| The duration in seconds before the object should be deleted. Value must be non-negative integer. The value zero indicates delete immediately. If this value is nil, the default grace period for the specified type will be used. Defaults to a per object value if not specified. zero means delete immediately. +optional. | [optional] 
 **delete_options_preconditions_uid** | **str**| Specifies the target UID. +optional. | [optional] 
 **delete_options_preconditions_resource_version** | **str**| Specifies the target ResourceVersion +optional. | [optional] 
 **delete_options_orphan_dependents** | **bool**| Deprecated: please use the PropagationPolicy, this field will be deprecated in 1.7. Should the dependent objects be orphaned. If true/false, the \&quot;orphan\&quot; finalizer will be added to/removed from the object&#39;s finalizers list. Either this field or PropagationPolicy may be set, but not both. +optional. | [optional] 
 **delete_options_propagation_policy** | **str**| Whether and how garbage collection will be performed. Either this field or OrphanDependents may be set, but not both. The default policy is decided by the existing finalizer set in the metadata.finalizers and the resource-specific default policy. Acceptable values are: &#39;Orphan&#39; - orphan the dependents; &#39;Background&#39; - allow the garbage collector to delete the dependents in the background; &#39;Foreground&#39; - a cascading policy that deletes all dependents in the foreground. +optional. | [optional] 
 **delete_options_dry_run** | [**List[str]**](str.md)| When present, indicates that modifications should not be persisted. An invalid or unrecognized dryRun directive will result in an error response and no further processing of the request. Valid values are: - All: all dry run stages will be processed +optional. | [optional] 

### Return type

**object**

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

# **get_sensor**
> IoArgoprojEventsV1alpha1Sensor get_sensor(namespace, name, get_options_resource_version=get_options_resource_version)



### Example

* Api Key Authentication (BearerToken):

```python
import argo_workflows
from argo_workflows.models.io_argoproj_events_v1alpha1_sensor import IoArgoprojEventsV1alpha1Sensor
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
    api_instance = argo_workflows.SensorServiceApi(api_client)
    namespace = 'namespace_example' # str | 
    name = 'name_example' # str | 
    get_options_resource_version = 'get_options_resource_version_example' # str | resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional (optional)

    try:
        api_response = api_instance.get_sensor(namespace, name, get_options_resource_version=get_options_resource_version)
        print("The response of SensorServiceApi->get_sensor:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SensorServiceApi->get_sensor: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  | 
 **name** | **str**|  | 
 **get_options_resource_version** | **str**| resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional | [optional] 

### Return type

[**IoArgoprojEventsV1alpha1Sensor**](IoArgoprojEventsV1alpha1Sensor.md)

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

# **list_sensors**
> IoArgoprojEventsV1alpha1SensorList list_sensors(namespace, list_options_label_selector=list_options_label_selector, list_options_field_selector=list_options_field_selector, list_options_watch=list_options_watch, list_options_allow_watch_bookmarks=list_options_allow_watch_bookmarks, list_options_resource_version=list_options_resource_version, list_options_resource_version_match=list_options_resource_version_match, list_options_timeout_seconds=list_options_timeout_seconds, list_options_limit=list_options_limit, list_options_continue=list_options_continue)



### Example

* Api Key Authentication (BearerToken):

```python
import argo_workflows
from argo_workflows.models.io_argoproj_events_v1alpha1_sensor_list import IoArgoprojEventsV1alpha1SensorList
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
    api_instance = argo_workflows.SensorServiceApi(api_client)
    namespace = 'namespace_example' # str | 
    list_options_label_selector = 'list_options_label_selector_example' # str | A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. (optional)
    list_options_field_selector = 'list_options_field_selector_example' # str | A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. (optional)
    list_options_watch = True # bool | Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. (optional)
    list_options_allow_watch_bookmarks = True # bool | allowWatchBookmarks requests watch events with type \"BOOKMARK\". Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server's discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. +optional. (optional)
    list_options_resource_version = 'list_options_resource_version_example' # str | resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional (optional)
    list_options_resource_version_match = 'list_options_resource_version_match_example' # str | resourceVersionMatch determines how resourceVersion is applied to list calls. It is highly recommended that resourceVersionMatch be set for list calls where resourceVersion is set See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional (optional)
    list_options_timeout_seconds = 'list_options_timeout_seconds_example' # str | Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. (optional)
    list_options_limit = 'list_options_limit_example' # str | limit is a maximum number of responses to return for a list call. If more items exist, the server will set the `continue` field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. (optional)
    list_options_continue = 'list_options_continue_example' # str | The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the \"next key\".  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. (optional)

    try:
        api_response = api_instance.list_sensors(namespace, list_options_label_selector=list_options_label_selector, list_options_field_selector=list_options_field_selector, list_options_watch=list_options_watch, list_options_allow_watch_bookmarks=list_options_allow_watch_bookmarks, list_options_resource_version=list_options_resource_version, list_options_resource_version_match=list_options_resource_version_match, list_options_timeout_seconds=list_options_timeout_seconds, list_options_limit=list_options_limit, list_options_continue=list_options_continue)
        print("The response of SensorServiceApi->list_sensors:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SensorServiceApi->list_sensors: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  | 
 **list_options_label_selector** | **str**| A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. | [optional] 
 **list_options_field_selector** | **str**| A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. | [optional] 
 **list_options_watch** | **bool**| Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. | [optional] 
 **list_options_allow_watch_bookmarks** | **bool**| allowWatchBookmarks requests watch events with type \&quot;BOOKMARK\&quot;. Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server&#39;s discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. +optional. | [optional] 
 **list_options_resource_version** | **str**| resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional | [optional] 
 **list_options_resource_version_match** | **str**| resourceVersionMatch determines how resourceVersion is applied to list calls. It is highly recommended that resourceVersionMatch be set for list calls where resourceVersion is set See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional | [optional] 
 **list_options_timeout_seconds** | **str**| Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. | [optional] 
 **list_options_limit** | **str**| limit is a maximum number of responses to return for a list call. If more items exist, the server will set the &#x60;continue&#x60; field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. | [optional] 
 **list_options_continue** | **str**| The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the \&quot;next key\&quot;.  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. | [optional] 

### Return type

[**IoArgoprojEventsV1alpha1SensorList**](IoArgoprojEventsV1alpha1SensorList.md)

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

# **sensors_logs**
> StreamResultOfSensorLogEntry sensors_logs(namespace, name=name, trigger_name=trigger_name, grep=grep, pod_log_options_container=pod_log_options_container, pod_log_options_follow=pod_log_options_follow, pod_log_options_previous=pod_log_options_previous, pod_log_options_since_seconds=pod_log_options_since_seconds, pod_log_options_since_time_seconds=pod_log_options_since_time_seconds, pod_log_options_since_time_nanos=pod_log_options_since_time_nanos, pod_log_options_timestamps=pod_log_options_timestamps, pod_log_options_tail_lines=pod_log_options_tail_lines, pod_log_options_limit_bytes=pod_log_options_limit_bytes, pod_log_options_insecure_skip_tls_verify_backend=pod_log_options_insecure_skip_tls_verify_backend)



### Example

* Api Key Authentication (BearerToken):

```python
import argo_workflows
from argo_workflows.models.stream_result_of_sensor_log_entry import StreamResultOfSensorLogEntry
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
    api_instance = argo_workflows.SensorServiceApi(api_client)
    namespace = 'namespace_example' # str | 
    name = 'name_example' # str | optional - only return entries for this sensor name. (optional)
    trigger_name = 'trigger_name_example' # str | optional - only return entries for this trigger. (optional)
    grep = 'grep_example' # str | option - only return entries where `msg` contains this regular expressions. (optional)
    pod_log_options_container = 'pod_log_options_container_example' # str | The container for which to stream logs. Defaults to only container if there is one container in the pod. +optional. (optional)
    pod_log_options_follow = True # bool | Follow the log stream of the pod. Defaults to false. +optional. (optional)
    pod_log_options_previous = True # bool | Return previous terminated container logs. Defaults to false. +optional. (optional)
    pod_log_options_since_seconds = 'pod_log_options_since_seconds_example' # str | A relative time in seconds before the current time from which to show logs. If this value precedes the time a pod was started, only logs since the pod start will be returned. If this value is in the future, no logs will be returned. Only one of sinceSeconds or sinceTime may be specified. +optional. (optional)
    pod_log_options_since_time_seconds = 'pod_log_options_since_time_seconds_example' # str | Represents seconds of UTC time since Unix epoch 1970-01-01T00:00:00Z. Must be from 0001-01-01T00:00:00Z to 9999-12-31T23:59:59Z inclusive. (optional)
    pod_log_options_since_time_nanos = 56 # int | Non-negative fractions of a second at nanosecond resolution. Negative second values with fractions must still have non-negative nanos values that count forward in time. Must be from 0 to 999,999,999 inclusive. This field may be limited in precision depending on context. (optional)
    pod_log_options_timestamps = True # bool | If true, add an RFC3339 or RFC3339Nano timestamp at the beginning of every line of log output. Defaults to false. +optional. (optional)
    pod_log_options_tail_lines = 'pod_log_options_tail_lines_example' # str | If set, the number of lines from the end of the logs to show. If not specified, logs are shown from the creation of the container or sinceSeconds or sinceTime +optional. (optional)
    pod_log_options_limit_bytes = 'pod_log_options_limit_bytes_example' # str | If set, the number of bytes to read from the server before terminating the log output. This may not display a complete final line of logging, and may return slightly more or slightly less than the specified limit. +optional. (optional)
    pod_log_options_insecure_skip_tls_verify_backend = True # bool | insecureSkipTLSVerifyBackend indicates that the apiserver should not confirm the validity of the serving certificate of the backend it is connecting to.  This will make the HTTPS connection between the apiserver and the backend insecure. This means the apiserver cannot verify the log data it is receiving came from the real kubelet.  If the kubelet is configured to verify the apiserver's TLS credentials, it does not mean the connection to the real kubelet is vulnerable to a man in the middle attack (e.g. an attacker could not intercept the actual log data coming from the real kubelet). +optional. (optional)

    try:
        api_response = api_instance.sensors_logs(namespace, name=name, trigger_name=trigger_name, grep=grep, pod_log_options_container=pod_log_options_container, pod_log_options_follow=pod_log_options_follow, pod_log_options_previous=pod_log_options_previous, pod_log_options_since_seconds=pod_log_options_since_seconds, pod_log_options_since_time_seconds=pod_log_options_since_time_seconds, pod_log_options_since_time_nanos=pod_log_options_since_time_nanos, pod_log_options_timestamps=pod_log_options_timestamps, pod_log_options_tail_lines=pod_log_options_tail_lines, pod_log_options_limit_bytes=pod_log_options_limit_bytes, pod_log_options_insecure_skip_tls_verify_backend=pod_log_options_insecure_skip_tls_verify_backend)
        print("The response of SensorServiceApi->sensors_logs:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SensorServiceApi->sensors_logs: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  | 
 **name** | **str**| optional - only return entries for this sensor name. | [optional] 
 **trigger_name** | **str**| optional - only return entries for this trigger. | [optional] 
 **grep** | **str**| option - only return entries where &#x60;msg&#x60; contains this regular expressions. | [optional] 
 **pod_log_options_container** | **str**| The container for which to stream logs. Defaults to only container if there is one container in the pod. +optional. | [optional] 
 **pod_log_options_follow** | **bool**| Follow the log stream of the pod. Defaults to false. +optional. | [optional] 
 **pod_log_options_previous** | **bool**| Return previous terminated container logs. Defaults to false. +optional. | [optional] 
 **pod_log_options_since_seconds** | **str**| A relative time in seconds before the current time from which to show logs. If this value precedes the time a pod was started, only logs since the pod start will be returned. If this value is in the future, no logs will be returned. Only one of sinceSeconds or sinceTime may be specified. +optional. | [optional] 
 **pod_log_options_since_time_seconds** | **str**| Represents seconds of UTC time since Unix epoch 1970-01-01T00:00:00Z. Must be from 0001-01-01T00:00:00Z to 9999-12-31T23:59:59Z inclusive. | [optional] 
 **pod_log_options_since_time_nanos** | **int**| Non-negative fractions of a second at nanosecond resolution. Negative second values with fractions must still have non-negative nanos values that count forward in time. Must be from 0 to 999,999,999 inclusive. This field may be limited in precision depending on context. | [optional] 
 **pod_log_options_timestamps** | **bool**| If true, add an RFC3339 or RFC3339Nano timestamp at the beginning of every line of log output. Defaults to false. +optional. | [optional] 
 **pod_log_options_tail_lines** | **str**| If set, the number of lines from the end of the logs to show. If not specified, logs are shown from the creation of the container or sinceSeconds or sinceTime +optional. | [optional] 
 **pod_log_options_limit_bytes** | **str**| If set, the number of bytes to read from the server before terminating the log output. This may not display a complete final line of logging, and may return slightly more or slightly less than the specified limit. +optional. | [optional] 
 **pod_log_options_insecure_skip_tls_verify_backend** | **bool**| insecureSkipTLSVerifyBackend indicates that the apiserver should not confirm the validity of the serving certificate of the backend it is connecting to.  This will make the HTTPS connection between the apiserver and the backend insecure. This means the apiserver cannot verify the log data it is receiving came from the real kubelet.  If the kubelet is configured to verify the apiserver&#39;s TLS credentials, it does not mean the connection to the real kubelet is vulnerable to a man in the middle attack (e.g. an attacker could not intercept the actual log data coming from the real kubelet). +optional. | [optional] 

### Return type

[**StreamResultOfSensorLogEntry**](StreamResultOfSensorLogEntry.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response.(streaming responses) |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **update_sensor**
> IoArgoprojEventsV1alpha1Sensor update_sensor(namespace, name, body)



### Example

* Api Key Authentication (BearerToken):

```python
import argo_workflows
from argo_workflows.models.io_argoproj_events_v1alpha1_sensor import IoArgoprojEventsV1alpha1Sensor
from argo_workflows.models.sensor_update_sensor_request import SensorUpdateSensorRequest
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
    api_instance = argo_workflows.SensorServiceApi(api_client)
    namespace = 'namespace_example' # str | 
    name = 'name_example' # str | 
    body = argo_workflows.SensorUpdateSensorRequest() # SensorUpdateSensorRequest | 

    try:
        api_response = api_instance.update_sensor(namespace, name, body)
        print("The response of SensorServiceApi->update_sensor:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SensorServiceApi->update_sensor: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  | 
 **name** | **str**|  | 
 **body** | [**SensorUpdateSensorRequest**](SensorUpdateSensorRequest.md)|  | 

### Return type

[**IoArgoprojEventsV1alpha1Sensor**](IoArgoprojEventsV1alpha1Sensor.md)

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

# **watch_sensors**
> StreamResultOfSensorSensorWatchEvent watch_sensors(namespace, list_options_label_selector=list_options_label_selector, list_options_field_selector=list_options_field_selector, list_options_watch=list_options_watch, list_options_allow_watch_bookmarks=list_options_allow_watch_bookmarks, list_options_resource_version=list_options_resource_version, list_options_resource_version_match=list_options_resource_version_match, list_options_timeout_seconds=list_options_timeout_seconds, list_options_limit=list_options_limit, list_options_continue=list_options_continue)



### Example

* Api Key Authentication (BearerToken):

```python
import argo_workflows
from argo_workflows.models.stream_result_of_sensor_sensor_watch_event import StreamResultOfSensorSensorWatchEvent
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
    api_instance = argo_workflows.SensorServiceApi(api_client)
    namespace = 'namespace_example' # str | 
    list_options_label_selector = 'list_options_label_selector_example' # str | A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. (optional)
    list_options_field_selector = 'list_options_field_selector_example' # str | A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. (optional)
    list_options_watch = True # bool | Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. (optional)
    list_options_allow_watch_bookmarks = True # bool | allowWatchBookmarks requests watch events with type \"BOOKMARK\". Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server's discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. +optional. (optional)
    list_options_resource_version = 'list_options_resource_version_example' # str | resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional (optional)
    list_options_resource_version_match = 'list_options_resource_version_match_example' # str | resourceVersionMatch determines how resourceVersion is applied to list calls. It is highly recommended that resourceVersionMatch be set for list calls where resourceVersion is set See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional (optional)
    list_options_timeout_seconds = 'list_options_timeout_seconds_example' # str | Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. (optional)
    list_options_limit = 'list_options_limit_example' # str | limit is a maximum number of responses to return for a list call. If more items exist, the server will set the `continue` field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. (optional)
    list_options_continue = 'list_options_continue_example' # str | The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the \"next key\".  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. (optional)

    try:
        api_response = api_instance.watch_sensors(namespace, list_options_label_selector=list_options_label_selector, list_options_field_selector=list_options_field_selector, list_options_watch=list_options_watch, list_options_allow_watch_bookmarks=list_options_allow_watch_bookmarks, list_options_resource_version=list_options_resource_version, list_options_resource_version_match=list_options_resource_version_match, list_options_timeout_seconds=list_options_timeout_seconds, list_options_limit=list_options_limit, list_options_continue=list_options_continue)
        print("The response of SensorServiceApi->watch_sensors:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SensorServiceApi->watch_sensors: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  | 
 **list_options_label_selector** | **str**| A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. | [optional] 
 **list_options_field_selector** | **str**| A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. | [optional] 
 **list_options_watch** | **bool**| Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. | [optional] 
 **list_options_allow_watch_bookmarks** | **bool**| allowWatchBookmarks requests watch events with type \&quot;BOOKMARK\&quot;. Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server&#39;s discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. +optional. | [optional] 
 **list_options_resource_version** | **str**| resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional | [optional] 
 **list_options_resource_version_match** | **str**| resourceVersionMatch determines how resourceVersion is applied to list calls. It is highly recommended that resourceVersionMatch be set for list calls where resourceVersion is set See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional | [optional] 
 **list_options_timeout_seconds** | **str**| Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. | [optional] 
 **list_options_limit** | **str**| limit is a maximum number of responses to return for a list call. If more items exist, the server will set the &#x60;continue&#x60; field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. | [optional] 
 **list_options_continue** | **str**| The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the \&quot;next key\&quot;.  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. | [optional] 

### Return type

[**StreamResultOfSensorSensorWatchEvent**](StreamResultOfSensorSensorWatchEvent.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response.(streaming responses) |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

