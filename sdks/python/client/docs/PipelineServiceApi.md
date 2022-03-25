# argo_workflows.PipelineServiceApi

All URIs are relative to *http://localhost:2746*

Method | HTTP request | Description
------------- | ------------- | -------------
[**delete_pipeline**](PipelineServiceApi.md#delete_pipeline) | **DELETE** /api/v1/pipelines/{namespace}/{name} | 
[**get_pipeline**](PipelineServiceApi.md#get_pipeline) | **GET** /api/v1/pipelines/{namespace}/{name} | 
[**list_pipelines**](PipelineServiceApi.md#list_pipelines) | **GET** /api/v1/pipelines/{namespace} | 
[**pipeline_logs**](PipelineServiceApi.md#pipeline_logs) | **GET** /api/v1/stream/pipelines/{namespace}/logs | 
[**restart_pipeline**](PipelineServiceApi.md#restart_pipeline) | **POST** /api/v1/pipelines/{namespace}/{name}/restart | 
[**watch_pipelines**](PipelineServiceApi.md#watch_pipelines) | **GET** /api/v1/stream/pipelines/{namespace} | 
[**watch_steps**](PipelineServiceApi.md#watch_steps) | **GET** /api/v1/stream/steps/{namespace} | 


# **delete_pipeline**
> bool, date, datetime, dict, float, int, list, str, none_type delete_pipeline(namespace, name)



### Example

```python
import time
import argo_workflows
from argo_workflows.api import pipeline_service_api
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
    api_instance = pipeline_service_api.PipelineServiceApi(api_client)
    namespace = "namespace_example" # str | 
    name = "name_example" # str | 
    delete_options_grace_period_seconds = "deleteOptions.gracePeriodSeconds_example" # str | The duration in seconds before the object should be deleted. Value must be non-negative integer. The value zero indicates delete immediately. If this value is nil, the default grace period for the specified type will be used. Defaults to a per object value if not specified. zero means delete immediately. +optional. (optional)
    delete_options_preconditions_uid = "deleteOptions.preconditions.uid_example" # str | Specifies the target UID. +optional. (optional)
    delete_options_preconditions_resource_version = "deleteOptions.preconditions.resourceVersion_example" # str | Specifies the target ResourceVersion +optional. (optional)
    delete_options_orphan_dependents = True # bool | Deprecated: please use the PropagationPolicy, this field will be deprecated in 1.7. Should the dependent objects be orphaned. If true/false, the \"orphan\" finalizer will be added to/removed from the object's finalizers list. Either this field or PropagationPolicy may be set, but not both. +optional. (optional)
    delete_options_propagation_policy = "deleteOptions.propagationPolicy_example" # str | Whether and how garbage collection will be performed. Either this field or OrphanDependents may be set, but not both. The default policy is decided by the existing finalizer set in the metadata.finalizers and the resource-specific default policy. Acceptable values are: 'Orphan' - orphan the dependents; 'Background' - allow the garbage collector to delete the dependents in the background; 'Foreground' - a cascading policy that deletes all dependents in the foreground. +optional. (optional)
    delete_options_dry_run = [
        "deleteOptions.dryRun_example",
    ] # [str] | When present, indicates that modifications should not be persisted. An invalid or unrecognized dryRun directive will result in an error response and no further processing of the request. Valid values are: - All: all dry run stages will be processed +optional. (optional)

    # example passing only required values which don't have defaults set
    try:
        api_response = api_instance.delete_pipeline(namespace, name)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling PipelineServiceApi->delete_pipeline: %s\n" % e)

    # example passing only required values which don't have defaults set
    # and optional values
    try:
        api_response = api_instance.delete_pipeline(namespace, name, delete_options_grace_period_seconds=delete_options_grace_period_seconds, delete_options_preconditions_uid=delete_options_preconditions_uid, delete_options_preconditions_resource_version=delete_options_preconditions_resource_version, delete_options_orphan_dependents=delete_options_orphan_dependents, delete_options_propagation_policy=delete_options_propagation_policy, delete_options_dry_run=delete_options_dry_run)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling PipelineServiceApi->delete_pipeline: %s\n" % e)
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
 **delete_options_dry_run** | **[str]**| When present, indicates that modifications should not be persisted. An invalid or unrecognized dryRun directive will result in an error response and no further processing of the request. Valid values are: - All: all dry run stages will be processed +optional. | [optional]

### Return type

**bool, date, datetime, dict, float, int, list, str, none_type**

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

# **get_pipeline**
> GithubComArgoprojLabsArgoDataflowApiV1alpha1Pipeline get_pipeline(namespace, name)



### Example

```python
import time
import argo_workflows
from argo_workflows.api import pipeline_service_api
from argo_workflows.model.github_com_argoproj_labs_argo_dataflow_api_v1alpha1_pipeline import GithubComArgoprojLabsArgoDataflowApiV1alpha1Pipeline
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
    api_instance = pipeline_service_api.PipelineServiceApi(api_client)
    namespace = "namespace_example" # str | 
    name = "name_example" # str | 
    get_options_resource_version = "getOptions.resourceVersion_example" # str | resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional (optional)

    # example passing only required values which don't have defaults set
    try:
        api_response = api_instance.get_pipeline(namespace, name)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling PipelineServiceApi->get_pipeline: %s\n" % e)

    # example passing only required values which don't have defaults set
    # and optional values
    try:
        api_response = api_instance.get_pipeline(namespace, name, get_options_resource_version=get_options_resource_version)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling PipelineServiceApi->get_pipeline: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  |
 **name** | **str**|  |
 **get_options_resource_version** | **str**| resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional | [optional]

### Return type

[**GithubComArgoprojLabsArgoDataflowApiV1alpha1Pipeline**](GithubComArgoprojLabsArgoDataflowApiV1alpha1Pipeline.md)

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

# **list_pipelines**
> GithubComArgoprojLabsArgoDataflowApiV1alpha1PipelineList list_pipelines(namespace)



### Example

```python
import time
import argo_workflows
from argo_workflows.api import pipeline_service_api
from argo_workflows.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from argo_workflows.model.github_com_argoproj_labs_argo_dataflow_api_v1alpha1_pipeline_list import GithubComArgoprojLabsArgoDataflowApiV1alpha1PipelineList
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = argo_workflows.Configuration(
    host = "http://localhost:2746"
)


# Enter a context with an instance of the API client
with argo_workflows.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = pipeline_service_api.PipelineServiceApi(api_client)
    namespace = "namespace_example" # str | 
    list_options_label_selector = "listOptions.labelSelector_example" # str | A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. (optional)
    list_options_field_selector = "listOptions.fieldSelector_example" # str | A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. (optional)
    list_options_watch = True # bool | Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. (optional)
    list_options_allow_watch_bookmarks = True # bool | allowWatchBookmarks requests watch events with type \"BOOKMARK\". Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server's discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. +optional. (optional)
    list_options_resource_version = "listOptions.resourceVersion_example" # str | resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional (optional)
    list_options_resource_version_match = "listOptions.resourceVersionMatch_example" # str | resourceVersionMatch determines how resourceVersion is applied to list calls. It is highly recommended that resourceVersionMatch be set for list calls where resourceVersion is set See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional (optional)
    list_options_timeout_seconds = "listOptions.timeoutSeconds_example" # str | Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. (optional)
    list_options_limit = "listOptions.limit_example" # str | limit is a maximum number of responses to return for a list call. If more items exist, the server will set the `continue` field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. (optional)
    list_options_continue = "listOptions.continue_example" # str | The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the \"next key\".  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. (optional)

    # example passing only required values which don't have defaults set
    try:
        api_response = api_instance.list_pipelines(namespace)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling PipelineServiceApi->list_pipelines: %s\n" % e)

    # example passing only required values which don't have defaults set
    # and optional values
    try:
        api_response = api_instance.list_pipelines(namespace, list_options_label_selector=list_options_label_selector, list_options_field_selector=list_options_field_selector, list_options_watch=list_options_watch, list_options_allow_watch_bookmarks=list_options_allow_watch_bookmarks, list_options_resource_version=list_options_resource_version, list_options_resource_version_match=list_options_resource_version_match, list_options_timeout_seconds=list_options_timeout_seconds, list_options_limit=list_options_limit, list_options_continue=list_options_continue)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling PipelineServiceApi->list_pipelines: %s\n" % e)
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

[**GithubComArgoprojLabsArgoDataflowApiV1alpha1PipelineList**](GithubComArgoprojLabsArgoDataflowApiV1alpha1PipelineList.md)

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

# **pipeline_logs**
> StreamResultOfPipelineLogEntry pipeline_logs(namespace)



### Example

```python
import time
import argo_workflows
from argo_workflows.api import pipeline_service_api
from argo_workflows.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from argo_workflows.model.stream_result_of_pipeline_log_entry import StreamResultOfPipelineLogEntry
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = argo_workflows.Configuration(
    host = "http://localhost:2746"
)


# Enter a context with an instance of the API client
with argo_workflows.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = pipeline_service_api.PipelineServiceApi(api_client)
    namespace = "namespace_example" # str | 
    name = "name_example" # str | optional - only return entries for this pipeline. (optional)
    step_name = "stepName_example" # str | optional - only return entries for this step. (optional)
    grep = "grep_example" # str | optional - only return entries which match this expresssion. (optional)
    pod_log_options_container = "podLogOptions.container_example" # str | The container for which to stream logs. Defaults to only container if there is one container in the pod. +optional. (optional)
    pod_log_options_follow = True # bool | Follow the log stream of the pod. Defaults to false. +optional. (optional)
    pod_log_options_previous = True # bool | Return previous terminated container logs. Defaults to false. +optional. (optional)
    pod_log_options_since_seconds = "podLogOptions.sinceSeconds_example" # str | A relative time in seconds before the current time from which to show logs. If this value precedes the time a pod was started, only logs since the pod start will be returned. If this value is in the future, no logs will be returned. Only one of sinceSeconds or sinceTime may be specified. +optional. (optional)
    pod_log_options_since_time_seconds = "podLogOptions.sinceTime.seconds_example" # str | Represents seconds of UTC time since Unix epoch 1970-01-01T00:00:00Z. Must be from 0001-01-01T00:00:00Z to 9999-12-31T23:59:59Z inclusive. (optional)
    pod_log_options_since_time_nanos = 1 # int | Non-negative fractions of a second at nanosecond resolution. Negative second values with fractions must still have non-negative nanos values that count forward in time. Must be from 0 to 999,999,999 inclusive. This field may be limited in precision depending on context. (optional)
    pod_log_options_timestamps = True # bool | If true, add an RFC3339 or RFC3339Nano timestamp at the beginning of every line of log output. Defaults to false. +optional. (optional)
    pod_log_options_tail_lines = "podLogOptions.tailLines_example" # str | If set, the number of lines from the end of the logs to show. If not specified, logs are shown from the creation of the container or sinceSeconds or sinceTime +optional. (optional)
    pod_log_options_limit_bytes = "podLogOptions.limitBytes_example" # str | If set, the number of bytes to read from the server before terminating the log output. This may not display a complete final line of logging, and may return slightly more or slightly less than the specified limit. +optional. (optional)
    pod_log_options_insecure_skip_tls_verify_backend = True # bool | insecureSkipTLSVerifyBackend indicates that the apiserver should not confirm the validity of the serving certificate of the backend it is connecting to.  This will make the HTTPS connection between the apiserver and the backend insecure. This means the apiserver cannot verify the log data it is receiving came from the real kubelet.  If the kubelet is configured to verify the apiserver's TLS credentials, it does not mean the connection to the real kubelet is vulnerable to a man in the middle attack (e.g. an attacker could not intercept the actual log data coming from the real kubelet). +optional. (optional)

    # example passing only required values which don't have defaults set
    try:
        api_response = api_instance.pipeline_logs(namespace)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_logs: %s\n" % e)

    # example passing only required values which don't have defaults set
    # and optional values
    try:
        api_response = api_instance.pipeline_logs(namespace, name=name, step_name=step_name, grep=grep, pod_log_options_container=pod_log_options_container, pod_log_options_follow=pod_log_options_follow, pod_log_options_previous=pod_log_options_previous, pod_log_options_since_seconds=pod_log_options_since_seconds, pod_log_options_since_time_seconds=pod_log_options_since_time_seconds, pod_log_options_since_time_nanos=pod_log_options_since_time_nanos, pod_log_options_timestamps=pod_log_options_timestamps, pod_log_options_tail_lines=pod_log_options_tail_lines, pod_log_options_limit_bytes=pod_log_options_limit_bytes, pod_log_options_insecure_skip_tls_verify_backend=pod_log_options_insecure_skip_tls_verify_backend)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_logs: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  |
 **name** | **str**| optional - only return entries for this pipeline. | [optional]
 **step_name** | **str**| optional - only return entries for this step. | [optional]
 **grep** | **str**| optional - only return entries which match this expresssion. | [optional]
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

[**StreamResultOfPipelineLogEntry**](StreamResultOfPipelineLogEntry.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response.(streaming responses) |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **restart_pipeline**
> bool, date, datetime, dict, float, int, list, str, none_type restart_pipeline(namespace, name)



### Example

```python
import time
import argo_workflows
from argo_workflows.api import pipeline_service_api
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
    api_instance = pipeline_service_api.PipelineServiceApi(api_client)
    namespace = "namespace_example" # str | 
    name = "name_example" # str | 

    # example passing only required values which don't have defaults set
    try:
        api_response = api_instance.restart_pipeline(namespace, name)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling PipelineServiceApi->restart_pipeline: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  |
 **name** | **str**|  |

### Return type

**bool, date, datetime, dict, float, int, list, str, none_type**

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

# **watch_pipelines**
> StreamResultOfPipelinePipelineWatchEvent watch_pipelines(namespace)



### Example

```python
import time
import argo_workflows
from argo_workflows.api import pipeline_service_api
from argo_workflows.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from argo_workflows.model.stream_result_of_pipeline_pipeline_watch_event import StreamResultOfPipelinePipelineWatchEvent
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = argo_workflows.Configuration(
    host = "http://localhost:2746"
)


# Enter a context with an instance of the API client
with argo_workflows.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = pipeline_service_api.PipelineServiceApi(api_client)
    namespace = "namespace_example" # str | 
    list_options_label_selector = "listOptions.labelSelector_example" # str | A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. (optional)
    list_options_field_selector = "listOptions.fieldSelector_example" # str | A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. (optional)
    list_options_watch = True # bool | Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. (optional)
    list_options_allow_watch_bookmarks = True # bool | allowWatchBookmarks requests watch events with type \"BOOKMARK\". Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server's discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. +optional. (optional)
    list_options_resource_version = "listOptions.resourceVersion_example" # str | resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional (optional)
    list_options_resource_version_match = "listOptions.resourceVersionMatch_example" # str | resourceVersionMatch determines how resourceVersion is applied to list calls. It is highly recommended that resourceVersionMatch be set for list calls where resourceVersion is set See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional (optional)
    list_options_timeout_seconds = "listOptions.timeoutSeconds_example" # str | Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. (optional)
    list_options_limit = "listOptions.limit_example" # str | limit is a maximum number of responses to return for a list call. If more items exist, the server will set the `continue` field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. (optional)
    list_options_continue = "listOptions.continue_example" # str | The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the \"next key\".  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. (optional)

    # example passing only required values which don't have defaults set
    try:
        api_response = api_instance.watch_pipelines(namespace)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling PipelineServiceApi->watch_pipelines: %s\n" % e)

    # example passing only required values which don't have defaults set
    # and optional values
    try:
        api_response = api_instance.watch_pipelines(namespace, list_options_label_selector=list_options_label_selector, list_options_field_selector=list_options_field_selector, list_options_watch=list_options_watch, list_options_allow_watch_bookmarks=list_options_allow_watch_bookmarks, list_options_resource_version=list_options_resource_version, list_options_resource_version_match=list_options_resource_version_match, list_options_timeout_seconds=list_options_timeout_seconds, list_options_limit=list_options_limit, list_options_continue=list_options_continue)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling PipelineServiceApi->watch_pipelines: %s\n" % e)
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

[**StreamResultOfPipelinePipelineWatchEvent**](StreamResultOfPipelinePipelineWatchEvent.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response.(streaming responses) |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **watch_steps**
> StreamResultOfPipelineStepWatchEvent watch_steps(namespace)



### Example

```python
import time
import argo_workflows
from argo_workflows.api import pipeline_service_api
from argo_workflows.model.grpc_gateway_runtime_error import GrpcGatewayRuntimeError
from argo_workflows.model.stream_result_of_pipeline_step_watch_event import StreamResultOfPipelineStepWatchEvent
from pprint import pprint
# Defining the host is optional and defaults to http://localhost:2746
# See configuration.py for a list of all supported configuration parameters.
configuration = argo_workflows.Configuration(
    host = "http://localhost:2746"
)


# Enter a context with an instance of the API client
with argo_workflows.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = pipeline_service_api.PipelineServiceApi(api_client)
    namespace = "namespace_example" # str | 
    list_options_label_selector = "listOptions.labelSelector_example" # str | A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. (optional)
    list_options_field_selector = "listOptions.fieldSelector_example" # str | A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. (optional)
    list_options_watch = True # bool | Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. (optional)
    list_options_allow_watch_bookmarks = True # bool | allowWatchBookmarks requests watch events with type \"BOOKMARK\". Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server's discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. +optional. (optional)
    list_options_resource_version = "listOptions.resourceVersion_example" # str | resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional (optional)
    list_options_resource_version_match = "listOptions.resourceVersionMatch_example" # str | resourceVersionMatch determines how resourceVersion is applied to list calls. It is highly recommended that resourceVersionMatch be set for list calls where resourceVersion is set See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional (optional)
    list_options_timeout_seconds = "listOptions.timeoutSeconds_example" # str | Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. (optional)
    list_options_limit = "listOptions.limit_example" # str | limit is a maximum number of responses to return for a list call. If more items exist, the server will set the `continue` field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. (optional)
    list_options_continue = "listOptions.continue_example" # str | The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the \"next key\".  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. (optional)

    # example passing only required values which don't have defaults set
    try:
        api_response = api_instance.watch_steps(namespace)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling PipelineServiceApi->watch_steps: %s\n" % e)

    # example passing only required values which don't have defaults set
    # and optional values
    try:
        api_response = api_instance.watch_steps(namespace, list_options_label_selector=list_options_label_selector, list_options_field_selector=list_options_field_selector, list_options_watch=list_options_watch, list_options_allow_watch_bookmarks=list_options_allow_watch_bookmarks, list_options_resource_version=list_options_resource_version, list_options_resource_version_match=list_options_resource_version_match, list_options_timeout_seconds=list_options_timeout_seconds, list_options_limit=list_options_limit, list_options_continue=list_options_continue)
        pprint(api_response)
    except argo_workflows.ApiException as e:
        print("Exception when calling PipelineServiceApi->watch_steps: %s\n" % e)
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

[**StreamResultOfPipelineStepWatchEvent**](StreamResultOfPipelineStepWatchEvent.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response.(streaming responses) |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

