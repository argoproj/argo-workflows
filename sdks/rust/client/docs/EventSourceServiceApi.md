# \EventSourceServiceApi

All URIs are relative to *http://localhost:2746*

Method | HTTP request | Description
------------- | ------------- | -------------
[**create_event_source**](EventSourceServiceApi.md#create_event_source) | **POST** /api/v1/event-sources/{namespace} | 
[**delete_event_source**](EventSourceServiceApi.md#delete_event_source) | **DELETE** /api/v1/event-sources/{namespace}/{name} | 
[**event_sources_logs**](EventSourceServiceApi.md#event_sources_logs) | **GET** /api/v1/stream/event-sources/{namespace}/logs | 
[**get_event_source**](EventSourceServiceApi.md#get_event_source) | **GET** /api/v1/event-sources/{namespace}/{name} | 
[**list_event_sources**](EventSourceServiceApi.md#list_event_sources) | **GET** /api/v1/event-sources/{namespace} | 
[**update_event_source**](EventSourceServiceApi.md#update_event_source) | **PUT** /api/v1/event-sources/{namespace}/{name} | 
[**watch_event_sources**](EventSourceServiceApi.md#watch_event_sources) | **GET** /api/v1/stream/event-sources/{namespace} | 



## create_event_source

> crate::models::IoArgoprojEventsV1alpha1EventSource create_event_source(namespace, body)


### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**namespace** | **String** |  | [required] |
**body** | [**EventsourceCreateEventSourceRequest**](EventsourceCreateEventSourceRequest.md) |  | [required] |

### Return type

[**crate::models::IoArgoprojEventsV1alpha1EventSource**](io.argoproj.events.v1alpha1.EventSource.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## delete_event_source

> serde_json::Value delete_event_source(namespace, name, delete_options_grace_period_seconds, delete_options_preconditions_uid, delete_options_preconditions_resource_version, delete_options_orphan_dependents, delete_options_propagation_policy, delete_options_dry_run)


### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**namespace** | **String** |  | [required] |
**name** | **String** |  | [required] |
**delete_options_grace_period_seconds** | Option<**String**> | The duration in seconds before the object should be deleted. Value must be non-negative integer. The value zero indicates delete immediately. If this value is nil, the default grace period for the specified type will be used. Defaults to a per object value if not specified. zero means delete immediately. +optional. |  |
**delete_options_preconditions_uid** | Option<**String**> | Specifies the target UID. +optional. |  |
**delete_options_preconditions_resource_version** | Option<**String**> | Specifies the target ResourceVersion +optional. |  |
**delete_options_orphan_dependents** | Option<**bool**> | Deprecated: please use the PropagationPolicy, this field will be deprecated in 1.7. Should the dependent objects be orphaned. If true/false, the \"orphan\" finalizer will be added to/removed from the object's finalizers list. Either this field or PropagationPolicy may be set, but not both. +optional. |  |
**delete_options_propagation_policy** | Option<**String**> | Whether and how garbage collection will be performed. Either this field or OrphanDependents may be set, but not both. The default policy is decided by the existing finalizer set in the metadata.finalizers and the resource-specific default policy. Acceptable values are: 'Orphan' - orphan the dependents; 'Background' - allow the garbage collector to delete the dependents in the background; 'Foreground' - a cascading policy that deletes all dependents in the foreground. +optional. |  |
**delete_options_dry_run** | Option<[**Vec<String>**](String.md)> | When present, indicates that modifications should not be persisted. An invalid or unrecognized dryRun directive will result in an error response and no further processing of the request. Valid values are: - All: all dry run stages will be processed +optional. |  |

### Return type

[**serde_json::Value**](serde_json::Value.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## event_sources_logs

> crate::models::StreamResultOfEventsourceLogEntry event_sources_logs(namespace, name, event_source_type, event_name, grep, pod_log_options_container, pod_log_options_follow, pod_log_options_previous, pod_log_options_since_seconds, pod_log_options_since_time_seconds, pod_log_options_since_time_nanos, pod_log_options_timestamps, pod_log_options_tail_lines, pod_log_options_limit_bytes, pod_log_options_insecure_skip_tls_verify_backend)


### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**namespace** | **String** |  | [required] |
**name** | Option<**String**> | optional - only return entries for this event source. |  |
**event_source_type** | Option<**String**> | optional - only return entries for this event source type (e.g. `webhook`). |  |
**event_name** | Option<**String**> | optional - only return entries for this event name (e.g. `example`). |  |
**grep** | Option<**String**> | optional - only return entries where `msg` matches this regular expression. |  |
**pod_log_options_container** | Option<**String**> | The container for which to stream logs. Defaults to only container if there is one container in the pod. +optional. |  |
**pod_log_options_follow** | Option<**bool**> | Follow the log stream of the pod. Defaults to false. +optional. |  |
**pod_log_options_previous** | Option<**bool**> | Return previous terminated container logs. Defaults to false. +optional. |  |
**pod_log_options_since_seconds** | Option<**String**> | A relative time in seconds before the current time from which to show logs. If this value precedes the time a pod was started, only logs since the pod start will be returned. If this value is in the future, no logs will be returned. Only one of sinceSeconds or sinceTime may be specified. +optional. |  |
**pod_log_options_since_time_seconds** | Option<**String**> | Represents seconds of UTC time since Unix epoch 1970-01-01T00:00:00Z. Must be from 0001-01-01T00:00:00Z to 9999-12-31T23:59:59Z inclusive. |  |
**pod_log_options_since_time_nanos** | Option<**i32**> | Non-negative fractions of a second at nanosecond resolution. Negative second values with fractions must still have non-negative nanos values that count forward in time. Must be from 0 to 999,999,999 inclusive. This field may be limited in precision depending on context. |  |
**pod_log_options_timestamps** | Option<**bool**> | If true, add an RFC3339 or RFC3339Nano timestamp at the beginning of every line of log output. Defaults to false. +optional. |  |
**pod_log_options_tail_lines** | Option<**String**> | If set, the number of lines from the end of the logs to show. If not specified, logs are shown from the creation of the container or sinceSeconds or sinceTime +optional. |  |
**pod_log_options_limit_bytes** | Option<**String**> | If set, the number of bytes to read from the server before terminating the log output. This may not display a complete final line of logging, and may return slightly more or slightly less than the specified limit. +optional. |  |
**pod_log_options_insecure_skip_tls_verify_backend** | Option<**bool**> | insecureSkipTLSVerifyBackend indicates that the apiserver should not confirm the validity of the serving certificate of the backend it is connecting to.  This will make the HTTPS connection between the apiserver and the backend insecure. This means the apiserver cannot verify the log data it is receiving came from the real kubelet.  If the kubelet is configured to verify the apiserver's TLS credentials, it does not mean the connection to the real kubelet is vulnerable to a man in the middle attack (e.g. an attacker could not intercept the actual log data coming from the real kubelet). +optional. |  |

### Return type

[**crate::models::StreamResultOfEventsourceLogEntry**](Stream_result_of_eventsource_LogEntry.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## get_event_source

> crate::models::IoArgoprojEventsV1alpha1EventSource get_event_source(namespace, name)


### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**namespace** | **String** |  | [required] |
**name** | **String** |  | [required] |

### Return type

[**crate::models::IoArgoprojEventsV1alpha1EventSource**](io.argoproj.events.v1alpha1.EventSource.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## list_event_sources

> crate::models::IoArgoprojEventsV1alpha1EventSourceList list_event_sources(namespace, list_options_label_selector, list_options_field_selector, list_options_watch, list_options_allow_watch_bookmarks, list_options_resource_version, list_options_resource_version_match, list_options_timeout_seconds, list_options_limit, list_options_continue)


### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**namespace** | **String** |  | [required] |
**list_options_label_selector** | Option<**String**> | A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. |  |
**list_options_field_selector** | Option<**String**> | A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. |  |
**list_options_watch** | Option<**bool**> | Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. |  |
**list_options_allow_watch_bookmarks** | Option<**bool**> | allowWatchBookmarks requests watch events with type \"BOOKMARK\". Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server's discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. +optional. |  |
**list_options_resource_version** | Option<**String**> | resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional |  |
**list_options_resource_version_match** | Option<**String**> | resourceVersionMatch determines how resourceVersion is applied to list calls. It is highly recommended that resourceVersionMatch be set for list calls where resourceVersion is set See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional |  |
**list_options_timeout_seconds** | Option<**String**> | Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. |  |
**list_options_limit** | Option<**String**> | limit is a maximum number of responses to return for a list call. If more items exist, the server will set the `continue` field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. |  |
**list_options_continue** | Option<**String**> | The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the \"next key\".  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. |  |

### Return type

[**crate::models::IoArgoprojEventsV1alpha1EventSourceList**](io.argoproj.events.v1alpha1.EventSourceList.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## update_event_source

> crate::models::IoArgoprojEventsV1alpha1EventSource update_event_source(namespace, name, body)


### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**namespace** | **String** |  | [required] |
**name** | **String** |  | [required] |
**body** | [**EventsourceUpdateEventSourceRequest**](EventsourceUpdateEventSourceRequest.md) |  | [required] |

### Return type

[**crate::models::IoArgoprojEventsV1alpha1EventSource**](io.argoproj.events.v1alpha1.EventSource.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## watch_event_sources

> crate::models::StreamResultOfEventsourceEventSourceWatchEvent watch_event_sources(namespace, list_options_label_selector, list_options_field_selector, list_options_watch, list_options_allow_watch_bookmarks, list_options_resource_version, list_options_resource_version_match, list_options_timeout_seconds, list_options_limit, list_options_continue)


### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**namespace** | **String** |  | [required] |
**list_options_label_selector** | Option<**String**> | A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. |  |
**list_options_field_selector** | Option<**String**> | A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. |  |
**list_options_watch** | Option<**bool**> | Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. |  |
**list_options_allow_watch_bookmarks** | Option<**bool**> | allowWatchBookmarks requests watch events with type \"BOOKMARK\". Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server's discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. +optional. |  |
**list_options_resource_version** | Option<**String**> | resourceVersion sets a constraint on what resource versions a request may be served from. See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional |  |
**list_options_resource_version_match** | Option<**String**> | resourceVersionMatch determines how resourceVersion is applied to list calls. It is highly recommended that resourceVersionMatch be set for list calls where resourceVersion is set See https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions for details.  Defaults to unset +optional |  |
**list_options_timeout_seconds** | Option<**String**> | Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. |  |
**list_options_limit** | Option<**String**> | limit is a maximum number of responses to return for a list call. If more items exist, the server will set the `continue` field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. |  |
**list_options_continue** | Option<**String**> | The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the \"next key\".  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. |  |

### Return type

[**crate::models::StreamResultOfEventsourceEventSourceWatchEvent**](Stream_result_of_eventsource_EventSourceWatchEvent.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

