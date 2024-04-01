# argo_workflows.CronWorkflowServiceApi

All URIs are relative to *http://localhost:2746*

Method | HTTP request | Description
------------- | ------------- | -------------
[**create_cron_workflow**](CronWorkflowServiceApi.md#create_cron_workflow) | **POST** /api/v1/cron-workflows/{namespace} | 
[**delete_cron_workflow**](CronWorkflowServiceApi.md#delete_cron_workflow) | **DELETE** /api/v1/cron-workflows/{namespace}/{name} | 
[**get_cron_workflow**](CronWorkflowServiceApi.md#get_cron_workflow) | **GET** /api/v1/cron-workflows/{namespace}/{name} | 
[**lint_cron_workflow**](CronWorkflowServiceApi.md#lint_cron_workflow) | **POST** /api/v1/cron-workflows/{namespace}/lint | 
[**list_cron_workflows**](CronWorkflowServiceApi.md#list_cron_workflows) | **GET** /api/v1/cron-workflows/{namespace} | 
[**resume_cron_workflow**](CronWorkflowServiceApi.md#resume_cron_workflow) | **PUT** /api/v1/cron-workflows/{namespace}/{name}/resume | 
[**suspend_cron_workflow**](CronWorkflowServiceApi.md#suspend_cron_workflow) | **PUT** /api/v1/cron-workflows/{namespace}/{name}/suspend | 
[**update_cron_workflow**](CronWorkflowServiceApi.md#update_cron_workflow) | **PUT** /api/v1/cron-workflows/{namespace}/{name} | 


# **create_cron_workflow**
> IoArgoprojWorkflowV1alpha1CronWorkflow create_cron_workflow(namespace, body)



### Example

* Api Key Authentication (BearerToken):

```python
import argo_workflows
from argo_workflows.models.io_argoproj_workflow_v1alpha1_create_cron_workflow_request import IoArgoprojWorkflowV1alpha1CreateCronWorkflowRequest
from argo_workflows.models.io_argoproj_workflow_v1alpha1_cron_workflow import IoArgoprojWorkflowV1alpha1CronWorkflow
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
    api_instance = argo_workflows.CronWorkflowServiceApi(api_client)
    namespace = 'namespace_example' # str | 
    body = argo_workflows.IoArgoprojWorkflowV1alpha1CreateCronWorkflowRequest() # IoArgoprojWorkflowV1alpha1CreateCronWorkflowRequest | 

    try:
        api_response = api_instance.create_cron_workflow(namespace, body)
        print("The response of CronWorkflowServiceApi->create_cron_workflow:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling CronWorkflowServiceApi->create_cron_workflow: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  | 
 **body** | [**IoArgoprojWorkflowV1alpha1CreateCronWorkflowRequest**](IoArgoprojWorkflowV1alpha1CreateCronWorkflowRequest.md)|  | 

### Return type

[**IoArgoprojWorkflowV1alpha1CronWorkflow**](IoArgoprojWorkflowV1alpha1CronWorkflow.md)

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

# **delete_cron_workflow**
> object delete_cron_workflow(namespace, name, delete_options_grace_period_seconds=delete_options_grace_period_seconds, delete_options_preconditions_uid=delete_options_preconditions_uid, delete_options_preconditions_resource_version=delete_options_preconditions_resource_version, delete_options_orphan_dependents=delete_options_orphan_dependents, delete_options_propagation_policy=delete_options_propagation_policy, delete_options_dry_run=delete_options_dry_run)



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
    api_instance = argo_workflows.CronWorkflowServiceApi(api_client)
    namespace = 'namespace_example' # str | 
    name = 'name_example' # str | 
    delete_options_grace_period_seconds = 'delete_options_grace_period_seconds_example' # str | The duration in seconds before the object should be deleted. Value must be non-negative integer. The value zero indicates delete immediately. If this value is nil, the default grace period for the specified type will be used. Defaults to a per object value if not specified. zero means delete immediately. +optional. (optional)
    delete_options_preconditions_uid = 'delete_options_preconditions_uid_example' # str | Specifies the target UID. +optional. (optional)
    delete_options_preconditions_resource_version = 'delete_options_preconditions_resource_version_example' # str | Specifies the target ResourceVersion +optional. (optional)
    delete_options_orphan_dependents = True # bool | Deprecated: please use the PropagationPolicy, this field will be deprecated in 1.7. Should the dependent objects be orphaned. If true/false, the \"orphan\" finalizer will be added to/removed from the object's finalizers list. Either this field or PropagationPolicy may be set, but not both. +optional. (optional)
    delete_options_propagation_policy = 'delete_options_propagation_policy_example' # str | Whether and how garbage collection will be performed. Either this field or OrphanDependents may be set, but not both. The default policy is decided by the existing finalizer set in the metadata.finalizers and the resource-specific default policy. Acceptable values are: 'Orphan' - orphan the dependents; 'Background' - allow the garbage collector to delete the dependents in the background; 'Foreground' - a cascading policy that deletes all dependents in the foreground. +optional. (optional)
    delete_options_dry_run = ['delete_options_dry_run_example'] # List[str] | When present, indicates that modifications should not be persisted. An invalid or unrecognized dryRun directive will result in an error response and no further processing of the request. Valid values are: - All: all dry run stages will be processed +optional. (optional)

    try:
        api_response = api_instance.delete_cron_workflow(namespace, name, delete_options_grace_period_seconds=delete_options_grace_period_seconds, delete_options_preconditions_uid=delete_options_preconditions_uid, delete_options_preconditions_resource_version=delete_options_preconditions_resource_version, delete_options_orphan_dependents=delete_options_orphan_dependents, delete_options_propagation_policy=delete_options_propagation_policy, delete_options_dry_run=delete_options_dry_run)
        print("The response of CronWorkflowServiceApi->delete_cron_workflow:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling CronWorkflowServiceApi->delete_cron_workflow: %s\n" % e)
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

# **get_cron_workflow**
> IoArgoprojWorkflowV1alpha1CronWorkflow get_cron_workflow(namespace, name, get_options_resource_version=get_options_resource_version)



### Example

* Api Key Authentication (BearerToken):

```python
import argo_workflows
from argo_workflows.models.io_argoproj_workflow_v1alpha1_cron_workflow import IoArgoprojWorkflowV1alpha1CronWorkflow
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
    api_instance = argo_workflows.CronWorkflowServiceApi(api_client)
    namespace = 'namespace_example' # str | 
    name = 'name_example' # str | 
    get_options_resource_version = 'get_options_resource_version_example' # str | resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional (optional)

    try:
        api_response = api_instance.get_cron_workflow(namespace, name, get_options_resource_version=get_options_resource_version)
        print("The response of CronWorkflowServiceApi->get_cron_workflow:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling CronWorkflowServiceApi->get_cron_workflow: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  | 
 **name** | **str**|  | 
 **get_options_resource_version** | **str**| resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional | [optional] 

### Return type

[**IoArgoprojWorkflowV1alpha1CronWorkflow**](IoArgoprojWorkflowV1alpha1CronWorkflow.md)

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

# **lint_cron_workflow**
> IoArgoprojWorkflowV1alpha1CronWorkflow lint_cron_workflow(namespace, body)



### Example

* Api Key Authentication (BearerToken):

```python
import argo_workflows
from argo_workflows.models.io_argoproj_workflow_v1alpha1_cron_workflow import IoArgoprojWorkflowV1alpha1CronWorkflow
from argo_workflows.models.io_argoproj_workflow_v1alpha1_lint_cron_workflow_request import IoArgoprojWorkflowV1alpha1LintCronWorkflowRequest
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
    api_instance = argo_workflows.CronWorkflowServiceApi(api_client)
    namespace = 'namespace_example' # str | 
    body = argo_workflows.IoArgoprojWorkflowV1alpha1LintCronWorkflowRequest() # IoArgoprojWorkflowV1alpha1LintCronWorkflowRequest | 

    try:
        api_response = api_instance.lint_cron_workflow(namespace, body)
        print("The response of CronWorkflowServiceApi->lint_cron_workflow:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling CronWorkflowServiceApi->lint_cron_workflow: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  | 
 **body** | [**IoArgoprojWorkflowV1alpha1LintCronWorkflowRequest**](IoArgoprojWorkflowV1alpha1LintCronWorkflowRequest.md)|  | 

### Return type

[**IoArgoprojWorkflowV1alpha1CronWorkflow**](IoArgoprojWorkflowV1alpha1CronWorkflow.md)

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

# **list_cron_workflows**
> IoArgoprojWorkflowV1alpha1CronWorkflowList list_cron_workflows(namespace, list_options_label_selector=list_options_label_selector, list_options_field_selector=list_options_field_selector, list_options_watch=list_options_watch, list_options_allow_watch_bookmarks=list_options_allow_watch_bookmarks, list_options_resource_version=list_options_resource_version, list_options_resource_version_match=list_options_resource_version_match, list_options_timeout_seconds=list_options_timeout_seconds, list_options_limit=list_options_limit, list_options_continue=list_options_continue)



### Example

* Api Key Authentication (BearerToken):

```python
import argo_workflows
from argo_workflows.models.io_argoproj_workflow_v1alpha1_cron_workflow_list import IoArgoprojWorkflowV1alpha1CronWorkflowList
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
    api_instance = argo_workflows.CronWorkflowServiceApi(api_client)
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
        api_response = api_instance.list_cron_workflows(namespace, list_options_label_selector=list_options_label_selector, list_options_field_selector=list_options_field_selector, list_options_watch=list_options_watch, list_options_allow_watch_bookmarks=list_options_allow_watch_bookmarks, list_options_resource_version=list_options_resource_version, list_options_resource_version_match=list_options_resource_version_match, list_options_timeout_seconds=list_options_timeout_seconds, list_options_limit=list_options_limit, list_options_continue=list_options_continue)
        print("The response of CronWorkflowServiceApi->list_cron_workflows:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling CronWorkflowServiceApi->list_cron_workflows: %s\n" % e)
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

[**IoArgoprojWorkflowV1alpha1CronWorkflowList**](IoArgoprojWorkflowV1alpha1CronWorkflowList.md)

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

# **resume_cron_workflow**
> IoArgoprojWorkflowV1alpha1CronWorkflow resume_cron_workflow(namespace, name, body)



### Example

* Api Key Authentication (BearerToken):

```python
import argo_workflows
from argo_workflows.models.io_argoproj_workflow_v1alpha1_cron_workflow import IoArgoprojWorkflowV1alpha1CronWorkflow
from argo_workflows.models.io_argoproj_workflow_v1alpha1_cron_workflow_resume_request import IoArgoprojWorkflowV1alpha1CronWorkflowResumeRequest
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
    api_instance = argo_workflows.CronWorkflowServiceApi(api_client)
    namespace = 'namespace_example' # str | 
    name = 'name_example' # str | 
    body = argo_workflows.IoArgoprojWorkflowV1alpha1CronWorkflowResumeRequest() # IoArgoprojWorkflowV1alpha1CronWorkflowResumeRequest | 

    try:
        api_response = api_instance.resume_cron_workflow(namespace, name, body)
        print("The response of CronWorkflowServiceApi->resume_cron_workflow:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling CronWorkflowServiceApi->resume_cron_workflow: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  | 
 **name** | **str**|  | 
 **body** | [**IoArgoprojWorkflowV1alpha1CronWorkflowResumeRequest**](IoArgoprojWorkflowV1alpha1CronWorkflowResumeRequest.md)|  | 

### Return type

[**IoArgoprojWorkflowV1alpha1CronWorkflow**](IoArgoprojWorkflowV1alpha1CronWorkflow.md)

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

# **suspend_cron_workflow**
> IoArgoprojWorkflowV1alpha1CronWorkflow suspend_cron_workflow(namespace, name, body)



### Example

* Api Key Authentication (BearerToken):

```python
import argo_workflows
from argo_workflows.models.io_argoproj_workflow_v1alpha1_cron_workflow import IoArgoprojWorkflowV1alpha1CronWorkflow
from argo_workflows.models.io_argoproj_workflow_v1alpha1_cron_workflow_suspend_request import IoArgoprojWorkflowV1alpha1CronWorkflowSuspendRequest
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
    api_instance = argo_workflows.CronWorkflowServiceApi(api_client)
    namespace = 'namespace_example' # str | 
    name = 'name_example' # str | 
    body = argo_workflows.IoArgoprojWorkflowV1alpha1CronWorkflowSuspendRequest() # IoArgoprojWorkflowV1alpha1CronWorkflowSuspendRequest | 

    try:
        api_response = api_instance.suspend_cron_workflow(namespace, name, body)
        print("The response of CronWorkflowServiceApi->suspend_cron_workflow:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling CronWorkflowServiceApi->suspend_cron_workflow: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  | 
 **name** | **str**|  | 
 **body** | [**IoArgoprojWorkflowV1alpha1CronWorkflowSuspendRequest**](IoArgoprojWorkflowV1alpha1CronWorkflowSuspendRequest.md)|  | 

### Return type

[**IoArgoprojWorkflowV1alpha1CronWorkflow**](IoArgoprojWorkflowV1alpha1CronWorkflow.md)

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

# **update_cron_workflow**
> IoArgoprojWorkflowV1alpha1CronWorkflow update_cron_workflow(namespace, name, body)



### Example

* Api Key Authentication (BearerToken):

```python
import argo_workflows
from argo_workflows.models.io_argoproj_workflow_v1alpha1_cron_workflow import IoArgoprojWorkflowV1alpha1CronWorkflow
from argo_workflows.models.io_argoproj_workflow_v1alpha1_update_cron_workflow_request import IoArgoprojWorkflowV1alpha1UpdateCronWorkflowRequest
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
    api_instance = argo_workflows.CronWorkflowServiceApi(api_client)
    namespace = 'namespace_example' # str | 
    name = 'name_example' # str | DEPRECATED: This field is ignored.
    body = argo_workflows.IoArgoprojWorkflowV1alpha1UpdateCronWorkflowRequest() # IoArgoprojWorkflowV1alpha1UpdateCronWorkflowRequest | 

    try:
        api_response = api_instance.update_cron_workflow(namespace, name, body)
        print("The response of CronWorkflowServiceApi->update_cron_workflow:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling CronWorkflowServiceApi->update_cron_workflow: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  | 
 **name** | **str**| DEPRECATED: This field is ignored. | 
 **body** | [**IoArgoprojWorkflowV1alpha1UpdateCronWorkflowRequest**](IoArgoprojWorkflowV1alpha1UpdateCronWorkflowRequest.md)|  | 

### Return type

[**IoArgoprojWorkflowV1alpha1CronWorkflow**](IoArgoprojWorkflowV1alpha1CronWorkflow.md)

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

