# \ArchivedWorkflowServiceApi

All URIs are relative to *http://localhost:2746*

Method | HTTP request | Description
------------- | ------------- | -------------
[**delete_archived_workflow**](ArchivedWorkflowServiceApi.md#delete_archived_workflow) | **DELETE** /api/v1/archived-workflows/{uid} | 
[**get_archived_workflow**](ArchivedWorkflowServiceApi.md#get_archived_workflow) | **GET** /api/v1/archived-workflows/{uid} | 
[**list_archived_workflow_label_keys**](ArchivedWorkflowServiceApi.md#list_archived_workflow_label_keys) | **GET** /api/v1/archived-workflows-label-keys | 
[**list_archived_workflow_label_values**](ArchivedWorkflowServiceApi.md#list_archived_workflow_label_values) | **GET** /api/v1/archived-workflows-label-values | 
[**list_archived_workflows**](ArchivedWorkflowServiceApi.md#list_archived_workflows) | **GET** /api/v1/archived-workflows | 
[**resubmit_archived_workflow**](ArchivedWorkflowServiceApi.md#resubmit_archived_workflow) | **PUT** /api/v1/archived-workflows/{uid}/resubmit | 
[**retry_archived_workflow**](ArchivedWorkflowServiceApi.md#retry_archived_workflow) | **PUT** /api/v1/archived-workflows/{uid}/retry | 



## delete_archived_workflow

> serde_json::Value delete_archived_workflow(uid, namespace)


### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**uid** | **String** |  | [required] |
**namespace** | Option<**String**> |  |  |

### Return type

[**serde_json::Value**](serde_json::Value.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## get_archived_workflow

> crate::models::IoArgoprojWorkflowV1alpha1Workflow get_archived_workflow(uid, namespace, name)


### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**uid** | **String** |  | [required] |
**namespace** | Option<**String**> |  |  |
**name** | Option<**String**> |  |  |

### Return type

[**crate::models::IoArgoprojWorkflowV1alpha1Workflow**](io.argoproj.workflow.v1alpha1.Workflow.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## list_archived_workflow_label_keys

> crate::models::IoArgoprojWorkflowV1alpha1LabelKeys list_archived_workflow_label_keys(namespace)


### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**namespace** | Option<**String**> |  |  |

### Return type

[**crate::models::IoArgoprojWorkflowV1alpha1LabelKeys**](io.argoproj.workflow.v1alpha1.LabelKeys.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## list_archived_workflow_label_values

> crate::models::IoArgoprojWorkflowV1alpha1LabelValues list_archived_workflow_label_values(list_options_label_selector, list_options_field_selector, list_options_watch, list_options_allow_watch_bookmarks, list_options_resource_version, list_options_resource_version_match, list_options_timeout_seconds, list_options_limit, list_options_continue, namespace)


### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**list_options_label_selector** | Option<**String**> | A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. |  |
**list_options_field_selector** | Option<**String**> | A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. |  |
**list_options_watch** | Option<**bool**> | Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. |  |
**list_options_allow_watch_bookmarks** | Option<**bool**> | allowWatchBookmarks requests watch events with type \"BOOKMARK\". Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server's discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. +optional. |  |
**list_options_resource_version** | Option<**String**> | resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional |  |
**list_options_resource_version_match** | Option<**String**> | resourceVersionMatch determines how resourceVersion is applied to list calls. It is highly recommended that resourceVersionMatch be set for list calls where resourceVersion is set See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional |  |
**list_options_timeout_seconds** | Option<**String**> | Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. |  |
**list_options_limit** | Option<**String**> | limit is a maximum number of responses to return for a list call. If more items exist, the server will set the `continue` field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. |  |
**list_options_continue** | Option<**String**> | The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the \"next key\".  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. |  |
**namespace** | Option<**String**> |  |  |

### Return type

[**crate::models::IoArgoprojWorkflowV1alpha1LabelValues**](io.argoproj.workflow.v1alpha1.LabelValues.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## list_archived_workflows

> crate::models::IoArgoprojWorkflowV1alpha1WorkflowList list_archived_workflows(list_options_label_selector, list_options_field_selector, list_options_watch, list_options_allow_watch_bookmarks, list_options_resource_version, list_options_resource_version_match, list_options_timeout_seconds, list_options_limit, list_options_continue, name_prefix, namespace)


### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**list_options_label_selector** | Option<**String**> | A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. |  |
**list_options_field_selector** | Option<**String**> | A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. |  |
**list_options_watch** | Option<**bool**> | Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. |  |
**list_options_allow_watch_bookmarks** | Option<**bool**> | allowWatchBookmarks requests watch events with type \"BOOKMARK\". Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server's discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. +optional. |  |
**list_options_resource_version** | Option<**String**> | resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional |  |
**list_options_resource_version_match** | Option<**String**> | resourceVersionMatch determines how resourceVersion is applied to list calls. It is highly recommended that resourceVersionMatch be set for list calls where resourceVersion is set See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional |  |
**list_options_timeout_seconds** | Option<**String**> | Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. |  |
**list_options_limit** | Option<**String**> | limit is a maximum number of responses to return for a list call. If more items exist, the server will set the `continue` field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. |  |
**list_options_continue** | Option<**String**> | The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the \"next key\".  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. |  |
**name_prefix** | Option<**String**> |  |  |
**namespace** | Option<**String**> |  |  |

### Return type

[**crate::models::IoArgoprojWorkflowV1alpha1WorkflowList**](io.argoproj.workflow.v1alpha1.WorkflowList.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## resubmit_archived_workflow

> crate::models::IoArgoprojWorkflowV1alpha1Workflow resubmit_archived_workflow(uid, body)


### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**uid** | **String** |  | [required] |
**body** | [**IoArgoprojWorkflowV1alpha1ResubmitArchivedWorkflowRequest**](IoArgoprojWorkflowV1alpha1ResubmitArchivedWorkflowRequest.md) |  | [required] |

### Return type

[**crate::models::IoArgoprojWorkflowV1alpha1Workflow**](io.argoproj.workflow.v1alpha1.Workflow.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## retry_archived_workflow

> crate::models::IoArgoprojWorkflowV1alpha1Workflow retry_archived_workflow(uid, body)


### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**uid** | **String** |  | [required] |
**body** | [**IoArgoprojWorkflowV1alpha1RetryArchivedWorkflowRequest**](IoArgoprojWorkflowV1alpha1RetryArchivedWorkflowRequest.md) |  | [required] |

### Return type

[**crate::models::IoArgoprojWorkflowV1alpha1Workflow**](io.argoproj.workflow.v1alpha1.Workflow.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

