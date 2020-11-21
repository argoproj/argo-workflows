# Argo Server API
You can get examples of requests and responses by using the CLI with `--gloglevel=9`, e.g. `argo list --gloglevel=9`

## Version: v2.11.8

### Security
**BearerToken**  

|apiKey|*API Key*|
|---|---|
|Description|Bearer Token authentication|
|Name|authorization|
|In|header|

**HTTPBasic**  

|basic|*Basic*|
|---|---|
|Description|HTTP Basic authentication|

### /api/v1/archived-workflows

#### GET
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| listOptions.labelSelector | query | A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. | No | string |
| listOptions.fieldSelector | query | A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. | No | string |
| listOptions.watch | query | Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. | No | boolean (boolean) |
| listOptions.allowWatchBookmarks | query | allowWatchBookmarks requests watch events with type "BOOKMARK". Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server's discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. If the feature gate WatchBookmarks is not enabled in apiserver, this field is ignored. +optional. | No | boolean (boolean) |
| listOptions.resourceVersion | query | When specified with a watch call, shows changes that occur after that particular version of a resource. Defaults to changes from the beginning of history. When specified for list: - if unset, then the result is returned from remote storage based on quorum-read flag; - if it's 0, then we simply return what we currently have in cache, no guarantee; - if set to non zero, then the result is at least as fresh as given rv. +optional. | No | string |
| listOptions.timeoutSeconds | query | Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. | No | string (int64) |
| listOptions.limit | query | limit is a maximum number of responses to return for a list call. If more items exist, the server will set the `continue` field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. | No | string (int64) |
| listOptions.continue | query | The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the "next key".  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. | No | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.WorkflowList](#io.argoproj.workflow.v1alpha1.workflowlist) |

### /api/v1/archived-workflows/{uid}

#### GET
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| uid | path |  | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.Workflow](#io.argoproj.workflow.v1alpha1.workflow) |

#### DELETE
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| uid | path |  | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.ArchivedWorkflowDeletedResponse](#io.argoproj.workflow.v1alpha1.archivedworkflowdeletedresponse) |

### /api/v1/cluster-workflow-templates

#### GET
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| listOptions.labelSelector | query | A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. | No | string |
| listOptions.fieldSelector | query | A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. | No | string |
| listOptions.watch | query | Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. | No | boolean (boolean) |
| listOptions.allowWatchBookmarks | query | allowWatchBookmarks requests watch events with type "BOOKMARK". Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server's discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. If the feature gate WatchBookmarks is not enabled in apiserver, this field is ignored. +optional. | No | boolean (boolean) |
| listOptions.resourceVersion | query | When specified with a watch call, shows changes that occur after that particular version of a resource. Defaults to changes from the beginning of history. When specified for list: - if unset, then the result is returned from remote storage based on quorum-read flag; - if it's 0, then we simply return what we currently have in cache, no guarantee; - if set to non zero, then the result is at least as fresh as given rv. +optional. | No | string |
| listOptions.timeoutSeconds | query | Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. | No | string (int64) |
| listOptions.limit | query | limit is a maximum number of responses to return for a list call. If more items exist, the server will set the `continue` field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. | No | string (int64) |
| listOptions.continue | query | The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the "next key".  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. | No | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplateList](#io.argoproj.workflow.v1alpha1.clusterworkflowtemplatelist) |

#### POST
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| body | body |  | Yes | [io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplateCreateRequest](#io.argoproj.workflow.v1alpha1.clusterworkflowtemplatecreaterequest) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplate](#io.argoproj.workflow.v1alpha1.clusterworkflowtemplate) |

### /api/v1/cluster-workflow-templates/lint

#### POST
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| body | body |  | Yes | [io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplateLintRequest](#io.argoproj.workflow.v1alpha1.clusterworkflowtemplatelintrequest) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplate](#io.argoproj.workflow.v1alpha1.clusterworkflowtemplate) |

### /api/v1/cluster-workflow-templates/{name}

#### GET
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| name | path |  | Yes | string |
| getOptions.resourceVersion | query | When specified: - if unset, then the result is returned from remote storage based on quorum-read flag; - if it's 0, then we simply return what we currently have in cache, no guarantee; - if set to non zero, then the result is at least as fresh as given rv. | No | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplate](#io.argoproj.workflow.v1alpha1.clusterworkflowtemplate) |

#### PUT
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| name | path | DEPRECATED: This field is ignored. | Yes | string |
| body | body |  | Yes | [io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplateUpdateRequest](#io.argoproj.workflow.v1alpha1.clusterworkflowtemplateupdaterequest) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplate](#io.argoproj.workflow.v1alpha1.clusterworkflowtemplate) |

#### DELETE
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| name | path |  | Yes | string |
| deleteOptions.gracePeriodSeconds | query | The duration in seconds before the object should be deleted. Value must be non-negative integer. The value zero indicates delete immediately. If this value is nil, the default grace period for the specified type will be used. Defaults to a per object value if not specified. zero means delete immediately. +optional. | No | string (int64) |
| deleteOptions.preconditions.uid | query | Specifies the target UID. +optional. | No | string |
| deleteOptions.preconditions.resourceVersion | query | Specifies the target ResourceVersion +optional. | No | string |
| deleteOptions.orphanDependents | query | Deprecated: please use the PropagationPolicy, this field will be deprecated in 1.7. Should the dependent objects be orphaned. If true/false, the "orphan" finalizer will be added to/removed from the object's finalizers list. Either this field or PropagationPolicy may be set, but not both. +optional. | No | boolean (boolean) |
| deleteOptions.propagationPolicy | query | Whether and how garbage collection will be performed. Either this field or OrphanDependents may be set, but not both. The default policy is decided by the existing finalizer set in the metadata.finalizers and the resource-specific default policy. Acceptable values are: 'Orphan' - orphan the dependents; 'Background' - allow the garbage collector to delete the dependents in the background; 'Foreground' - a cascading policy that deletes all dependents in the foreground. +optional. | No | string |
| deleteOptions.dryRun | query | When present, indicates that modifications should not be persisted. An invalid or unrecognized dryRun directive will result in an error response and no further processing of the request. Valid values are: - All: all dry run stages will be processed +optional. | No | [ string ] |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplateDeleteResponse](#io.argoproj.workflow.v1alpha1.clusterworkflowtemplatedeleteresponse) |

### /api/v1/cron-workflows/{namespace}

#### GET
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| listOptions.labelSelector | query | A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. | No | string |
| listOptions.fieldSelector | query | A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. | No | string |
| listOptions.watch | query | Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. | No | boolean (boolean) |
| listOptions.allowWatchBookmarks | query | allowWatchBookmarks requests watch events with type "BOOKMARK". Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server's discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. If the feature gate WatchBookmarks is not enabled in apiserver, this field is ignored. +optional. | No | boolean (boolean) |
| listOptions.resourceVersion | query | When specified with a watch call, shows changes that occur after that particular version of a resource. Defaults to changes from the beginning of history. When specified for list: - if unset, then the result is returned from remote storage based on quorum-read flag; - if it's 0, then we simply return what we currently have in cache, no guarantee; - if set to non zero, then the result is at least as fresh as given rv. +optional. | No | string |
| listOptions.timeoutSeconds | query | Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. | No | string (int64) |
| listOptions.limit | query | limit is a maximum number of responses to return for a list call. If more items exist, the server will set the `continue` field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. | No | string (int64) |
| listOptions.continue | query | The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the "next key".  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. | No | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.CronWorkflowList](#io.argoproj.workflow.v1alpha1.cronworkflowlist) |

#### POST
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| body | body |  | Yes | [io.argoproj.workflow.v1alpha1.CreateCronWorkflowRequest](#io.argoproj.workflow.v1alpha1.createcronworkflowrequest) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.CronWorkflow](#io.argoproj.workflow.v1alpha1.cronworkflow) |

### /api/v1/cron-workflows/{namespace}/lint

#### POST
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| body | body |  | Yes | [io.argoproj.workflow.v1alpha1.LintCronWorkflowRequest](#io.argoproj.workflow.v1alpha1.lintcronworkflowrequest) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.CronWorkflow](#io.argoproj.workflow.v1alpha1.cronworkflow) |

### /api/v1/cron-workflows/{namespace}/{name}

#### GET
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| name | path |  | Yes | string |
| getOptions.resourceVersion | query | When specified: - if unset, then the result is returned from remote storage based on quorum-read flag; - if it's 0, then we simply return what we currently have in cache, no guarantee; - if set to non zero, then the result is at least as fresh as given rv. | No | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.CronWorkflow](#io.argoproj.workflow.v1alpha1.cronworkflow) |

#### PUT
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| name | path | DEPRECATED: This field is ignored. | Yes | string |
| body | body |  | Yes | [io.argoproj.workflow.v1alpha1.UpdateCronWorkflowRequest](#io.argoproj.workflow.v1alpha1.updatecronworkflowrequest) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.CronWorkflow](#io.argoproj.workflow.v1alpha1.cronworkflow) |

#### DELETE
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| name | path |  | Yes | string |
| deleteOptions.gracePeriodSeconds | query | The duration in seconds before the object should be deleted. Value must be non-negative integer. The value zero indicates delete immediately. If this value is nil, the default grace period for the specified type will be used. Defaults to a per object value if not specified. zero means delete immediately. +optional. | No | string (int64) |
| deleteOptions.preconditions.uid | query | Specifies the target UID. +optional. | No | string |
| deleteOptions.preconditions.resourceVersion | query | Specifies the target ResourceVersion +optional. | No | string |
| deleteOptions.orphanDependents | query | Deprecated: please use the PropagationPolicy, this field will be deprecated in 1.7. Should the dependent objects be orphaned. If true/false, the "orphan" finalizer will be added to/removed from the object's finalizers list. Either this field or PropagationPolicy may be set, but not both. +optional. | No | boolean (boolean) |
| deleteOptions.propagationPolicy | query | Whether and how garbage collection will be performed. Either this field or OrphanDependents may be set, but not both. The default policy is decided by the existing finalizer set in the metadata.finalizers and the resource-specific default policy. Acceptable values are: 'Orphan' - orphan the dependents; 'Background' - allow the garbage collector to delete the dependents in the background; 'Foreground' - a cascading policy that deletes all dependents in the foreground. +optional. | No | string |
| deleteOptions.dryRun | query | When present, indicates that modifications should not be persisted. An invalid or unrecognized dryRun directive will result in an error response and no further processing of the request. Valid values are: - All: all dry run stages will be processed +optional. | No | [ string ] |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.CronWorkflowDeletedResponse](#io.argoproj.workflow.v1alpha1.cronworkflowdeletedresponse) |

### /api/v1/events/{namespace}/{discriminator}

#### POST
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path | The namespace for the io.argoproj.workflow.v1alpha1. This can be empty if the client has cluster scoped permissions. If empty, then the event is "broadcast" to workflow event binding in all namespaces. | Yes | string |
| discriminator | path | Optional discriminator for the io.argoproj.workflow.v1alpha1. This should almost always be empty. Used for edge-cases where the event payload alone is not provide enough information to discriminate the event. This MUST NOT be used as security mechanism, e.g. to allow two clients to use the same access token, or to support webhooks on unsecured server. Instead, use access tokens. This is made available as `discriminator` in the event binding selector (`/spec/event/selector)` | Yes | string |
| body | body | The event itself can be any data. | Yes | [io.argoproj.workflow.v1alpha1.Item](#io.argoproj.workflow.v1alpha1.item) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.EventResponse](#io.argoproj.workflow.v1alpha1.eventresponse) |

### /api/v1/info

#### GET
##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.InfoResponse](#io.argoproj.workflow.v1alpha1.inforesponse) |

### /api/v1/stream/events/{namespace}

#### GET
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| listOptions.labelSelector | query | A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. | No | string |
| listOptions.fieldSelector | query | A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. | No | string |
| listOptions.watch | query | Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. | No | boolean (boolean) |
| listOptions.allowWatchBookmarks | query | allowWatchBookmarks requests watch events with type "BOOKMARK". Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server's discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. If the feature gate WatchBookmarks is not enabled in apiserver, this field is ignored. +optional. | No | boolean (boolean) |
| listOptions.resourceVersion | query | When specified with a watch call, shows changes that occur after that particular version of a resource. Defaults to changes from the beginning of history. When specified for list: - if unset, then the result is returned from remote storage based on quorum-read flag; - if it's 0, then we simply return what we currently have in cache, no guarantee; - if set to non zero, then the result is at least as fresh as given rv. +optional. | No | string |
| listOptions.timeoutSeconds | query | Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. | No | string (int64) |
| listOptions.limit | query | limit is a maximum number of responses to return for a list call. If more items exist, the server will set the `continue` field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. | No | string (int64) |
| listOptions.continue | query | The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the "next key".  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. | No | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response.(streaming responses) | object |

### /api/v1/userinfo

#### GET
##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.GetUserInfoResponse](#io.argoproj.workflow.v1alpha1.getuserinforesponse) |

### /api/v1/version

#### GET
##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.Version](#io.argoproj.workflow.v1alpha1.version) |

### /api/v1/workflow-events/{namespace}

#### GET
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| listOptions.labelSelector | query | A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. | No | string |
| listOptions.fieldSelector | query | A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. | No | string |
| listOptions.watch | query | Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. | No | boolean (boolean) |
| listOptions.allowWatchBookmarks | query | allowWatchBookmarks requests watch events with type "BOOKMARK". Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server's discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. If the feature gate WatchBookmarks is not enabled in apiserver, this field is ignored. +optional. | No | boolean (boolean) |
| listOptions.resourceVersion | query | When specified with a watch call, shows changes that occur after that particular version of a resource. Defaults to changes from the beginning of history. When specified for list: - if unset, then the result is returned from remote storage based on quorum-read flag; - if it's 0, then we simply return what we currently have in cache, no guarantee; - if set to non zero, then the result is at least as fresh as given rv. +optional. | No | string |
| listOptions.timeoutSeconds | query | Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. | No | string (int64) |
| listOptions.limit | query | limit is a maximum number of responses to return for a list call. If more items exist, the server will set the `continue` field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. | No | string (int64) |
| listOptions.continue | query | The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the "next key".  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. | No | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response.(streaming responses) | object |

### /api/v1/workflow-templates/{namespace}

#### GET
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| listOptions.labelSelector | query | A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. | No | string |
| listOptions.fieldSelector | query | A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. | No | string |
| listOptions.watch | query | Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. | No | boolean (boolean) |
| listOptions.allowWatchBookmarks | query | allowWatchBookmarks requests watch events with type "BOOKMARK". Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server's discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. If the feature gate WatchBookmarks is not enabled in apiserver, this field is ignored. +optional. | No | boolean (boolean) |
| listOptions.resourceVersion | query | When specified with a watch call, shows changes that occur after that particular version of a resource. Defaults to changes from the beginning of history. When specified for list: - if unset, then the result is returned from remote storage based on quorum-read flag; - if it's 0, then we simply return what we currently have in cache, no guarantee; - if set to non zero, then the result is at least as fresh as given rv. +optional. | No | string |
| listOptions.timeoutSeconds | query | Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. | No | string (int64) |
| listOptions.limit | query | limit is a maximum number of responses to return for a list call. If more items exist, the server will set the `continue` field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. | No | string (int64) |
| listOptions.continue | query | The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the "next key".  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. | No | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.WorkflowTemplateList](#io.argoproj.workflow.v1alpha1.workflowtemplatelist) |

#### POST
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| body | body |  | Yes | [io.argoproj.workflow.v1alpha1.WorkflowTemplateCreateRequest](#io.argoproj.workflow.v1alpha1.workflowtemplatecreaterequest) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.WorkflowTemplate](#io.argoproj.workflow.v1alpha1.workflowtemplate) |

### /api/v1/workflow-templates/{namespace}/lint

#### POST
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| body | body |  | Yes | [io.argoproj.workflow.v1alpha1.WorkflowTemplateLintRequest](#io.argoproj.workflow.v1alpha1.workflowtemplatelintrequest) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.WorkflowTemplate](#io.argoproj.workflow.v1alpha1.workflowtemplate) |

### /api/v1/workflow-templates/{namespace}/{name}

#### GET
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| name | path |  | Yes | string |
| getOptions.resourceVersion | query | When specified: - if unset, then the result is returned from remote storage based on quorum-read flag; - if it's 0, then we simply return what we currently have in cache, no guarantee; - if set to non zero, then the result is at least as fresh as given rv. | No | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.WorkflowTemplate](#io.argoproj.workflow.v1alpha1.workflowtemplate) |

#### PUT
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| name | path | DEPRECATED: This field is ignored. | Yes | string |
| body | body |  | Yes | [io.argoproj.workflow.v1alpha1.WorkflowTemplateUpdateRequest](#io.argoproj.workflow.v1alpha1.workflowtemplateupdaterequest) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.WorkflowTemplate](#io.argoproj.workflow.v1alpha1.workflowtemplate) |

#### DELETE
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| name | path |  | Yes | string |
| deleteOptions.gracePeriodSeconds | query | The duration in seconds before the object should be deleted. Value must be non-negative integer. The value zero indicates delete immediately. If this value is nil, the default grace period for the specified type will be used. Defaults to a per object value if not specified. zero means delete immediately. +optional. | No | string (int64) |
| deleteOptions.preconditions.uid | query | Specifies the target UID. +optional. | No | string |
| deleteOptions.preconditions.resourceVersion | query | Specifies the target ResourceVersion +optional. | No | string |
| deleteOptions.orphanDependents | query | Deprecated: please use the PropagationPolicy, this field will be deprecated in 1.7. Should the dependent objects be orphaned. If true/false, the "orphan" finalizer will be added to/removed from the object's finalizers list. Either this field or PropagationPolicy may be set, but not both. +optional. | No | boolean (boolean) |
| deleteOptions.propagationPolicy | query | Whether and how garbage collection will be performed. Either this field or OrphanDependents may be set, but not both. The default policy is decided by the existing finalizer set in the metadata.finalizers and the resource-specific default policy. Acceptable values are: 'Orphan' - orphan the dependents; 'Background' - allow the garbage collector to delete the dependents in the background; 'Foreground' - a cascading policy that deletes all dependents in the foreground. +optional. | No | string |
| deleteOptions.dryRun | query | When present, indicates that modifications should not be persisted. An invalid or unrecognized dryRun directive will result in an error response and no further processing of the request. Valid values are: - All: all dry run stages will be processed +optional. | No | [ string ] |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.WorkflowTemplateDeleteResponse](#io.argoproj.workflow.v1alpha1.workflowtemplatedeleteresponse) |

### /api/v1/workflows/{namespace}

#### GET
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| listOptions.labelSelector | query | A selector to restrict the list of returned objects by their labels. Defaults to everything. +optional. | No | string |
| listOptions.fieldSelector | query | A selector to restrict the list of returned objects by their fields. Defaults to everything. +optional. | No | string |
| listOptions.watch | query | Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion. +optional. | No | boolean (boolean) |
| listOptions.allowWatchBookmarks | query | allowWatchBookmarks requests watch events with type "BOOKMARK". Servers that do not implement bookmarks may ignore this flag and bookmarks are sent at the server's discretion. Clients should not assume bookmarks are returned at any specific interval, nor may they assume the server will send any BOOKMARK event during a session. If this is not a watch, this field is ignored. If the feature gate WatchBookmarks is not enabled in apiserver, this field is ignored. +optional. | No | boolean (boolean) |
| listOptions.resourceVersion | query | When specified with a watch call, shows changes that occur after that particular version of a resource. Defaults to changes from the beginning of history. When specified for list: - if unset, then the result is returned from remote storage based on quorum-read flag; - if it's 0, then we simply return what we currently have in cache, no guarantee; - if set to non zero, then the result is at least as fresh as given rv. +optional. | No | string |
| listOptions.timeoutSeconds | query | Timeout for the list/watch call. This limits the duration of the call, regardless of any activity or inactivity. +optional. | No | string (int64) |
| listOptions.limit | query | limit is a maximum number of responses to return for a list call. If more items exist, the server will set the `continue` field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.  The server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned. | No | string (int64) |
| listOptions.continue | query | The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server, the server will respond with a 410 ResourceExpired error together with a continue token. If the client needs a consistent list, it must restart their list without the continue field. Otherwise, the client may send another list request with the token received with the 410 error, the server will respond with a list starting from the next key, but from the latest snapshot, which is inconsistent from the previous list results - objects that are created, modified, or deleted after the first list request will be included in the response, as long as their keys are after the "next key".  This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications. | No | string |
| fields | query | Fields to be included or excluded in the response. e.g. "items.spec,items.status.phase", "-items.status.nodes". | No | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.WorkflowList](#io.argoproj.workflow.v1alpha1.workflowlist) |

#### POST
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| body | body |  | Yes | [io.argoproj.workflow.v1alpha1.WorkflowCreateRequest](#io.argoproj.workflow.v1alpha1.workflowcreaterequest) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.Workflow](#io.argoproj.workflow.v1alpha1.workflow) |

### /api/v1/workflows/{namespace}/lint

#### POST
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| body | body |  | Yes | [io.argoproj.workflow.v1alpha1.WorkflowLintRequest](#io.argoproj.workflow.v1alpha1.workflowlintrequest) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.Workflow](#io.argoproj.workflow.v1alpha1.workflow) |

### /api/v1/workflows/{namespace}/submit

#### POST
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| body | body |  | Yes | [io.argoproj.workflow.v1alpha1.WorkflowSubmitRequest](#io.argoproj.workflow.v1alpha1.workflowsubmitrequest) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.Workflow](#io.argoproj.workflow.v1alpha1.workflow) |

### /api/v1/workflows/{namespace}/{name}

#### GET
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| name | path |  | Yes | string |
| getOptions.resourceVersion | query | When specified: - if unset, then the result is returned from remote storage based on quorum-read flag; - if it's 0, then we simply return what we currently have in cache, no guarantee; - if set to non zero, then the result is at least as fresh as given rv. | No | string |
| fields | query | Fields to be included or excluded in the response. e.g. "spec,status.phase", "-status.nodes". | No | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.Workflow](#io.argoproj.workflow.v1alpha1.workflow) |

#### DELETE
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| name | path |  | Yes | string |
| deleteOptions.gracePeriodSeconds | query | The duration in seconds before the object should be deleted. Value must be non-negative integer. The value zero indicates delete immediately. If this value is nil, the default grace period for the specified type will be used. Defaults to a per object value if not specified. zero means delete immediately. +optional. | No | string (int64) |
| deleteOptions.preconditions.uid | query | Specifies the target UID. +optional. | No | string |
| deleteOptions.preconditions.resourceVersion | query | Specifies the target ResourceVersion +optional. | No | string |
| deleteOptions.orphanDependents | query | Deprecated: please use the PropagationPolicy, this field will be deprecated in 1.7. Should the dependent objects be orphaned. If true/false, the "orphan" finalizer will be added to/removed from the object's finalizers list. Either this field or PropagationPolicy may be set, but not both. +optional. | No | boolean (boolean) |
| deleteOptions.propagationPolicy | query | Whether and how garbage collection will be performed. Either this field or OrphanDependents may be set, but not both. The default policy is decided by the existing finalizer set in the metadata.finalizers and the resource-specific default policy. Acceptable values are: 'Orphan' - orphan the dependents; 'Background' - allow the garbage collector to delete the dependents in the background; 'Foreground' - a cascading policy that deletes all dependents in the foreground. +optional. | No | string |
| deleteOptions.dryRun | query | When present, indicates that modifications should not be persisted. An invalid or unrecognized dryRun directive will result in an error response and no further processing of the request. Valid values are: - All: all dry run stages will be processed +optional. | No | [ string ] |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.WorkflowDeleteResponse](#io.argoproj.workflow.v1alpha1.workflowdeleteresponse) |

### /api/v1/workflows/{namespace}/{name}/resubmit

#### PUT
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| name | path |  | Yes | string |
| body | body |  | Yes | [io.argoproj.workflow.v1alpha1.WorkflowResubmitRequest](#io.argoproj.workflow.v1alpha1.workflowresubmitrequest) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.Workflow](#io.argoproj.workflow.v1alpha1.workflow) |

### /api/v1/workflows/{namespace}/{name}/resume

#### PUT
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| name | path |  | Yes | string |
| body | body |  | Yes | [io.argoproj.workflow.v1alpha1.WorkflowResumeRequest](#io.argoproj.workflow.v1alpha1.workflowresumerequest) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.Workflow](#io.argoproj.workflow.v1alpha1.workflow) |

### /api/v1/workflows/{namespace}/{name}/retry

#### PUT
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| name | path |  | Yes | string |
| body | body |  | Yes | [io.argoproj.workflow.v1alpha1.WorkflowRetryRequest](#io.argoproj.workflow.v1alpha1.workflowretryrequest) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.Workflow](#io.argoproj.workflow.v1alpha1.workflow) |

### /api/v1/workflows/{namespace}/{name}/set

#### PUT
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| name | path |  | Yes | string |
| body | body |  | Yes | [io.argoproj.workflow.v1alpha1.WorkflowSetRequest](#io.argoproj.workflow.v1alpha1.workflowsetrequest) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.Workflow](#io.argoproj.workflow.v1alpha1.workflow) |

### /api/v1/workflows/{namespace}/{name}/stop

#### PUT
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| name | path |  | Yes | string |
| body | body |  | Yes | [io.argoproj.workflow.v1alpha1.WorkflowStopRequest](#io.argoproj.workflow.v1alpha1.workflowstoprequest) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.Workflow](#io.argoproj.workflow.v1alpha1.workflow) |

### /api/v1/workflows/{namespace}/{name}/suspend

#### PUT
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| name | path |  | Yes | string |
| body | body |  | Yes | [io.argoproj.workflow.v1alpha1.WorkflowSuspendRequest](#io.argoproj.workflow.v1alpha1.workflowsuspendrequest) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.Workflow](#io.argoproj.workflow.v1alpha1.workflow) |

### /api/v1/workflows/{namespace}/{name}/terminate

#### PUT
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| name | path |  | Yes | string |
| body | body |  | Yes | [io.argoproj.workflow.v1alpha1.WorkflowTerminateRequest](#io.argoproj.workflow.v1alpha1.workflowterminaterequest) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [io.argoproj.workflow.v1alpha1.Workflow](#io.argoproj.workflow.v1alpha1.workflow) |

### /api/v1/workflows/{namespace}/{name}/{podName}/log

#### GET
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| namespace | path |  | Yes | string |
| name | path |  | Yes | string |
| podName | path |  | Yes | string |
| logOptions.container | query | The container for which to stream logs. Defaults to only container if there is one container in the pod. +optional. | No | string |
| logOptions.follow | query | Follow the log stream of the pod. Defaults to false. +optional. | No | boolean (boolean) |
| logOptions.previous | query | Return previous terminated container logs. Defaults to false. +optional. | No | boolean (boolean) |
| logOptions.sinceSeconds | query | A relative time in seconds before the current time from which to show logs. If this value precedes the time a pod was started, only logs since the pod start will be returned. If this value is in the future, no logs will be returned. Only one of sinceSeconds or sinceTime may be specified. +optional. | No | string (int64) |
| logOptions.sinceTime.seconds | query | Represents seconds of UTC time since Unix epoch 1970-01-01T00:00:00Z. Must be from 0001-01-01T00:00:00Z to 9999-12-31T23:59:59Z inclusive. | No | string (int64) |
| logOptions.sinceTime.nanos | query | Non-negative fractions of a second at nanosecond resolution. Negative second values with fractions must still have non-negative nanos values that count forward in time. Must be from 0 to 999,999,999 inclusive. This field may be limited in precision depending on context. | No | integer |
| logOptions.timestamps | query | If true, add an RFC3339 or RFC3339Nano timestamp at the beginning of every line of log output. Defaults to false. +optional. | No | boolean (boolean) |
| logOptions.tailLines | query | If set, the number of lines from the end of the logs to show. If not specified, logs are shown from the creation of the container or sinceSeconds or sinceTime +optional. | No | string (int64) |
| logOptions.limitBytes | query | If set, the number of bytes to read from the server before terminating the log output. This may not display a complete final line of logging, and may return slightly more or slightly less than the specified limit. +optional. | No | string (int64) |
| logOptions.insecureSkipTLSVerifyBackend | query | insecureSkipTLSVerifyBackend indicates that the apiserver should not confirm the validity of the serving certificate of the backend it is connecting to.  This will make the HTTPS connection between the apiserver and the backend insecure. This means the apiserver cannot verify the log data it is receiving came from the real kubelet.  If the kubelet is configured to verify the apiserver's TLS credentials, it does not mean the connection to the real kubelet is vulnerable to a man in the middle attack (e.g. an attacker could not intercept the actual log data coming from the real kubelet). +optional. | No | boolean (boolean) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response.(streaming responses) | object |

### Models

#### google.protobuf.Any

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| type_url | string |  | No |
| value | byte |  | No |

#### grpc.gateway.runtime.StreamError

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| details | [ [google.protobuf.Any](#google.protobuf.any) ] |  | No |
| grpc_code | integer |  | No |
| http_code | integer |  | No |
| http_status | string |  | No |
| message | string |  | No |

#### io.argoproj.workflow.v1alpha1.Amount

Amount represent a numeric amount.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| io.argoproj.workflow.v1alpha1.Amount | number | Amount represent a numeric amount. |  |

#### io.argoproj.workflow.v1alpha1.ArchiveStrategy

ArchiveStrategy describes how to archive files/directory when saving artifacts

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| none | [io.argoproj.workflow.v1alpha1.NoneStrategy](#io.argoproj.workflow.v1alpha1.nonestrategy) |  | No |
| tar | [io.argoproj.workflow.v1alpha1.TarStrategy](#io.argoproj.workflow.v1alpha1.tarstrategy) |  | No |

#### io.argoproj.workflow.v1alpha1.ArchivedWorkflowDeletedResponse

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| io.argoproj.workflow.v1alpha1.ArchivedWorkflowDeletedResponse | object |  |  |

#### io.argoproj.workflow.v1alpha1.Arguments

Arguments to a template

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| artifacts | [ [io.argoproj.workflow.v1alpha1.Artifact](#io.argoproj.workflow.v1alpha1.artifact) ] | Artifacts is the list of artifacts to pass to the template or workflow | No |
| parameters | [ [io.argoproj.workflow.v1alpha1.Parameter](#io.argoproj.workflow.v1alpha1.parameter) ] | Parameters is the list of parameters to pass to the template or workflow | No |

#### io.argoproj.workflow.v1alpha1.Artifact

Artifact indicates an artifact to place at a specified path

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| archive | [io.argoproj.workflow.v1alpha1.ArchiveStrategy](#io.argoproj.workflow.v1alpha1.archivestrategy) | Archive controls how the artifact will be saved to the artifact repository. | No |
| archiveLogs | boolean | ArchiveLogs indicates if the container logs should be archived | No |
| artifactory | [io.argoproj.workflow.v1alpha1.ArtifactoryArtifact](#io.argoproj.workflow.v1alpha1.artifactoryartifact) | Artifactory contains artifactory artifact location details | No |
| from | string | From allows an artifact to reference an artifact from a previous step | No |
| gcs | [io.argoproj.workflow.v1alpha1.GCSArtifact](#io.argoproj.workflow.v1alpha1.gcsartifact) | GCS contains GCS artifact location details | No |
| git | [io.argoproj.workflow.v1alpha1.GitArtifact](#io.argoproj.workflow.v1alpha1.gitartifact) | Git contains git artifact location details | No |
| globalName | string | GlobalName exports an output artifact to the global scope, making it available as '{{io.argoproj.workflow.v1alpha1.outputs.artifacts.XXXX}} and in workflow.status.outputs.artifacts | No |
| hdfs | [io.argoproj.workflow.v1alpha1.HDFSArtifact](#io.argoproj.workflow.v1alpha1.hdfsartifact) | HDFS contains HDFS artifact location details | No |
| http | [io.argoproj.workflow.v1alpha1.HTTPArtifact](#io.argoproj.workflow.v1alpha1.httpartifact) | HTTP contains HTTP artifact location details | No |
| mode | integer | mode bits to use on this file, must be a value between 0 and 0777 set when loading input artifacts. | No |
| name | string | name of the artifact. must be unique within a template's inputs/outputs. | Yes |
| optional | boolean | Make Artifacts optional, if Artifacts doesn't generate or exist | No |
| oss | [io.argoproj.workflow.v1alpha1.OSSArtifact](#io.argoproj.workflow.v1alpha1.ossartifact) | OSS contains OSS artifact location details | No |
| path | string | Path is the container path to the artifact | No |
| raw | [io.argoproj.workflow.v1alpha1.RawArtifact](#io.argoproj.workflow.v1alpha1.rawartifact) | Raw contains raw artifact location details | No |
| s3 | [io.argoproj.workflow.v1alpha1.S3Artifact](#io.argoproj.workflow.v1alpha1.s3artifact) | S3 contains S3 artifact location details | No |
| subPath | string | SubPath allows an artifact to be sourced from a subpath within the specified source | No |

#### io.argoproj.workflow.v1alpha1.ArtifactLocation

ArtifactLocation describes a location for a single or multiple artifacts. It is used as single artifact in the context of inputs/outputs (e.g. outputs.artifacts.artname). It is also used to describe the location of multiple artifacts such as the archive location of a single workflow step, which the executor will use as a default location to store its files.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| archiveLogs | boolean | ArchiveLogs indicates if the container logs should be archived | No |
| artifactory | [io.argoproj.workflow.v1alpha1.ArtifactoryArtifact](#io.argoproj.workflow.v1alpha1.artifactoryartifact) | Artifactory contains artifactory artifact location details | No |
| gcs | [io.argoproj.workflow.v1alpha1.GCSArtifact](#io.argoproj.workflow.v1alpha1.gcsartifact) | GCS contains GCS artifact location details | No |
| git | [io.argoproj.workflow.v1alpha1.GitArtifact](#io.argoproj.workflow.v1alpha1.gitartifact) | Git contains git artifact location details | No |
| hdfs | [io.argoproj.workflow.v1alpha1.HDFSArtifact](#io.argoproj.workflow.v1alpha1.hdfsartifact) | HDFS contains HDFS artifact location details | No |
| http | [io.argoproj.workflow.v1alpha1.HTTPArtifact](#io.argoproj.workflow.v1alpha1.httpartifact) | HTTP contains HTTP artifact location details | No |
| oss | [io.argoproj.workflow.v1alpha1.OSSArtifact](#io.argoproj.workflow.v1alpha1.ossartifact) | OSS contains OSS artifact location details | No |
| raw | [io.argoproj.workflow.v1alpha1.RawArtifact](#io.argoproj.workflow.v1alpha1.rawartifact) | Raw contains raw artifact location details | No |
| s3 | [io.argoproj.workflow.v1alpha1.S3Artifact](#io.argoproj.workflow.v1alpha1.s3artifact) | S3 contains S3 artifact location details | No |

#### io.argoproj.workflow.v1alpha1.ArtifactRepositoryRef

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| configMap | string |  | No |
| key | string |  | No |

#### io.argoproj.workflow.v1alpha1.ArtifactoryArtifact

ArtifactoryArtifact is the location of an artifactory artifact

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| passwordSecret | [io.k8s.api.core.v1.SecretKeySelector](#io.k8s.api.core.v1.secretkeyselector) | PasswordSecret is the secret selector to the repository password | No |
| url | string | URL of the artifact | Yes |
| usernameSecret | [io.k8s.api.core.v1.SecretKeySelector](#io.k8s.api.core.v1.secretkeyselector) | UsernameSecret is the secret selector to the repository username | No |

#### io.argoproj.workflow.v1alpha1.Backoff

Backoff is a backoff strategy to use within retryStrategy

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| duration | string | Duration is the amount to back off. Default unit is seconds, but could also be a duration (e.g. "2m", "1h") | No |
| factor | [io.k8s.apimachinery.pkg.util.intstr.IntOrString](#io.k8s.apimachinery.pkg.util.intstr.intorstring) | Factor is a factor to multiply the base duration after each failed retry | No |
| maxDuration | string | MaxDuration is the maximum amount of time allowed for the backoff strategy | No |

#### io.argoproj.workflow.v1alpha1.Cache

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| configMap | [io.k8s.api.core.v1.ConfigMapKeySelector](#io.k8s.api.core.v1.configmapkeyselector) |  | Yes |

#### io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplate

ClusterWorkflowTemplate is the definition of a workflow template resource in cluster scope

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| apiVersion | string | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: <https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#resources> | No |
| kind | string | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: <https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#types-kinds> | No |
| metadata | [io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta](#io.k8s.apimachinery.pkg.apis.meta.v1.objectmeta) |  | Yes |
| spec | [io.argoproj.workflow.v1alpha1.WorkflowTemplateSpec](#io.argoproj.workflow.v1alpha1.workflowtemplatespec) |  | Yes |

#### io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplateCreateRequest

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| createOptions | [io.k8s.apimachinery.pkg.apis.meta.v1.CreateOptions](#io.k8s.apimachinery.pkg.apis.meta.v1.createoptions) |  | No |
| template | [io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplate](#io.argoproj.workflow.v1alpha1.clusterworkflowtemplate) |  | No |

#### io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplateDeleteResponse

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplateDeleteResponse | object |  |  |

#### io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplateLintRequest

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| createOptions | [io.k8s.apimachinery.pkg.apis.meta.v1.CreateOptions](#io.k8s.apimachinery.pkg.apis.meta.v1.createoptions) |  | No |
| template | [io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplate](#io.argoproj.workflow.v1alpha1.clusterworkflowtemplate) |  | No |

#### io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplateList

ClusterWorkflowTemplateList is list of ClusterWorkflowTemplate resources

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| apiVersion | string | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: <https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#resources> | No |
| items | [ [io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplate](#io.argoproj.workflow.v1alpha1.clusterworkflowtemplate) ] |  | Yes |
| kind | string | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: <https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#types-kinds> | No |
| metadata | [io.k8s.apimachinery.pkg.apis.meta.v1.ListMeta](#io.k8s.apimachinery.pkg.apis.meta.v1.listmeta) |  | Yes |

#### io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplateUpdateRequest

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| name | string | DEPRECATED: This field is ignored. | No |
| template | [io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplate](#io.argoproj.workflow.v1alpha1.clusterworkflowtemplate) |  | No |

#### io.argoproj.workflow.v1alpha1.Condition

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| message | string | Message is the condition message | No |
| status | string | Status is the status of the condition | No |
| type | string | Type is the type of condition | No |

#### io.argoproj.workflow.v1alpha1.ContinueOn

ContinueOn defines if a workflow should continue even if a task or step fails/errors. It can be specified if the workflow should continue when the pod errors, fails or both.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| error | boolean |  | No |
| failed | boolean |  | No |

#### io.argoproj.workflow.v1alpha1.Counter

Counter is a Counter prometheus metric

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| value | string | Value is the value of the metric | Yes |

#### io.argoproj.workflow.v1alpha1.CreateCronWorkflowRequest

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| createOptions | [io.k8s.apimachinery.pkg.apis.meta.v1.CreateOptions](#io.k8s.apimachinery.pkg.apis.meta.v1.createoptions) |  | No |
| cronWorkflow | [io.argoproj.workflow.v1alpha1.CronWorkflow](#io.argoproj.workflow.v1alpha1.cronworkflow) |  | No |
| namespace | string |  | No |

#### io.argoproj.workflow.v1alpha1.CronWorkflow

CronWorkflow is the definition of a scheduled workflow resource

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| apiVersion | string | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: <https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#resources> | No |
| kind | string | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: <https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#types-kinds> | No |
| metadata | [io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta](#io.k8s.apimachinery.pkg.apis.meta.v1.objectmeta) |  | Yes |
| spec | [io.argoproj.workflow.v1alpha1.CronWorkflowSpec](#io.argoproj.workflow.v1alpha1.cronworkflowspec) |  | Yes |
| status | [io.argoproj.workflow.v1alpha1.CronWorkflowStatus](#io.argoproj.workflow.v1alpha1.cronworkflowstatus) |  | No |

#### io.argoproj.workflow.v1alpha1.CronWorkflowDeletedResponse

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| io.argoproj.workflow.v1alpha1.CronWorkflowDeletedResponse | object |  |  |

#### io.argoproj.workflow.v1alpha1.CronWorkflowList

CronWorkflowList is list of CronWorkflow resources

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| apiVersion | string | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: <https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#resources> | No |
| items | [ [io.argoproj.workflow.v1alpha1.CronWorkflow](#io.argoproj.workflow.v1alpha1.cronworkflow) ] |  | Yes |
| kind | string | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: <https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#types-kinds> | No |
| metadata | [io.k8s.apimachinery.pkg.apis.meta.v1.ListMeta](#io.k8s.apimachinery.pkg.apis.meta.v1.listmeta) |  | Yes |

#### io.argoproj.workflow.v1alpha1.CronWorkflowSpec

CronWorkflowSpec is the specification of a CronWorkflow

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| concurrencyPolicy | string | ConcurrencyPolicy is the K8s-style concurrency policy that will be used | No |
| failedJobsHistoryLimit | integer | FailedJobsHistoryLimit is the number of successful jobs to be kept at a time | No |
| schedule | string | Schedule is a schedule to run the Workflow in Cron format | Yes |
| startingDeadlineSeconds | long | StartingDeadlineSeconds is the K8s-style deadline that will limit the time a CronWorkflow will be run after its original scheduled time if it is missed. | No |
| successfulJobsHistoryLimit | integer | SuccessfulJobsHistoryLimit is the number of successful jobs to be kept at a time | No |
| suspend | boolean | Suspend is a flag that will stop new CronWorkflows from running if set to true | No |
| timezone | string | Timezone is the timezone against which the cron schedule will be calculated, e.g. "Asia/Tokyo". Default is machine's local time. | No |
| workflowMetadata | [io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta](#io.k8s.apimachinery.pkg.apis.meta.v1.objectmeta) | WorkflowMetadata contains some metadata of the workflow to be run | No |
| workflowSpec | [io.argoproj.workflow.v1alpha1.WorkflowSpec](#io.argoproj.workflow.v1alpha1.workflowspec) | WorkflowSpec is the spec of the workflow to be run | Yes |

#### io.argoproj.workflow.v1alpha1.CronWorkflowStatus

CronWorkflowStatus is the status of a CronWorkflow

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| active | [ [io.k8s.api.core.v1.ObjectReference](#io.k8s.api.core.v1.objectreference) ] | Active is a list of active workflows stemming from this CronWorkflow | No |
| conditions | [ [io.argoproj.workflow.v1alpha1.Condition](#io.argoproj.workflow.v1alpha1.condition) ] | Conditions is a list of conditions the CronWorkflow may have | No |
| lastScheduledTime | [io.k8s.apimachinery.pkg.apis.meta.v1.Time](#io.k8s.apimachinery.pkg.apis.meta.v1.time) | LastScheduleTime is the last time the CronWorkflow was scheduled | No |

#### io.argoproj.workflow.v1alpha1.DAGTask

DAGTask represents a node in the graph during DAG execution

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| arguments | [io.argoproj.workflow.v1alpha1.Arguments](#io.argoproj.workflow.v1alpha1.arguments) | Arguments are the parameter and artifact arguments to the template | No |
| continueOn | [io.argoproj.workflow.v1alpha1.ContinueOn](#io.argoproj.workflow.v1alpha1.continueon) | ContinueOn makes argo to proceed with the following step even if this step fails. Errors and Failed states can be specified | No |
| dependencies | [ string ] | Dependencies are name of other targets which this depends on | No |
| depends | string | Depends are name of other targets which this depends on | No |
| name | string | Name is the name of the target | Yes |
| onExit | string | OnExit is a template reference which is invoked at the end of the template, irrespective of the success, failure, or error of the primary template. | No |
| template | string | Name of template to execute | Yes |
| templateRef | [io.argoproj.workflow.v1alpha1.TemplateRef](#io.argoproj.workflow.v1alpha1.templateref) | TemplateRef is the reference to the template resource to execute. | No |
| when | string | When is an expression in which the task should conditionally execute | No |
| withItems | [ [io.argoproj.workflow.v1alpha1.Item](#io.argoproj.workflow.v1alpha1.item) ] | WithItems expands a task into multiple parallel tasks from the items in the list | No |
| withParam | string | WithParam expands a task into multiple parallel tasks from the value in the parameter, which is expected to be a JSON list. | No |
| withSequence | [io.argoproj.workflow.v1alpha1.Sequence](#io.argoproj.workflow.v1alpha1.sequence) | WithSequence expands a task into a numeric sequence | No |

#### io.argoproj.workflow.v1alpha1.DAGTemplate

DAGTemplate is a template subtype for directed acyclic graph templates

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| failFast | boolean | This flag is for DAG logic. The DAG logic has a built-in "fail fast" feature to stop scheduling new steps, as soon as it detects that one of the DAG nodes is failed. Then it waits until all DAG nodes are completed before failing the DAG itself. The FailFast flag default is true,  if set to false, it will allow a DAG to run all branches of the DAG to completion (either success or failure), regardless of the failed outcomes of branches in the DAG. More info and example about this feature at <https://github.com/argoproj/argo/issues/1442> | No |
| target | string | Target are one or more names of targets to execute in a DAG | No |
| tasks | [ [io.argoproj.workflow.v1alpha1.DAGTask](#io.argoproj.workflow.v1alpha1.dagtask) ] | Tasks are a list of DAG tasks | Yes |

#### io.argoproj.workflow.v1alpha1.Event

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| selector | string | Selector (<https://github.com/antonmedv/expr>) that we must must match the io.argoproj.workflow.v1alpha1. E.g. `payload.message == "test"` | Yes |

#### io.argoproj.workflow.v1alpha1.EventResponse

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| io.argoproj.workflow.v1alpha1.EventResponse | object |  |  |

#### io.argoproj.workflow.v1alpha1.ExecutorConfig

ExecutorConfig holds configurations of an executor container.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| serviceAccountName | string | ServiceAccountName specifies the service account name of the executor container. | No |

#### io.argoproj.workflow.v1alpha1.GCSArtifact

GCSArtifact is the location of a GCS artifact

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| bucket | string | Bucket is the name of the bucket | Yes |
| key | string | Key is the path in the bucket where the artifact resides | Yes |
| serviceAccountKeySecret | [io.k8s.api.core.v1.SecretKeySelector](#io.k8s.api.core.v1.secretkeyselector) | ServiceAccountKeySecret is the secret selector to the bucket's service account key | No |

#### io.argoproj.workflow.v1alpha1.Gauge

Gauge is a Gauge prometheus metric

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| realtime | boolean | Realtime emits this metric in real time if applicable | Yes |
| value | string | Value is the value of the metric | Yes |

#### io.argoproj.workflow.v1alpha1.GetUserInfoResponse

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| issuer | string |  | No |
| subject | string |  | No |

#### io.argoproj.workflow.v1alpha1.GitArtifact

GitArtifact is the location of an git artifact

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| depth | long | Depth specifies clones/fetches should be shallow and include the given number of commits from the branch tip | No |
| fetch | [ string ] | Fetch specifies a number of refs that should be fetched before checkout | No |
| insecureIgnoreHostKey | boolean | InsecureIgnoreHostKey disables SSH strict host key checking during git clone | No |
| passwordSecret | [io.k8s.api.core.v1.SecretKeySelector](#io.k8s.api.core.v1.secretkeyselector) | PasswordSecret is the secret selector to the repository password | No |
| repo | string | Repo is the git repository | Yes |
| revision | string | Revision is the git commit, tag, branch to checkout | No |
| sshPrivateKeySecret | [io.k8s.api.core.v1.SecretKeySelector](#io.k8s.api.core.v1.secretkeyselector) | SSHPrivateKeySecret is the secret selector to the repository ssh private key | No |
| usernameSecret | [io.k8s.api.core.v1.SecretKeySelector](#io.k8s.api.core.v1.secretkeyselector) | UsernameSecret is the secret selector to the repository username | No |

#### io.argoproj.workflow.v1alpha1.HDFSArtifact

HDFSArtifact is the location of an HDFS artifact

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| addresses | [ string ] | Addresses is accessible addresses of HDFS name nodes | Yes |
| force | boolean | Force copies a file forcibly even if it exists (default: false) | No |
| hdfsUser | string | HDFSUser is the user to access HDFS file system. It is ignored if either ccache or keytab is used. | No |
| krbCCacheSecret | [io.k8s.api.core.v1.SecretKeySelector](#io.k8s.api.core.v1.secretkeyselector) | KrbCCacheSecret is the secret selector for Kerberos ccache Either ccache or keytab can be set to use Kerberos. | No |
| krbConfigConfigMap | [io.k8s.api.core.v1.ConfigMapKeySelector](#io.k8s.api.core.v1.configmapkeyselector) | KrbConfig is the configmap selector for Kerberos config as string It must be set if either ccache or keytab is used. | No |
| krbKeytabSecret | [io.k8s.api.core.v1.SecretKeySelector](#io.k8s.api.core.v1.secretkeyselector) | KrbKeytabSecret is the secret selector for Kerberos keytab Either ccache or keytab can be set to use Kerberos. | No |
| krbRealm | string | KrbRealm is the Kerberos realm used with Kerberos keytab It must be set if keytab is used. | No |
| krbServicePrincipalName | string | KrbServicePrincipalName is the principal name of Kerberos service It must be set if either ccache or keytab is used. | No |
| krbUsername | string | KrbUsername is the Kerberos username used with Kerberos keytab It must be set if keytab is used. | No |
| path | string | Path is a file path in HDFS | Yes |

#### io.argoproj.workflow.v1alpha1.HTTPArtifact

HTTPArtifact allows an file served on HTTP to be placed as an input artifact in a container

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| url | string | URL of the artifact | Yes |

#### io.argoproj.workflow.v1alpha1.Histogram

Histogram is a Histogram prometheus metric

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| buckets | [ [io.argoproj.workflow.v1alpha1.Amount](#io.argoproj.workflow.v1alpha1.amount) ] | Buckets is a list of bucket divisors for the histogram | Yes |
| value | string | Value is the value of the metric | Yes |

#### io.argoproj.workflow.v1alpha1.InfoResponse

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| links | [ [io.argoproj.workflow.v1alpha1.Link](#io.argoproj.workflow.v1alpha1.link) ] |  | No |
| managedNamespace | string |  | No |

#### io.argoproj.workflow.v1alpha1.Inputs

Inputs are the mechanism for passing parameters, artifacts, volumes from one template to another

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| artifacts | [ [io.argoproj.workflow.v1alpha1.Artifact](#io.argoproj.workflow.v1alpha1.artifact) ] | Artifact are a list of artifacts passed as inputs | No |
| parameters | [ [io.argoproj.workflow.v1alpha1.Parameter](#io.argoproj.workflow.v1alpha1.parameter) ] | Parameters are a list of parameters passed as inputs | No |

#### io.argoproj.workflow.v1alpha1.Item

Item expands a single workflow step into multiple parallel steps The value of Item can be a map, string, bool, or number

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| io.argoproj.workflow.v1alpha1.Item |  | Item expands a single workflow step into multiple parallel steps The value of Item can be a map, string, bool, or number |  |

#### io.argoproj.workflow.v1alpha1.Link

A link to another app.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| name | string | The name of the link, E.g. "Workflow Logs" or "Pod Logs" | Yes |
| scope | string | Either "workflow" or "pod" | Yes |
| url | string | The URL. May contain "${metadata.namespace}" and "${metadata.name}". | Yes |

#### io.argoproj.workflow.v1alpha1.LintCronWorkflowRequest

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| cronWorkflow | [io.argoproj.workflow.v1alpha1.CronWorkflow](#io.argoproj.workflow.v1alpha1.cronworkflow) |  | No |
| namespace | string |  | No |

#### io.argoproj.workflow.v1alpha1.LogEntry

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| content | string |  | No |
| podName | string |  | No |

#### io.argoproj.workflow.v1alpha1.MemoizationStatus

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| cacheName | string |  | Yes |
| hit | boolean |  | Yes |
| key | string |  | Yes |

#### io.argoproj.workflow.v1alpha1.Memoize

Memoization

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| cache | [io.argoproj.workflow.v1alpha1.Cache](#io.argoproj.workflow.v1alpha1.cache) |  | Yes |
| key | string |  | Yes |

#### io.argoproj.workflow.v1alpha1.Metadata

Pod metdata

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| annotations | object |  | No |
| labels | object |  | No |

#### io.argoproj.workflow.v1alpha1.MetricLabel

MetricLabel is a single label for a prometheus metric

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| key | string |  | Yes |
| value | string |  | Yes |

#### io.argoproj.workflow.v1alpha1.Metrics

Metrics are a list of metrics emitted from a Workflow/Template

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| prometheus | [ [io.argoproj.workflow.v1alpha1.Prometheus](#io.argoproj.workflow.v1alpha1.prometheus) ] | Prometheus is a list of prometheus metrics to be emitted | Yes |

#### io.argoproj.workflow.v1alpha1.Mutex

Mutex holds Mutex configuration

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| name | string | name of the mutex | No |

#### io.argoproj.workflow.v1alpha1.MutexHolding

MutexHolding describes the mutex and the object which is holding it.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| holder | string | Holder is a reference to the object which holds the Mutex. Holding Scenario:   1. Current workflow's NodeID which is holding the lock.      e.g: ${NodeID} Waiting Scenario:   1. Current workflow or other workflow NodeID which is holding the lock.      e.g: ${WorkflowName}/${NodeID} | No |
| mutex | string | Reference for the mutex e.g: ${namespace}/mutex/${mutexName} | No |

#### io.argoproj.workflow.v1alpha1.MutexStatus

MutexStatus contains which objects hold  mutex locks, and which objects this workflow is waiting on to release locks.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| holding | [ [io.argoproj.workflow.v1alpha1.MutexHolding](#io.argoproj.workflow.v1alpha1.mutexholding) ] | Holding is a list of mutexes and their respective objects that are held by mutex lock for this io.argoproj.workflow.v1alpha1. | No |
| waiting | [ [io.argoproj.workflow.v1alpha1.MutexHolding](#io.argoproj.workflow.v1alpha1.mutexholding) ] | Waiting is a list of mutexes and their respective objects this workflow is waiting for. | No |

#### io.argoproj.workflow.v1alpha1.NodeStatus

NodeStatus contains status information about an individual node in the workflow

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| boundaryID | string | BoundaryID indicates the node ID of the associated template root node in which this node belongs to | No |
| children | [ string ] | Children is a list of child node IDs | No |
| daemoned | boolean | Daemoned tracks whether or not this node was daemoned and need to be terminated | No |
| displayName | string | DisplayName is a human readable representation of the node. Unique within a template boundary | No |
| finishedAt | [io.k8s.apimachinery.pkg.apis.meta.v1.Time](#io.k8s.apimachinery.pkg.apis.meta.v1.time) | Time at which this node completed | No |
| hostNodeName | string | HostNodeName name of the Kubernetes node on which the Pod is running, if applicable | No |
| id | string | ID is a unique identifier of a node within the worklow It is implemented as a hash of the node name, which makes the ID deterministic | Yes |
| inputs | [io.argoproj.workflow.v1alpha1.Inputs](#io.argoproj.workflow.v1alpha1.inputs) | Inputs captures input parameter values and artifact locations supplied to this template invocation | No |
| memoizationStatus | [io.argoproj.workflow.v1alpha1.MemoizationStatus](#io.argoproj.workflow.v1alpha1.memoizationstatus) | MemoizationStatus holds information about cached nodes | No |
| message | string | A human readable message indicating details about why the node is in this condition. | No |
| name | string | Name is unique name in the node tree used to generate the node ID | Yes |
| outboundNodes | [ string ] | OutboundNodes tracks the node IDs which are considered "outbound" nodes to a template invocation. For every invocation of a template, there are nodes which we considered as "outbound". Essentially, these are last nodes in the execution sequence to run, before the template is considered completed. These nodes are then connected as parents to a following step.  In the case of single pod steps (i.e. container, script, resource templates), this list will be nil since the pod itself is already considered the "outbound" node. In the case of DAGs, outbound nodes are the "target" tasks (tasks with no children). In the case of steps, outbound nodes are all the containers involved in the last step group. NOTE: since templates are composable, the list of outbound nodes are carried upwards when a DAG/steps template invokes another DAG/steps template. In other words, the outbound nodes of a template, will be a superset of the outbound nodes of its last children. | No |
| outputs | [io.argoproj.workflow.v1alpha1.Outputs](#io.argoproj.workflow.v1alpha1.outputs) | Outputs captures output parameter values and artifact locations produced by this template invocation | No |
| phase | string | Phase a simple, high-level summary of where the node is in its lifecycle. Can be used as a state machine. | No |
| podIP | string | PodIP captures the IP of the pod for daemoned steps | No |
| resourcesDuration | object | ResourcesDuration is indicative, but not accurate, resource duration. This is populated when the nodes completes. | No |
| startedAt | [io.k8s.apimachinery.pkg.apis.meta.v1.Time](#io.k8s.apimachinery.pkg.apis.meta.v1.time) | Time at which this node started | No |
| storedTemplateID | string | StoredTemplateID is the ID of stored template. DEPRECATED: This value is not used anymore. | No |
| synchronizationStatus | [io.argoproj.workflow.v1alpha1.NodeSynchronizationStatus](#io.argoproj.workflow.v1alpha1.nodesynchronizationstatus) | SynchronizationStatus is the synchronization status of the node | No |
| templateName | string | TemplateName is the template name which this node corresponds to. Not applicable to virtual nodes (e.g. Retry, StepGroup) | No |
| templateRef | [io.argoproj.workflow.v1alpha1.TemplateRef](#io.argoproj.workflow.v1alpha1.templateref) | TemplateRef is the reference to the template resource which this node corresponds to. Not applicable to virtual nodes (e.g. Retry, StepGroup) | No |
| templateScope | string | TemplateScope is the template scope in which the template of this node was retrieved. | No |
| type | string | Type indicates type of node | Yes |
| workflowTemplateName | string | WorkflowTemplateName is the WorkflowTemplate resource name on which the resolved template of this node is retrieved. DEPRECATED: This value is not used anymore. | No |

#### io.argoproj.workflow.v1alpha1.NodeSynchronizationStatus

NodeSynchronizationStatus stores the status of a node

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| waiting | string | Waiting is the name of the lock that this node is waiting for | No |

#### io.argoproj.workflow.v1alpha1.NoneStrategy

NoneStrategy indicates to skip tar process and upload the files or directory tree as independent files. Note that if the artifact is a directory, the artifact driver must support the ability to save/load the directory appropriately.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| io.argoproj.workflow.v1alpha1.NoneStrategy | object | NoneStrategy indicates to skip tar process and upload the files or directory tree as independent files. Note that if the artifact is a directory, the artifact driver must support the ability to save/load the directory appropriately. |  |

#### io.argoproj.workflow.v1alpha1.OSSArtifact

OSSArtifact is the location of an Alibaba Cloud OSS artifact

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| accessKeySecret | [io.k8s.api.core.v1.SecretKeySelector](#io.k8s.api.core.v1.secretkeyselector) | AccessKeySecret is the secret selector to the bucket's access key | Yes |
| bucket | string | Bucket is the name of the bucket | Yes |
| endpoint | string | Endpoint is the hostname of the bucket endpoint | Yes |
| key | string | Key is the path in the bucket where the artifact resides | Yes |
| secretKeySecret | [io.k8s.api.core.v1.SecretKeySelector](#io.k8s.api.core.v1.secretkeyselector) | SecretKeySecret is the secret selector to the bucket's secret key | Yes |

#### io.argoproj.workflow.v1alpha1.Outputs

Outputs hold parameters, artifacts, and results from a step

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| artifacts | [ [io.argoproj.workflow.v1alpha1.Artifact](#io.argoproj.workflow.v1alpha1.artifact) ] | Artifacts holds the list of output artifacts produced by a step | No |
| exitCode | string | ExitCode holds the exit code of a script template | No |
| parameters | [ [io.argoproj.workflow.v1alpha1.Parameter](#io.argoproj.workflow.v1alpha1.parameter) ] | Parameters holds the list of output parameters produced by a step | No |
| result | string | Result holds the result (stdout) of a script template | No |

#### io.argoproj.workflow.v1alpha1.ParallelSteps

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| io.argoproj.workflow.v1alpha1.ParallelSteps | array |  |  |

#### io.argoproj.workflow.v1alpha1.Parameter

Parameter indicate a passed string parameter to a service template with an optional default value

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| default | string | Default is the default value to use for an input parameter if a value was not supplied | No |
| globalName | string | GlobalName exports an output parameter to the global scope, making it available as '{{io.argoproj.workflow.v1alpha1.outputs.parameters.XXXX}} and in workflow.status.outputs.parameters | No |
| name | string | Name is the parameter name | Yes |
| value | string | Value is the literal value to use for the parameter. If specified in the context of an input parameter, the value takes precedence over any passed values | No |
| valueFrom | [io.argoproj.workflow.v1alpha1.ValueFrom](#io.argoproj.workflow.v1alpha1.valuefrom) | ValueFrom is the source for the output parameter's value | No |

#### io.argoproj.workflow.v1alpha1.PodGC

PodGC describes how to delete completed pods as they complete

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| strategy | string | Strategy is the strategy to use. One of "OnPodCompletion", "OnPodSuccess", "OnWorkflowCompletion", "OnWorkflowSuccess" | No |

#### io.argoproj.workflow.v1alpha1.Prometheus

Prometheus is a prometheus metric to be emitted

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| counter | [io.argoproj.workflow.v1alpha1.Counter](#io.argoproj.workflow.v1alpha1.counter) | Counter is a counter metric | No |
| gauge | [io.argoproj.workflow.v1alpha1.Gauge](#io.argoproj.workflow.v1alpha1.gauge) | Gauge is a gauge metric | No |
| help | string | Help is a string that describes the metric | Yes |
| histogram | [io.argoproj.workflow.v1alpha1.Histogram](#io.argoproj.workflow.v1alpha1.histogram) | Histogram is a histogram metric | No |
| labels | [ [io.argoproj.workflow.v1alpha1.MetricLabel](#io.argoproj.workflow.v1alpha1.metriclabel) ] | Labels is a list of metric labels | No |
| name | string | Name is the name of the metric | Yes |
| when | string | When is a conditional statement that decides when to emit the metric | No |

#### io.argoproj.workflow.v1alpha1.RawArtifact

RawArtifact allows raw string content to be placed as an artifact in a container

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| data | string | Data is the string contents of the artifact | Yes |

#### io.argoproj.workflow.v1alpha1.ResourceTemplate

ResourceTemplate is a template subtype to manipulate kubernetes resources

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| action | string | Action is the action to perform to the resource. Must be one of: get, create, apply, delete, replace, patch | Yes |
| failureCondition | string | FailureCondition is a label selector expression which describes the conditions of the k8s resource in which the step was considered failed | No |
| flags | [ string ] | Flags is a set of additional options passed to kubectl before submitting a resource I.e. to disable resource validation: flags: [  "--validate=false"  # disable resource validation ] | No |
| manifest | string | Manifest contains the kubernetes manifest | No |
| mergeStrategy | string | MergeStrategy is the strategy used to merge a patch. It defaults to "strategic" Must be one of: strategic, merge, json | No |
| setOwnerReference | boolean | SetOwnerReference sets the reference to the workflow on the OwnerReference of generated resource. | No |
| successCondition | string | SuccessCondition is a label selector expression which describes the conditions of the k8s resource in which it is acceptable to proceed to the following step | No |

#### io.argoproj.workflow.v1alpha1.RetryStrategy

RetryStrategy provides controls on how to retry a workflow step

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| backoff | [io.argoproj.workflow.v1alpha1.Backoff](#io.argoproj.workflow.v1alpha1.backoff) | Backoff is a backoff strategy | No |
| limit | [io.k8s.apimachinery.pkg.util.intstr.IntOrString](#io.k8s.apimachinery.pkg.util.intstr.intorstring) | Limit is the maximum number of attempts when retrying a container | No |
| retryPolicy | string | RetryPolicy is a policy of NodePhase statuses that will be retried | No |

#### io.argoproj.workflow.v1alpha1.S3Artifact

S3Artifact is the location of an S3 artifact

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| accessKeySecret | [io.k8s.api.core.v1.SecretKeySelector](#io.k8s.api.core.v1.secretkeyselector) | AccessKeySecret is the secret selector to the bucket's access key | Yes |
| bucket | string | Bucket is the name of the bucket | Yes |
| endpoint | string | Endpoint is the hostname of the bucket endpoint | Yes |
| insecure | boolean | Insecure will connect to the service with TLS | No |
| key | string | Key is the key in the bucket where the artifact resides | Yes |
| region | string | Region contains the optional bucket region | No |
| roleARN | string | RoleARN is the Amazon Resource Name (ARN) of the role to assume. | No |
| secretKeySecret | [io.k8s.api.core.v1.SecretKeySelector](#io.k8s.api.core.v1.secretkeyselector) | SecretKeySecret is the secret selector to the bucket's secret key | Yes |
| useSDKCreds | boolean | UseSDKCreds tells the driver to figure out credentials based on sdk defaults. | No |

#### io.argoproj.workflow.v1alpha1.ScriptTemplate

ScriptTemplate is a template subtype to enable scripting through code steps

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| args | [ string ] | Arguments to the entrypoint. The docker image's CMD is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: <https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell> | No |
| command | [ string ] | Entrypoint array. Not executed within a shell. The docker image's ENTRYPOINT is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: <https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell> | No |
| env | [ [io.k8s.api.core.v1.EnvVar](#io.k8s.api.core.v1.envvar) ] | List of environment variables to set in the container. Cannot be updated. | No |
| envFrom | [ [io.k8s.api.core.v1.EnvFromSource](#io.k8s.api.core.v1.envfromsource) ] | List of sources to populate environment variables in the container. The keys defined within a source must be a C_IDENTIFIER. All invalid keys will be reported as an event when the container is starting. When a key exists in multiple sources, the value associated with the last source will take precedence. Values defined by an Env with a duplicate key will take precedence. Cannot be updated. | No |
| image | string | Docker image name. More info: <https://kubernetes.io/docs/concepts/containers/images> This field is optional to allow higher level config management to default or override container images in workload controllers like Deployments and StatefulSets. | Yes |
| imagePullPolicy | string | Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: <https://kubernetes.io/docs/concepts/containers/images#updating-images> | No |
| lifecycle | [io.k8s.api.core.v1.Lifecycle](#io.k8s.api.core.v1.lifecycle) | Actions that the management system should take in response to container lifecycle events. Cannot be updated. | No |
| livenessProbe | [io.k8s.api.core.v1.Probe](#io.k8s.api.core.v1.probe) | Periodic probe of container liveness. Container will be restarted if the probe fails. Cannot be updated. More info: <https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes> | No |
| name | string | Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated. | No |
| ports | [ [io.k8s.api.core.v1.ContainerPort](#io.k8s.api.core.v1.containerport) ] | List of ports to expose from the container. Exposing a port here gives the system additional information about the network connections a container uses, but is primarily informational. Not specifying a port here DOES NOT prevent that port from being exposed. Any port which is listening on the default "0.0.0.0" address inside a container will be accessible from the network. Cannot be updated. | No |
| readinessProbe | [io.k8s.api.core.v1.Probe](#io.k8s.api.core.v1.probe) | Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. Cannot be updated. More info: <https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes> | No |
| resources | [io.k8s.api.core.v1.ResourceRequirements](#io.k8s.api.core.v1.resourcerequirements) | Compute Resources required by this container. Cannot be updated. More info: <https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/> | No |
| securityContext | [io.k8s.api.core.v1.SecurityContext](#io.k8s.api.core.v1.securitycontext) | Security options the pod should run with. More info: <https://kubernetes.io/docs/concepts/policy/security-context/> More info: <https://kubernetes.io/docs/tasks/configure-pod-container/security-context/> | No |
| source | string | Source contains the source code of the script to execute | Yes |
| startupProbe | [io.k8s.api.core.v1.Probe](#io.k8s.api.core.v1.probe) | StartupProbe indicates that the Pod has successfully initialized. If specified, no other probes are executed until this completes successfully. If this probe fails, the Pod will be restarted, just as if the livenessProbe failed. This can be used to provide different probe parameters at the beginning of a Pod's lifecycle, when it might take a long time to load data or warm a cache, than during steady-state operation. This cannot be updated. This is an alpha feature enabled by the StartupProbe feature flag. More info: <https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes> | No |
| stdin | boolean | Whether this container should allocate a buffer for stdin in the container runtime. If this is not set, reads from stdin in the container will always result in EOF. Default is false. | No |
| stdinOnce | boolean | Whether the container runtime should close the stdin channel after it has been opened by a single attach. When stdin is true the stdin stream will remain open across multiple attach sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the first client attaches to stdin, and then remains open and accepts data until the client disconnects, at which time stdin is closed and remains closed until the container is restarted. If this flag is false, a container processes that reads from stdin will never receive an EOF. Default is false | No |
| terminationMessagePath | string | Optional: Path at which the file to which the container's termination message will be written is mounted into the container's filesystem. Message written is intended to be brief final status, such as an assertion failure message. Will be truncated by the node if greater than 4096 bytes. The total message length across all containers will be limited to 12kb. Defaults to /dev/termination-log. Cannot be updated. | No |
| terminationMessagePolicy | string | Indicate how the termination message should be populated. File will use the contents of terminationMessagePath to populate the container status message on both success and failure. FallbackToLogsOnError will use the last chunk of container log output if the termination message file is empty and the container exited with an error. The log output is limited to 2048 bytes or 80 lines, whichever is smaller. Defaults to File. Cannot be updated. | No |
| tty | boolean | Whether this container should allocate a TTY for itself, also requires 'stdin' to be true. Default is false. | No |
| volumeDevices | [ [io.k8s.api.core.v1.VolumeDevice](#io.k8s.api.core.v1.volumedevice) ] | volumeDevices is the list of block devices to be used by the container. This is a beta feature. | No |
| volumeMounts | [ [io.k8s.api.core.v1.VolumeMount](#io.k8s.api.core.v1.volumemount) ] | Pod volumes to mount into the container's filesystem. Cannot be updated. | No |
| workingDir | string | Container's working directory. If not specified, the container runtime's default will be used, which might be configured in the container image. Cannot be updated. | No |

#### io.argoproj.workflow.v1alpha1.SemaphoreHolding

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| holders | [ string ] | Holders stores the list of current holder names in the io.argoproj.workflow.v1alpha1. | No |
| semaphore | string | Semaphore stores the semaphore name. | No |

#### io.argoproj.workflow.v1alpha1.SemaphoreRef

SemaphoreRef is a reference of Semaphore

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| configMapKeyRef | [io.k8s.api.core.v1.ConfigMapKeySelector](#io.k8s.api.core.v1.configmapkeyselector) | ConfigMapKeyRef is configmap selector for Semaphore configuration | No |

#### io.argoproj.workflow.v1alpha1.SemaphoreStatus

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| holding | [ [io.argoproj.workflow.v1alpha1.SemaphoreHolding](#io.argoproj.workflow.v1alpha1.semaphoreholding) ] | Holding stores the list of resource acquired synchronization lock for workflows. | No |
| waiting | [ [io.argoproj.workflow.v1alpha1.SemaphoreHolding](#io.argoproj.workflow.v1alpha1.semaphoreholding) ] | Waiting indicates the list of current synchronization lock holders. | No |

#### io.argoproj.workflow.v1alpha1.Sequence

Sequence expands a workflow step into numeric range

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| count | [io.k8s.apimachinery.pkg.util.intstr.IntOrString](#io.k8s.apimachinery.pkg.util.intstr.intorstring) | Count is number of elements in the sequence (default: 0). Not to be used with end | No |
| end | [io.k8s.apimachinery.pkg.util.intstr.IntOrString](#io.k8s.apimachinery.pkg.util.intstr.intorstring) | Number at which to end the sequence (default: 0). Not to be used with Count | No |
| format | string | Format is a printf format string to format the value in the sequence | No |
| start | [io.k8s.apimachinery.pkg.util.intstr.IntOrString](#io.k8s.apimachinery.pkg.util.intstr.intorstring) | Number at which to start the sequence (default: 0) | No |

#### io.argoproj.workflow.v1alpha1.Submit

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| arguments | [io.argoproj.workflow.v1alpha1.Arguments](#io.argoproj.workflow.v1alpha1.arguments) | Arguments extracted from the event and then set as arguments to the workflow created. | No |
| workflowTemplateRef | [io.argoproj.workflow.v1alpha1.WorkflowTemplateRef](#io.argoproj.workflow.v1alpha1.workflowtemplateref) | WorkflowTemplateRef the workflow template to submit | Yes |

#### io.argoproj.workflow.v1alpha1.SubmitOpts

SubmitOpts are workflow submission options

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| dryRun | boolean | DryRun validates the workflow on the client-side without creating it. This option is not supported in API | No |
| entryPoint | string | Entrypoint overrides spec.entrypoint | No |
| generateName | string | GenerateName overrides metadata.generateName | No |
| labels | string | Labels adds to metadata.labels | No |
| name | string | Name overrides metadata.name | No |
| ownerReference | [io.k8s.apimachinery.pkg.apis.meta.v1.OwnerReference](#io.k8s.apimachinery.pkg.apis.meta.v1.ownerreference) | OwnerReference creates a metadata.ownerReference | No |
| parameterFile | string | ParameterFile holds a reference to a parameter file. This option is not supported in API | No |
| parameters | [ string ] | Parameters passes input parameters to workflow | No |
| serverDryRun | boolean | ServerDryRun validates the workflow on the server-side without creating it | No |
| serviceAccount | string | ServiceAccount runs all pods in the workflow using specified ServiceAccount. | No |

#### io.argoproj.workflow.v1alpha1.SuppliedValueFrom

SuppliedValueFrom is a placeholder for a value to be filled in directly, either through the CLI, API, etc.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| io.argoproj.workflow.v1alpha1.SuppliedValueFrom | object | SuppliedValueFrom is a placeholder for a value to be filled in directly, either through the CLI, API, etc. |  |

#### io.argoproj.workflow.v1alpha1.SuspendTemplate

SuspendTemplate is a template subtype to suspend a workflow at a predetermined point in time

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| duration | string | Duration is the seconds to wait before automatically resuming a template | No |

#### io.argoproj.workflow.v1alpha1.Synchronization

Synchronization holds synchronization lock configuration

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| mutex | [io.argoproj.workflow.v1alpha1.Mutex](#io.argoproj.workflow.v1alpha1.mutex) | Mutex holds the Mutex lock details | No |
| semaphore | [io.argoproj.workflow.v1alpha1.SemaphoreRef](#io.argoproj.workflow.v1alpha1.semaphoreref) | Semaphore holds the Semaphore configuration | No |

#### io.argoproj.workflow.v1alpha1.SynchronizationStatus

SynchronizationStatus stores the status of semaphore and mutex.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| mutex | [io.argoproj.workflow.v1alpha1.MutexStatus](#io.argoproj.workflow.v1alpha1.mutexstatus) | Mutex stores this workflow's mutex holder details | No |
| semaphore | [io.argoproj.workflow.v1alpha1.SemaphoreStatus](#io.argoproj.workflow.v1alpha1.semaphorestatus) | Semaphore stores this workflow's Semaphore holder details | No |

#### io.argoproj.workflow.v1alpha1.TTLStrategy

TTLStrategy is the strategy for the time to live depending on if the workflow succeeded or failed

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| secondsAfterCompletion | integer | SecondsAfterCompletion is the number of seconds to live after completion | No |
| secondsAfterFailure | integer | SecondsAfterFailure is the number of seconds to live after failure | No |
| secondsAfterSuccess | integer | SecondsAfterSuccess is the number of seconds to live after success | No |

#### io.argoproj.workflow.v1alpha1.TarStrategy

TarStrategy will tar and gzip the file or directory when saving

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| compressionLevel | integer | CompressionLevel specifies the gzip compression level to use for the artifact. Defaults to gzip.DefaultCompression. | No |

#### io.argoproj.workflow.v1alpha1.Template

Template is a reusable and composable unit of execution in a workflow

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| activeDeadlineSeconds | [io.k8s.apimachinery.pkg.util.intstr.IntOrString](#io.k8s.apimachinery.pkg.util.intstr.intorstring) | Optional duration in seconds relative to the StartTime that the pod may be active on a node before the system actively tries to terminate the pod; value must be positive integer This field is only applicable to container and script templates. | No |
| affinity | [io.k8s.api.core.v1.Affinity](#io.k8s.api.core.v1.affinity) | Affinity sets the pod's scheduling constraints Overrides the affinity set at the workflow level (if any) | No |
| archiveLocation | [io.argoproj.workflow.v1alpha1.ArtifactLocation](#io.argoproj.workflow.v1alpha1.artifactlocation) | Location in which all files related to the step will be stored (logs, artifacts, etc...). Can be overridden by individual items in Outputs. If omitted, will use the default artifact repository location configured in the controller, appended with the <workflowname>/<nodename> in the key. | No |
| arguments | [io.argoproj.workflow.v1alpha1.Arguments](#io.argoproj.workflow.v1alpha1.arguments) | Arguments hold arguments to the template. DEPRECATED: This field is not used. | No |
| automountServiceAccountToken | boolean | AutomountServiceAccountToken indicates whether a service account token should be automatically mounted in pods. ServiceAccountName of ExecutorConfig must be specified if this value is false. | No |
| container | [io.k8s.api.core.v1.Container](#io.k8s.api.core.v1.container) | Container is the main container image to run in the pod | No |
| daemon | boolean | Deamon will allow a workflow to proceed to the next step so long as the container reaches readiness | No |
| dag | [io.argoproj.workflow.v1alpha1.DAGTemplate](#io.argoproj.workflow.v1alpha1.dagtemplate) | DAG template subtype which runs a DAG | No |
| executor | [io.argoproj.workflow.v1alpha1.ExecutorConfig](#io.argoproj.workflow.v1alpha1.executorconfig) | Executor holds configurations of the executor container. | No |
| hostAliases | [ [io.k8s.api.core.v1.HostAlias](#io.k8s.api.core.v1.hostalias) ] | HostAliases is an optional list of hosts and IPs that will be injected into the pod spec | No |
| initContainers | [ [io.argoproj.workflow.v1alpha1.UserContainer](#io.argoproj.workflow.v1alpha1.usercontainer) ] | InitContainers is a list of containers which run before the main container. | No |
| inputs | [io.argoproj.workflow.v1alpha1.Inputs](#io.argoproj.workflow.v1alpha1.inputs) | Inputs describe what inputs parameters and artifacts are supplied to this template | No |
| memoize | [io.argoproj.workflow.v1alpha1.Memoize](#io.argoproj.workflow.v1alpha1.memoize) | Memoize allows templates to use outputs generated from already executed templates | No |
| metadata | [io.argoproj.workflow.v1alpha1.Metadata](#io.argoproj.workflow.v1alpha1.metadata) | Metdata sets the pods's metadata, i.e. annotations and labels | No |
| metrics | [io.argoproj.workflow.v1alpha1.Metrics](#io.argoproj.workflow.v1alpha1.metrics) | Metrics are a list of metrics emitted from this template | No |
| name | string | Name is the name of the template | Yes |
| nodeSelector | object | NodeSelector is a selector to schedule this step of the workflow to be run on the selected node(s). Overrides the selector set at the workflow level. | No |
| outputs | [io.argoproj.workflow.v1alpha1.Outputs](#io.argoproj.workflow.v1alpha1.outputs) | Outputs describe the parameters and artifacts that this template produces | No |
| parallelism | long | Parallelism limits the max total parallel pods that can execute at the same time within the boundaries of this template invocation. If additional steps/dag templates are invoked, the pods created by those templates will not be counted towards this total. | No |
| podSpecPatch | string | PodSpecPatch holds strategic merge patch to apply against the pod spec. Allows parameterization of container fields which are not strings (e.g. resource limits). | No |
| priority | integer | Priority to apply to workflow pods. | No |
| priorityClassName | string | PriorityClassName to apply to workflow pods. | No |
| resource | [io.argoproj.workflow.v1alpha1.ResourceTemplate](#io.argoproj.workflow.v1alpha1.resourcetemplate) | Resource template subtype which can run k8s resources | No |
| resubmitPendingPods | boolean | ResubmitPendingPods is a flag to enable resubmitting pods that remain Pending after initial submission | No |
| retryStrategy | [io.argoproj.workflow.v1alpha1.RetryStrategy](#io.argoproj.workflow.v1alpha1.retrystrategy) | RetryStrategy describes how to retry a template when it fails | No |
| schedulerName | string | If specified, the pod will be dispatched by specified scheduler. Or it will be dispatched by workflow scope scheduler if specified. If neither specified, the pod will be dispatched by default scheduler. | No |
| script | [io.argoproj.workflow.v1alpha1.ScriptTemplate](#io.argoproj.workflow.v1alpha1.scripttemplate) | Script runs a portion of code against an interpreter | No |
| securityContext | [io.k8s.api.core.v1.PodSecurityContext](#io.k8s.api.core.v1.podsecuritycontext) | SecurityContext holds pod-level security attributes and common container settings. Optional: Defaults to empty.  See type description for default values of each field. | No |
| serviceAccountName | string | ServiceAccountName to apply to workflow pods | No |
| sidecars | [ [io.argoproj.workflow.v1alpha1.UserContainer](#io.argoproj.workflow.v1alpha1.usercontainer) ] | Sidecars is a list of containers which run alongside the main container Sidecars are automatically killed when the main container completes | No |
| steps | [ [io.argoproj.workflow.v1alpha1.ParallelSteps](#io.argoproj.workflow.v1alpha1.parallelsteps) ] | Steps define a series of sequential/parallel workflow steps | No |
| suspend | [io.argoproj.workflow.v1alpha1.SuspendTemplate](#io.argoproj.workflow.v1alpha1.suspendtemplate) | Suspend template subtype which can suspend a workflow when reaching the step | No |
| synchronization | [io.argoproj.workflow.v1alpha1.Synchronization](#io.argoproj.workflow.v1alpha1.synchronization) | Synchronization holds synchronization lock configuration for this template | No |
| template | string | Template is the name of the template which is used as the base of this template. DEPRECATED: This field is not used. | No |
| templateRef | [io.argoproj.workflow.v1alpha1.TemplateRef](#io.argoproj.workflow.v1alpha1.templateref) | TemplateRef is the reference to the template resource which is used as the base of this template. DEPRECATED: This field is not used. | No |
| timeout | string | Timout allows to set the total node execution timeout duration counting from the node's start time. This duration also includes time in which the node spends in Pending state. This duration may not be applied to Step or DAG templates. | No |
| tolerations | [ [io.k8s.api.core.v1.Toleration](#io.k8s.api.core.v1.toleration) ] | Tolerations to apply to workflow pods. | No |
| volumes | [ [io.k8s.api.core.v1.Volume](#io.k8s.api.core.v1.volume) ] | Volumes is a list of volumes that can be mounted by containers in a template. | No |

#### io.argoproj.workflow.v1alpha1.TemplateRef

TemplateRef is a reference of template resource.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| clusterScope | boolean | ClusterScope indicates the referred template is cluster scoped (i.e. a ClusterWorkflowTemplate). | No |
| name | string | Name is the resource name of the template. | No |
| runtimeResolution | boolean | RuntimeResolution skips validation at creation time. By enabling this option, you can create the referred workflow template before the actual runtime. | No |
| template | string | Template is the name of referred template in the resource. | No |

#### io.argoproj.workflow.v1alpha1.UpdateCronWorkflowRequest

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| cronWorkflow | [io.argoproj.workflow.v1alpha1.CronWorkflow](#io.argoproj.workflow.v1alpha1.cronworkflow) |  | No |
| name | string | DEPRECATED: This field is ignored. | No |
| namespace | string |  | No |

#### io.argoproj.workflow.v1alpha1.UserContainer

UserContainer is a container specified by a user.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| args | [ string ] | Arguments to the entrypoint. The docker image's CMD is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: <https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell> | No |
| command | [ string ] | Entrypoint array. Not executed within a shell. The docker image's ENTRYPOINT is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: <https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell> | No |
| env | [ [io.k8s.api.core.v1.EnvVar](#io.k8s.api.core.v1.envvar) ] | List of environment variables to set in the container. Cannot be updated. | No |
| envFrom | [ [io.k8s.api.core.v1.EnvFromSource](#io.k8s.api.core.v1.envfromsource) ] | List of sources to populate environment variables in the container. The keys defined within a source must be a C_IDENTIFIER. All invalid keys will be reported as an event when the container is starting. When a key exists in multiple sources, the value associated with the last source will take precedence. Values defined by an Env with a duplicate key will take precedence. Cannot be updated. | No |
| image | string | Docker image name. More info: <https://kubernetes.io/docs/concepts/containers/images> This field is optional to allow higher level config management to default or override container images in workload controllers like Deployments and StatefulSets. | No |
| imagePullPolicy | string | Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: <https://kubernetes.io/docs/concepts/containers/images#updating-images> | No |
| lifecycle | [io.k8s.api.core.v1.Lifecycle](#io.k8s.api.core.v1.lifecycle) | Actions that the management system should take in response to container lifecycle events. Cannot be updated. | No |
| livenessProbe | [io.k8s.api.core.v1.Probe](#io.k8s.api.core.v1.probe) | Periodic probe of container liveness. Container will be restarted if the probe fails. Cannot be updated. More info: <https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes> | No |
| mirrorVolumeMounts | boolean | MirrorVolumeMounts will mount the same volumes specified in the main container to the container (including artifacts), at the same mountPaths. This enables dind daemon to partially see the same filesystem as the main container in order to use features such as docker volume binding | No |
| name | string | Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated. | Yes |
| ports | [ [io.k8s.api.core.v1.ContainerPort](#io.k8s.api.core.v1.containerport) ] | List of ports to expose from the container. Exposing a port here gives the system additional information about the network connections a container uses, but is primarily informational. Not specifying a port here DOES NOT prevent that port from being exposed. Any port which is listening on the default "0.0.0.0" address inside a container will be accessible from the network. Cannot be updated. | No |
| readinessProbe | [io.k8s.api.core.v1.Probe](#io.k8s.api.core.v1.probe) | Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. Cannot be updated. More info: <https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes> | No |
| resources | [io.k8s.api.core.v1.ResourceRequirements](#io.k8s.api.core.v1.resourcerequirements) | Compute Resources required by this container. Cannot be updated. More info: <https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/> | No |
| securityContext | [io.k8s.api.core.v1.SecurityContext](#io.k8s.api.core.v1.securitycontext) | Security options the pod should run with. More info: <https://kubernetes.io/docs/concepts/policy/security-context/> More info: <https://kubernetes.io/docs/tasks/configure-pod-container/security-context/> | No |
| startupProbe | [io.k8s.api.core.v1.Probe](#io.k8s.api.core.v1.probe) | StartupProbe indicates that the Pod has successfully initialized. If specified, no other probes are executed until this completes successfully. If this probe fails, the Pod will be restarted, just as if the livenessProbe failed. This can be used to provide different probe parameters at the beginning of a Pod's lifecycle, when it might take a long time to load data or warm a cache, than during steady-state operation. This cannot be updated. This is an alpha feature enabled by the StartupProbe feature flag. More info: <https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes> | No |
| stdin | boolean | Whether this container should allocate a buffer for stdin in the container runtime. If this is not set, reads from stdin in the container will always result in EOF. Default is false. | No |
| stdinOnce | boolean | Whether the container runtime should close the stdin channel after it has been opened by a single attach. When stdin is true the stdin stream will remain open across multiple attach sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the first client attaches to stdin, and then remains open and accepts data until the client disconnects, at which time stdin is closed and remains closed until the container is restarted. If this flag is false, a container processes that reads from stdin will never receive an EOF. Default is false | No |
| terminationMessagePath | string | Optional: Path at which the file to which the container's termination message will be written is mounted into the container's filesystem. Message written is intended to be brief final status, such as an assertion failure message. Will be truncated by the node if greater than 4096 bytes. The total message length across all containers will be limited to 12kb. Defaults to /dev/termination-log. Cannot be updated. | No |
| terminationMessagePolicy | string | Indicate how the termination message should be populated. File will use the contents of terminationMessagePath to populate the container status message on both success and failure. FallbackToLogsOnError will use the last chunk of container log output if the termination message file is empty and the container exited with an error. The log output is limited to 2048 bytes or 80 lines, whichever is smaller. Defaults to File. Cannot be updated. | No |
| tty | boolean | Whether this container should allocate a TTY for itself, also requires 'stdin' to be true. Default is false. | No |
| volumeDevices | [ [io.k8s.api.core.v1.VolumeDevice](#io.k8s.api.core.v1.volumedevice) ] | volumeDevices is the list of block devices to be used by the container. This is a beta feature. | No |
| volumeMounts | [ [io.k8s.api.core.v1.VolumeMount](#io.k8s.api.core.v1.volumemount) ] | Pod volumes to mount into the container's filesystem. Cannot be updated. | No |
| workingDir | string | Container's working directory. If not specified, the container runtime's default will be used, which might be configured in the container image. Cannot be updated. | No |

#### io.argoproj.workflow.v1alpha1.ValueFrom

ValueFrom describes a location in which to obtain the value to a parameter

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| default | string | Default specifies a value to be used if retrieving the value from the specified source fails | No |
| event | string | Selector (<https://github.com/antonmedv/expr>) that is evaluated against the event to get the value of the parameter. E.g. `payload.message` | No |
| jqFilter | string | JQFilter expression against the resource object in resource templates | No |
| jsonPath | string | JSONPath of a resource to retrieve an output parameter value from in resource templates | No |
| parameter | string | Parameter reference to a step or dag task in which to retrieve an output parameter value from (e.g. '{{steps.mystep.outputs.myparam}}') | No |
| path | string | Path in the container to retrieve an output parameter value from in container templates | No |
| supplied | [io.argoproj.workflow.v1alpha1.SuppliedValueFrom](#io.argoproj.workflow.v1alpha1.suppliedvaluefrom) | Supplied value to be filled in directly, either through the CLI, API, etc. | No |

#### io.argoproj.workflow.v1alpha1.Version

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| buildDate | string |  | Yes |
| compiler | string |  | Yes |
| gitCommit | string |  | Yes |
| gitTag | string |  | Yes |
| gitTreeState | string |  | Yes |
| goVersion | string |  | Yes |
| platform | string |  | Yes |
| version | string |  | Yes |

#### io.argoproj.workflow.v1alpha1.Workflow

Workflow is the definition of a workflow resource

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| apiVersion | string | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: <https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#resources> | No |
| kind | string | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: <https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#types-kinds> | No |
| metadata | [io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta](#io.k8s.apimachinery.pkg.apis.meta.v1.objectmeta) |  | Yes |
| spec | [io.argoproj.workflow.v1alpha1.WorkflowSpec](#io.argoproj.workflow.v1alpha1.workflowspec) |  | Yes |
| status | [io.argoproj.workflow.v1alpha1.WorkflowStatus](#io.argoproj.workflow.v1alpha1.workflowstatus) |  | No |

#### io.argoproj.workflow.v1alpha1.WorkflowCreateRequest

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| createOptions | [io.k8s.apimachinery.pkg.apis.meta.v1.CreateOptions](#io.k8s.apimachinery.pkg.apis.meta.v1.createoptions) |  | No |
| instanceID | string | This field is no longer used. | No |
| namespace | string |  | No |
| serverDryRun | boolean (boolean) |  | No |
| workflow | [io.argoproj.workflow.v1alpha1.Workflow](#io.argoproj.workflow.v1alpha1.workflow) |  | No |

#### io.argoproj.workflow.v1alpha1.WorkflowDeleteResponse

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| io.argoproj.workflow.v1alpha1.WorkflowDeleteResponse | object |  |  |

#### io.argoproj.workflow.v1alpha1.WorkflowEventBinding

WorkflowEventBinding is the definition of an event resource

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| apiVersion | string | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: <https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#resources> | No |
| kind | string | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: <https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#types-kinds> | No |
| metadata | [io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta](#io.k8s.apimachinery.pkg.apis.meta.v1.objectmeta) |  | Yes |
| spec | [io.argoproj.workflow.v1alpha1.WorkflowEventBindingSpec](#io.argoproj.workflow.v1alpha1.workfloweventbindingspec) |  | Yes |

#### io.argoproj.workflow.v1alpha1.WorkflowEventBindingSpec

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| event | [io.argoproj.workflow.v1alpha1.Event](#io.argoproj.workflow.v1alpha1.event) | Event is the event to bind to | Yes |
| submit | [io.argoproj.workflow.v1alpha1.Submit](#io.argoproj.workflow.v1alpha1.submit) | Submit is the workflow template to submit | No |

#### io.argoproj.workflow.v1alpha1.WorkflowLintRequest

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| namespace | string |  | No |
| workflow | [io.argoproj.workflow.v1alpha1.Workflow](#io.argoproj.workflow.v1alpha1.workflow) |  | No |

#### io.argoproj.workflow.v1alpha1.WorkflowList

WorkflowList is list of Workflow resources

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| apiVersion | string | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: <https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#resources> | No |
| items | [ [io.argoproj.workflow.v1alpha1.Workflow](#io.argoproj.workflow.v1alpha1.workflow) ] |  | Yes |
| kind | string | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: <https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#types-kinds> | No |
| metadata | [io.k8s.apimachinery.pkg.apis.meta.v1.ListMeta](#io.k8s.apimachinery.pkg.apis.meta.v1.listmeta) |  | Yes |

#### io.argoproj.workflow.v1alpha1.WorkflowResubmitRequest

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| memoized | boolean (boolean) |  | No |
| name | string |  | No |
| namespace | string |  | No |

#### io.argoproj.workflow.v1alpha1.WorkflowResumeRequest

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| name | string |  | No |
| namespace | string |  | No |
| nodeFieldSelector | string |  | No |

#### io.argoproj.workflow.v1alpha1.WorkflowRetryRequest

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| name | string |  | No |
| namespace | string |  | No |
| nodeFieldSelector | string |  | No |
| restartSuccessful | boolean (boolean) |  | No |

#### io.argoproj.workflow.v1alpha1.WorkflowSetRequest

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| message | string |  | No |
| name | string |  | No |
| namespace | string |  | No |
| nodeFieldSelector | string |  | No |
| outputParameters | string |  | No |
| phase | string |  | No |

#### io.argoproj.workflow.v1alpha1.WorkflowSpec

WorkflowSpec is the specification of a Workflow.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| activeDeadlineSeconds | long | Optional duration in seconds relative to the workflow start time which the workflow is allowed to run before the controller terminates the io.argoproj.workflow.v1alpha1. A value of zero is used to terminate a Running workflow | No |
| affinity | [io.k8s.api.core.v1.Affinity](#io.k8s.api.core.v1.affinity) | Affinity sets the scheduling constraints for all pods in the io.argoproj.workflow.v1alpha1. Can be overridden by an affinity specified in the template | No |
| arguments | [io.argoproj.workflow.v1alpha1.Arguments](#io.argoproj.workflow.v1alpha1.arguments) | Arguments contain the parameters and artifacts sent to the workflow entrypoint Parameters are referencable globally using the 'workflow' variable prefix. e.g. {{io.argoproj.workflow.v1alpha1.parameters.myparam}} | No |
| artifactRepositoryRef | [io.argoproj.workflow.v1alpha1.ArtifactRepositoryRef](#io.argoproj.workflow.v1alpha1.artifactrepositoryref) | ArtifactRepositoryRef specifies the configMap name and key containing the artifact repository config. | No |
| automountServiceAccountToken | boolean | AutomountServiceAccountToken indicates whether a service account token should be automatically mounted in pods. ServiceAccountName of ExecutorConfig must be specified if this value is false. | No |
| dnsConfig | [io.k8s.api.core.v1.PodDNSConfig](#io.k8s.api.core.v1.poddnsconfig) | PodDNSConfig defines the DNS parameters of a pod in addition to those generated from DNSPolicy. | No |
| dnsPolicy | string | Set DNS policy for the pod. Defaults to "ClusterFirst". Valid values are 'ClusterFirstWithHostNet', 'ClusterFirst', 'Default' or 'None'. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy. To have DNS options set along with hostNetwork, you have to specify DNS policy explicitly to 'ClusterFirstWithHostNet'. | No |
| entrypoint | string | Entrypoint is a template reference to the starting point of the io.argoproj.workflow.v1alpha1. | No |
| executor | [io.argoproj.workflow.v1alpha1.ExecutorConfig](#io.argoproj.workflow.v1alpha1.executorconfig) | Executor holds configurations of executor containers of the io.argoproj.workflow.v1alpha1. | No |
| hostAliases | [ [io.k8s.api.core.v1.HostAlias](#io.k8s.api.core.v1.hostalias) ] |  | No |
| hostNetwork | boolean | Host networking requested for this workflow pod. Default to false. | No |
| imagePullSecrets | [ [io.k8s.api.core.v1.LocalObjectReference](#io.k8s.api.core.v1.localobjectreference) ] | ImagePullSecrets is a list of references to secrets in the same namespace to use for pulling any images in pods that reference this ServiceAccount. ImagePullSecrets are distinct from Secrets because Secrets can be mounted in the pod, but ImagePullSecrets are only accessed by the kubelet. More info: <https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod> | No |
| metrics | [io.argoproj.workflow.v1alpha1.Metrics](#io.argoproj.workflow.v1alpha1.metrics) | Metrics are a list of metrics emitted from this Workflow | No |
| nodeSelector | object | NodeSelector is a selector which will result in all pods of the workflow to be scheduled on the selected node(s). This is able to be overridden by a nodeSelector specified in the template. | No |
| onExit | string | OnExit is a template reference which is invoked at the end of the workflow, irrespective of the success, failure, or error of the primary io.argoproj.workflow.v1alpha1. | No |
| parallelism | long | Parallelism limits the max total parallel pods that can execute at the same time in a workflow | No |
| podDisruptionBudget | [io.k8s.api.policy.v1beta1.PodDisruptionBudgetSpec](#io.k8s.api.policy.v1beta1.poddisruptionbudgetspec) | PodDisruptionBudget holds the number of concurrent disruptions that you allow for Workflow's Pods. Controller will automatically add the selector with workflow name, if selector is empty. Optional: Defaults to empty. | No |
| podGC | [io.argoproj.workflow.v1alpha1.PodGC](#io.argoproj.workflow.v1alpha1.podgc) | PodGC describes the strategy to use when to deleting completed pods | No |
| podPriority | integer | Priority to apply to workflow pods. | No |
| podPriorityClassName | string | PriorityClassName to apply to workflow pods. | No |
| podSpecPatch | string | PodSpecPatch holds strategic merge patch to apply against the pod spec. Allows parameterization of container fields which are not strings (e.g. resource limits). | No |
| priority | integer | Priority is used if controller is configured to process limited number of workflows in parallel. Workflows with higher priority are processed first. | No |
| schedulerName | string | Set scheduler name for all pods. Will be overridden if container/script template's scheduler name is set. Default scheduler will be used if neither specified. | No |
| securityContext | [io.k8s.api.core.v1.PodSecurityContext](#io.k8s.api.core.v1.podsecuritycontext) | SecurityContext holds pod-level security attributes and common container settings. Optional: Defaults to empty.  See type description for default values of each field. | No |
| serviceAccountName | string | ServiceAccountName is the name of the ServiceAccount to run all pods of the workflow as. | No |
| shutdown | string | Shutdown will shutdown the workflow according to its ShutdownStrategy | No |
| suspend | boolean | Suspend will suspend the workflow and prevent execution of any future steps in the workflow | No |
| synchronization | [io.argoproj.workflow.v1alpha1.Synchronization](#io.argoproj.workflow.v1alpha1.synchronization) | Synchronization holds synchronization lock configuration for this Workflow | No |
| templates | [ [io.argoproj.workflow.v1alpha1.Template](#io.argoproj.workflow.v1alpha1.template) ] | Templates is a list of workflow templates used in a workflow | No |
| tolerations | [ [io.k8s.api.core.v1.Toleration](#io.k8s.api.core.v1.toleration) ] | Tolerations to apply to workflow pods. | No |
| ttlSecondsAfterFinished | integer | TTLSecondsAfterFinished limits the lifetime of a Workflow that has finished execution (Succeeded, Failed, Error). If this field is set, once the Workflow finishes, it will be deleted after ttlSecondsAfterFinished expires. If this field is unset, ttlSecondsAfterFinished will not expire. If this field is set to zero, ttlSecondsAfterFinished expires immediately after the Workflow finishes. DEPRECATED: Use TTLStrategy.SecondsAfterCompletion instead. | No |
| ttlStrategy | [io.argoproj.workflow.v1alpha1.TTLStrategy](#io.argoproj.workflow.v1alpha1.ttlstrategy) | TTLStrategy limits the lifetime of a Workflow that has finished execution depending on if it Succeeded or Failed. If this struct is set, once the Workflow finishes, it will be deleted after the time to live expires. If this field is unset, the controller config map will hold the default values. | No |
| volumeClaimTemplates | [ [io.k8s.api.core.v1.PersistentVolumeClaim](#io.k8s.api.core.v1.persistentvolumeclaim) ] | VolumeClaimTemplates is a list of claims that containers are allowed to reference. The Workflow controller will create the claims at the beginning of the workflow and delete the claims upon completion of the workflow | No |
| volumes | [ [io.k8s.api.core.v1.Volume](#io.k8s.api.core.v1.volume) ] | Volumes is a list of volumes that can be mounted by containers in a io.argoproj.workflow.v1alpha1. | No |
| workflowTemplateRef | [io.argoproj.workflow.v1alpha1.WorkflowTemplateRef](#io.argoproj.workflow.v1alpha1.workflowtemplateref) | WorkflowTemplateRef holds a reference to a WorkflowTemplate for execution | No |

#### io.argoproj.workflow.v1alpha1.WorkflowStatus

WorkflowStatus contains overall status information about a workflow

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| compressedNodes | string | Compressed and base64 decoded Nodes map | No |
| conditions | [ [io.argoproj.workflow.v1alpha1.Condition](#io.argoproj.workflow.v1alpha1.condition) ] | Conditions is a list of conditions the Workflow may have | No |
| finishedAt | [io.k8s.apimachinery.pkg.apis.meta.v1.Time](#io.k8s.apimachinery.pkg.apis.meta.v1.time) | Time at which this workflow completed | No |
| message | string | A human readable message indicating details about why the workflow is in this condition. | No |
| nodes | object | Nodes is a mapping between a node ID and the node's status. | No |
| offloadNodeStatusVersion | string | Whether on not node status has been offloaded to a database. If exists, then Nodes and CompressedNodes will be empty. This will actually be populated with a hash of the offloaded data. | No |
| outputs | [io.argoproj.workflow.v1alpha1.Outputs](#io.argoproj.workflow.v1alpha1.outputs) | Outputs captures output values and artifact locations produced by the workflow via global outputs | No |
| persistentVolumeClaims | [ [io.k8s.api.core.v1.Volume](#io.k8s.api.core.v1.volume) ] | PersistentVolumeClaims tracks all PVCs that were created as part of the io.argoproj.workflow.v1alpha1. The contents of this list are drained at the end of the workflow. | No |
| phase | string | Phase a simple, high-level summary of where the workflow is in its lifecycle. | No |
| resourcesDuration | object | ResourcesDuration is the total for the workflow | No |
| startedAt | [io.k8s.apimachinery.pkg.apis.meta.v1.Time](#io.k8s.apimachinery.pkg.apis.meta.v1.time) | Time at which this workflow started | No |
| storedTemplates | object | StoredTemplates is a mapping between a template ref and the node's status. | No |
| storedWorkflowTemplateSpec | [io.argoproj.workflow.v1alpha1.WorkflowSpec](#io.argoproj.workflow.v1alpha1.workflowspec) | StoredWorkflowSpec stores the WorkflowTemplate spec for future execution. | No |
| synchronization | [io.argoproj.workflow.v1alpha1.SynchronizationStatus](#io.argoproj.workflow.v1alpha1.synchronizationstatus) | Synchronization stores the status of synchronization locks | No |

#### io.argoproj.workflow.v1alpha1.WorkflowStep

WorkflowStep is a reference to a template to execute in a series of step

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| arguments | [io.argoproj.workflow.v1alpha1.Arguments](#io.argoproj.workflow.v1alpha1.arguments) | Arguments hold arguments to the template | No |
| continueOn | [io.argoproj.workflow.v1alpha1.ContinueOn](#io.argoproj.workflow.v1alpha1.continueon) | ContinueOn makes argo to proceed with the following step even if this step fails. Errors and Failed states can be specified | No |
| name | string | Name of the step | No |
| onExit | string | OnExit is a template reference which is invoked at the end of the template, irrespective of the success, failure, or error of the primary template. | No |
| template | string | Template is the name of the template to execute as the step | No |
| templateRef | [io.argoproj.workflow.v1alpha1.TemplateRef](#io.argoproj.workflow.v1alpha1.templateref) | TemplateRef is the reference to the template resource to execute as the step. | No |
| when | string | When is an expression in which the step should conditionally execute | No |
| withItems | [ [io.argoproj.workflow.v1alpha1.Item](#io.argoproj.workflow.v1alpha1.item) ] | WithItems expands a step into multiple parallel steps from the items in the list | No |
| withParam | string | WithParam expands a step into multiple parallel steps from the value in the parameter, which is expected to be a JSON list. | No |
| withSequence | [io.argoproj.workflow.v1alpha1.Sequence](#io.argoproj.workflow.v1alpha1.sequence) | WithSequence expands a step into a numeric sequence | No |

#### io.argoproj.workflow.v1alpha1.WorkflowStopRequest

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| message | string |  | No |
| name | string |  | No |
| namespace | string |  | No |
| nodeFieldSelector | string |  | No |

#### io.argoproj.workflow.v1alpha1.WorkflowSubmitRequest

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| namespace | string |  | No |
| resourceKind | string |  | No |
| resourceName | string |  | No |
| submitOptions | [io.argoproj.workflow.v1alpha1.SubmitOpts](#io.argoproj.workflow.v1alpha1.submitopts) |  | No |

#### io.argoproj.workflow.v1alpha1.WorkflowSuspendRequest

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| name | string |  | No |
| namespace | string |  | No |

#### io.argoproj.workflow.v1alpha1.WorkflowTemplate

WorkflowTemplate is the definition of a workflow template resource

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| apiVersion | string | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: <https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#resources> | No |
| kind | string | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: <https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#types-kinds> | No |
| metadata | [io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta](#io.k8s.apimachinery.pkg.apis.meta.v1.objectmeta) |  | Yes |
| spec | [io.argoproj.workflow.v1alpha1.WorkflowTemplateSpec](#io.argoproj.workflow.v1alpha1.workflowtemplatespec) |  | Yes |

#### io.argoproj.workflow.v1alpha1.WorkflowTemplateCreateRequest

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| createOptions | [io.k8s.apimachinery.pkg.apis.meta.v1.CreateOptions](#io.k8s.apimachinery.pkg.apis.meta.v1.createoptions) |  | No |
| namespace | string |  | No |
| template | [io.argoproj.workflow.v1alpha1.WorkflowTemplate](#io.argoproj.workflow.v1alpha1.workflowtemplate) |  | No |

#### io.argoproj.workflow.v1alpha1.WorkflowTemplateDeleteResponse

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| io.argoproj.workflow.v1alpha1.WorkflowTemplateDeleteResponse | object |  |  |

#### io.argoproj.workflow.v1alpha1.WorkflowTemplateLintRequest

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| createOptions | [io.k8s.apimachinery.pkg.apis.meta.v1.CreateOptions](#io.k8s.apimachinery.pkg.apis.meta.v1.createoptions) |  | No |
| namespace | string |  | No |
| template | [io.argoproj.workflow.v1alpha1.WorkflowTemplate](#io.argoproj.workflow.v1alpha1.workflowtemplate) |  | No |

#### io.argoproj.workflow.v1alpha1.WorkflowTemplateList

WorkflowTemplateList is list of WorkflowTemplate resources

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| apiVersion | string | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: <https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#resources> | No |
| items | [ [io.argoproj.workflow.v1alpha1.WorkflowTemplate](#io.argoproj.workflow.v1alpha1.workflowtemplate) ] |  | Yes |
| kind | string | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: <https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#types-kinds> | No |
| metadata | [io.k8s.apimachinery.pkg.apis.meta.v1.ListMeta](#io.k8s.apimachinery.pkg.apis.meta.v1.listmeta) |  | Yes |

#### io.argoproj.workflow.v1alpha1.WorkflowTemplateRef

WorkflowTemplateRef is a reference to a WorkflowTemplate resource.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| clusterScope | boolean | ClusterScope indicates the referred template is cluster scoped (i.e. a ClusterWorkflowTemplate). | No |
| name | string | Name is the resource name of the workflow template. | No |

#### io.argoproj.workflow.v1alpha1.WorkflowTemplateSpec

WorkflowTemplateSpec is a spec of WorkflowTemplate.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| activeDeadlineSeconds | long | Optional duration in seconds relative to the workflow start time which the workflow is allowed to run before the controller terminates the io.argoproj.workflow.v1alpha1. A value of zero is used to terminate a Running workflow | No |
| affinity | [io.k8s.api.core.v1.Affinity](#io.k8s.api.core.v1.affinity) | Affinity sets the scheduling constraints for all pods in the io.argoproj.workflow.v1alpha1. Can be overridden by an affinity specified in the template | No |
| arguments | [io.argoproj.workflow.v1alpha1.Arguments](#io.argoproj.workflow.v1alpha1.arguments) | Arguments contain the parameters and artifacts sent to the workflow entrypoint Parameters are referencable globally using the 'workflow' variable prefix. e.g. {{io.argoproj.workflow.v1alpha1.parameters.myparam}} | No |
| artifactRepositoryRef | [io.argoproj.workflow.v1alpha1.ArtifactRepositoryRef](#io.argoproj.workflow.v1alpha1.artifactrepositoryref) | ArtifactRepositoryRef specifies the configMap name and key containing the artifact repository config. | No |
| automountServiceAccountToken | boolean | AutomountServiceAccountToken indicates whether a service account token should be automatically mounted in pods. ServiceAccountName of ExecutorConfig must be specified if this value is false. | No |
| dnsConfig | [io.k8s.api.core.v1.PodDNSConfig](#io.k8s.api.core.v1.poddnsconfig) | PodDNSConfig defines the DNS parameters of a pod in addition to those generated from DNSPolicy. | No |
| dnsPolicy | string | Set DNS policy for the pod. Defaults to "ClusterFirst". Valid values are 'ClusterFirstWithHostNet', 'ClusterFirst', 'Default' or 'None'. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy. To have DNS options set along with hostNetwork, you have to specify DNS policy explicitly to 'ClusterFirstWithHostNet'. | No |
| entrypoint | string | Entrypoint is a template reference to the starting point of the io.argoproj.workflow.v1alpha1. | No |
| executor | [io.argoproj.workflow.v1alpha1.ExecutorConfig](#io.argoproj.workflow.v1alpha1.executorconfig) | Executor holds configurations of executor containers of the io.argoproj.workflow.v1alpha1. | No |
| hostAliases | [ [io.k8s.api.core.v1.HostAlias](#io.k8s.api.core.v1.hostalias) ] |  | No |
| hostNetwork | boolean | Host networking requested for this workflow pod. Default to false. | No |
| imagePullSecrets | [ [io.k8s.api.core.v1.LocalObjectReference](#io.k8s.api.core.v1.localobjectreference) ] | ImagePullSecrets is a list of references to secrets in the same namespace to use for pulling any images in pods that reference this ServiceAccount. ImagePullSecrets are distinct from Secrets because Secrets can be mounted in the pod, but ImagePullSecrets are only accessed by the kubelet. More info: <https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod> | No |
| metrics | [io.argoproj.workflow.v1alpha1.Metrics](#io.argoproj.workflow.v1alpha1.metrics) | Metrics are a list of metrics emitted from this Workflow | No |
| nodeSelector | object | NodeSelector is a selector which will result in all pods of the workflow to be scheduled on the selected node(s). This is able to be overridden by a nodeSelector specified in the template. | No |
| onExit | string | OnExit is a template reference which is invoked at the end of the workflow, irrespective of the success, failure, or error of the primary io.argoproj.workflow.v1alpha1. | No |
| parallelism | long | Parallelism limits the max total parallel pods that can execute at the same time in a workflow | No |
| podDisruptionBudget | [io.k8s.api.policy.v1beta1.PodDisruptionBudgetSpec](#io.k8s.api.policy.v1beta1.poddisruptionbudgetspec) | PodDisruptionBudget holds the number of concurrent disruptions that you allow for Workflow's Pods. Controller will automatically add the selector with workflow name, if selector is empty. Optional: Defaults to empty. | No |
| podGC | [io.argoproj.workflow.v1alpha1.PodGC](#io.argoproj.workflow.v1alpha1.podgc) | PodGC describes the strategy to use when to deleting completed pods | No |
| podPriority | integer | Priority to apply to workflow pods. | No |
| podPriorityClassName | string | PriorityClassName to apply to workflow pods. | No |
| podSpecPatch | string | PodSpecPatch holds strategic merge patch to apply against the pod spec. Allows parameterization of container fields which are not strings (e.g. resource limits). | No |
| priority | integer | Priority is used if controller is configured to process limited number of workflows in parallel. Workflows with higher priority are processed first. | No |
| schedulerName | string | Set scheduler name for all pods. Will be overridden if container/script template's scheduler name is set. Default scheduler will be used if neither specified. | No |
| securityContext | [io.k8s.api.core.v1.PodSecurityContext](#io.k8s.api.core.v1.podsecuritycontext) | SecurityContext holds pod-level security attributes and common container settings. Optional: Defaults to empty.  See type description for default values of each field. | No |
| serviceAccountName | string | ServiceAccountName is the name of the ServiceAccount to run all pods of the workflow as. | No |
| shutdown | string | Shutdown will shutdown the workflow according to its ShutdownStrategy | No |
| suspend | boolean | Suspend will suspend the workflow and prevent execution of any future steps in the workflow | No |
| synchronization | [io.argoproj.workflow.v1alpha1.Synchronization](#io.argoproj.workflow.v1alpha1.synchronization) | Synchronization holds synchronization lock configuration for this Workflow | No |
| templates | [ [io.argoproj.workflow.v1alpha1.Template](#io.argoproj.workflow.v1alpha1.template) ] | Templates is a list of workflow templates used in a workflow | No |
| tolerations | [ [io.k8s.api.core.v1.Toleration](#io.k8s.api.core.v1.toleration) ] | Tolerations to apply to workflow pods. | No |
| ttlSecondsAfterFinished | integer | TTLSecondsAfterFinished limits the lifetime of a Workflow that has finished execution (Succeeded, Failed, Error). If this field is set, once the Workflow finishes, it will be deleted after ttlSecondsAfterFinished expires. If this field is unset, ttlSecondsAfterFinished will not expire. If this field is set to zero, ttlSecondsAfterFinished expires immediately after the Workflow finishes. DEPRECATED: Use TTLStrategy.SecondsAfterCompletion instead. | No |
| ttlStrategy | [io.argoproj.workflow.v1alpha1.TTLStrategy](#io.argoproj.workflow.v1alpha1.ttlstrategy) | TTLStrategy limits the lifetime of a Workflow that has finished execution depending on if it Succeeded or Failed. If this struct is set, once the Workflow finishes, it will be deleted after the time to live expires. If this field is unset, the controller config map will hold the default values. | No |
| volumeClaimTemplates | [ [io.k8s.api.core.v1.PersistentVolumeClaim](#io.k8s.api.core.v1.persistentvolumeclaim) ] | VolumeClaimTemplates is a list of claims that containers are allowed to reference. The Workflow controller will create the claims at the beginning of the workflow and delete the claims upon completion of the workflow | No |
| volumes | [ [io.k8s.api.core.v1.Volume](#io.k8s.api.core.v1.volume) ] | Volumes is a list of volumes that can be mounted by containers in a io.argoproj.workflow.v1alpha1. | No |
| workflowMetadata | [io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta](#io.k8s.apimachinery.pkg.apis.meta.v1.objectmeta) | WorkflowMetadata contains some metadata of the workflow to be refer | No |
| workflowTemplateRef | [io.argoproj.workflow.v1alpha1.WorkflowTemplateRef](#io.argoproj.workflow.v1alpha1.workflowtemplateref) | WorkflowTemplateRef holds a reference to a WorkflowTemplate for execution | No |

#### io.argoproj.workflow.v1alpha1.WorkflowTemplateUpdateRequest

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| name | string | DEPRECATED: This field is ignored. | No |
| namespace | string |  | No |
| template | [io.argoproj.workflow.v1alpha1.WorkflowTemplate](#io.argoproj.workflow.v1alpha1.workflowtemplate) |  | No |

#### io.argoproj.workflow.v1alpha1.WorkflowTerminateRequest

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| name | string |  | No |
| namespace | string |  | No |

#### io.argoproj.workflow.v1alpha1.WorkflowWatchEvent

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| object | [io.argoproj.workflow.v1alpha1.Workflow](#io.argoproj.workflow.v1alpha1.workflow) |  | No |
| type | string |  | No |

#### io.k8s.api.core.v1.AWSElasticBlockStoreVolumeSource

Represents a Persistent Disk resource in AWS.

An AWS EBS disk must exist before mounting to a container. The disk must also be in the same AWS zone as the kubelet. An AWS EBS disk can only be mounted as read/write once. AWS EBS volumes support ownership management and SELinux relabeling.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| fsType | string | Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: <https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore> | No |
| partition | integer | The partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as "1". Similarly, the volume partition for /dev/sda is "0" (or you can leave the property empty). | No |
| readOnly | boolean | Specify "true" to force and set the ReadOnly property in VolumeMounts to "true". If omitted, the default is "false". More info: <https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore> | No |
| volumeID | string | Unique ID of the persistent disk resource in AWS (Amazon EBS volume). More info: <https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore> | Yes |

#### io.k8s.api.core.v1.Affinity

Affinity is a group of affinity scheduling rules.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| nodeAffinity | [io.k8s.api.core.v1.NodeAffinity](#io.k8s.api.core.v1.nodeaffinity) | Describes node affinity scheduling rules for the pod. | No |
| podAffinity | [io.k8s.api.core.v1.PodAffinity](#io.k8s.api.core.v1.podaffinity) | Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)). | No |
| podAntiAffinity | [io.k8s.api.core.v1.PodAntiAffinity](#io.k8s.api.core.v1.podantiaffinity) | Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)). | No |

#### io.k8s.api.core.v1.AzureDiskVolumeSource

AzureDisk represents an Azure Data Disk mount on the host and bind mount to the pod.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| cachingMode | string | Host Caching mode: None, Read Only, Read Write. | No |
| diskName | string | The Name of the data disk in the blob storage | Yes |
| diskURI | string | The URI the data disk in the blob storage | Yes |
| fsType | string | Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. | No |
| kind | string | Expected values Shared: multiple blob disks per storage account  Dedicated: single blob disk per storage account  Managed: azure managed data disk (only in managed availability set). defaults to shared | No |
| readOnly | boolean | Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. | No |

#### io.k8s.api.core.v1.AzureFileVolumeSource

AzureFile represents an Azure File Service mount on the host and bind mount to the pod.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| readOnly | boolean | Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. | No |
| secretName | string | the name of secret that contains Azure Storage Account Name and Key | Yes |
| shareName | string | Share Name | Yes |

#### io.k8s.api.core.v1.CSIVolumeSource

Represents a source location of a volume to mount, managed by an external CSI driver

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| driver | string | Driver is the name of the CSI driver that handles this volume. Consult with your admin for the correct name as registered in the cluster. | Yes |
| fsType | string | Filesystem type to mount. Ex. "ext4", "xfs", "ntfs". If not provided, the empty value is passed to the associated CSI driver which will determine the default filesystem to apply. | No |
| nodePublishSecretRef | [io.k8s.api.core.v1.LocalObjectReference](#io.k8s.api.core.v1.localobjectreference) | NodePublishSecretRef is a reference to the secret object containing sensitive information to pass to the CSI driver to complete the CSI NodePublishVolume and NodeUnpublishVolume calls. This field is optional, and  may be empty if no secret is required. If the secret object contains more than one secret, all secret references are passed. | No |
| readOnly | boolean | Specifies a read-only configuration for the volume. Defaults to false (read/write). | No |
| volumeAttributes | object | VolumeAttributes stores driver-specific properties that are passed to the CSI driver. Consult your driver's documentation for supported values. | No |

#### io.k8s.api.core.v1.Capabilities

Adds and removes POSIX capabilities from running containers.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| add | [ string ] | Added capabilities | No |
| drop | [ string ] | Removed capabilities | No |

#### io.k8s.api.core.v1.CephFSVolumeSource

Represents a Ceph Filesystem mount that lasts the lifetime of a pod Cephfs volumes do not support ownership management or SELinux relabeling.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| monitors | [ string ] | Required: Monitors is a collection of Ceph monitors More info: <https://releases.k8s.io/HEAD/examples/volumes/cephfs/README.md#how-to-use-it> | Yes |
| path | string | Optional: Used as the mounted root, rather than the full Ceph tree, default is / | No |
| readOnly | boolean | Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: <https://releases.k8s.io/HEAD/examples/volumes/cephfs/README.md#how-to-use-it> | No |
| secretFile | string | Optional: SecretFile is the path to key ring for User, default is /etc/ceph/user.secret More info: <https://releases.k8s.io/HEAD/examples/volumes/cephfs/README.md#how-to-use-it> | No |
| secretRef | [io.k8s.api.core.v1.LocalObjectReference](#io.k8s.api.core.v1.localobjectreference) | Optional: SecretRef is reference to the authentication secret for User, default is empty. More info: <https://releases.k8s.io/HEAD/examples/volumes/cephfs/README.md#how-to-use-it> | No |
| user | string | Optional: User is the rados user name, default is admin More info: <https://releases.k8s.io/HEAD/examples/volumes/cephfs/README.md#how-to-use-it> | No |

#### io.k8s.api.core.v1.CinderVolumeSource

Represents a cinder volume resource in Openstack. A Cinder volume must exist before mounting to a container. The volume must also be in the same region as the kubelet. Cinder volumes support ownership management and SELinux relabeling.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| fsType | string | Filesystem type to mount. Must be a filesystem type supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: <https://releases.k8s.io/HEAD/examples/mysql-cinder-pd/README.md> | No |
| readOnly | boolean | Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: <https://releases.k8s.io/HEAD/examples/mysql-cinder-pd/README.md> | No |
| secretRef | [io.k8s.api.core.v1.LocalObjectReference](#io.k8s.api.core.v1.localobjectreference) | Optional: points to a secret object containing parameters used to connect to OpenStack. | No |
| volumeID | string | volume id used to identify the volume in cinder More info: <https://releases.k8s.io/HEAD/examples/mysql-cinder-pd/README.md> | Yes |

#### io.k8s.api.core.v1.ConfigMapEnvSource

ConfigMapEnvSource selects a ConfigMap to populate the environment variables with.

The contents of the target ConfigMap's Data field will represent the key-value pairs as environment variables.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| name | string | Name of the referent. More info: <https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names> | No |
| optional | boolean | Specify whether the ConfigMap must be defined | No |

#### io.k8s.api.core.v1.ConfigMapKeySelector

Selects a key from a ConfigMap.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| key | string | The key to select. | Yes |
| name | string | Name of the referent. More info: <https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names> | No |
| optional | boolean | Specify whether the ConfigMap or its key must be defined | No |

#### io.k8s.api.core.v1.ConfigMapProjection

Adapts a ConfigMap into a projected volume.

The contents of the target ConfigMap's Data field will be presented in a projected volume as files using the keys in the Data field as the file names, unless the items element is populated with specific mappings of keys to paths. Note that this is identical to a configmap volume source without the default mode.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| items | [ [io.k8s.api.core.v1.KeyToPath](#io.k8s.api.core.v1.keytopath) ] | If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'. | No |
| name | string | Name of the referent. More info: <https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names> | No |
| optional | boolean | Specify whether the ConfigMap or its keys must be defined | No |

#### io.k8s.api.core.v1.ConfigMapVolumeSource

Adapts a ConfigMap into a volume.

The contents of the target ConfigMap's Data field will be presented in a volume as files using the keys in the Data field as the file names, unless the items element is populated with specific mappings of keys to paths. ConfigMap volumes support ownership management and SELinux relabeling.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| defaultMode | integer | Optional: mode bits to use on created files by default. Must be a value between 0 and 0777. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set. | No |
| items | [ [io.k8s.api.core.v1.KeyToPath](#io.k8s.api.core.v1.keytopath) ] | If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'. | No |
| name | string | Name of the referent. More info: <https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names> | No |
| optional | boolean | Specify whether the ConfigMap or its keys must be defined | No |

#### io.k8s.api.core.v1.Container

A single application container that you want to run within a pod.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| args | [ string ] | Arguments to the entrypoint. The docker image's CMD is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: <https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell> | No |
| command | [ string ] | Entrypoint array. Not executed within a shell. The docker image's ENTRYPOINT is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: <https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell> | No |
| env | [ [io.k8s.api.core.v1.EnvVar](#io.k8s.api.core.v1.envvar) ] | List of environment variables to set in the container. Cannot be updated. | No |
| envFrom | [ [io.k8s.api.core.v1.EnvFromSource](#io.k8s.api.core.v1.envfromsource) ] | List of sources to populate environment variables in the container. The keys defined within a source must be a C_IDENTIFIER. All invalid keys will be reported as an event when the container is starting. When a key exists in multiple sources, the value associated with the last source will take precedence. Values defined by an Env with a duplicate key will take precedence. Cannot be updated. | No |
| image | string | Docker image name. More info: <https://kubernetes.io/docs/concepts/containers/images> This field is optional to allow higher level config management to default or override container images in workload controllers like Deployments and StatefulSets. | Yes |
| imagePullPolicy | string | Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: <https://kubernetes.io/docs/concepts/containers/images#updating-images> | No |
| lifecycle | [io.k8s.api.core.v1.Lifecycle](#io.k8s.api.core.v1.lifecycle) | Actions that the management system should take in response to container lifecycle events. Cannot be updated. | No |
| livenessProbe | [io.k8s.api.core.v1.Probe](#io.k8s.api.core.v1.probe) | Periodic probe of container liveness. Container will be restarted if the probe fails. Cannot be updated. More info: <https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes> | No |
| name | string | Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated. | No |
| ports | [ [io.k8s.api.core.v1.ContainerPort](#io.k8s.api.core.v1.containerport) ] | List of ports to expose from the container. Exposing a port here gives the system additional information about the network connections a container uses, but is primarily informational. Not specifying a port here DOES NOT prevent that port from being exposed. Any port which is listening on the default "0.0.0.0" address inside a container will be accessible from the network. Cannot be updated. | No |
| readinessProbe | [io.k8s.api.core.v1.Probe](#io.k8s.api.core.v1.probe) | Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. Cannot be updated. More info: <https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes> | No |
| resources | [io.k8s.api.core.v1.ResourceRequirements](#io.k8s.api.core.v1.resourcerequirements) | Compute Resources required by this container. Cannot be updated. More info: <https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/> | No |
| securityContext | [io.k8s.api.core.v1.SecurityContext](#io.k8s.api.core.v1.securitycontext) | Security options the pod should run with. More info: <https://kubernetes.io/docs/concepts/policy/security-context/> More info: <https://kubernetes.io/docs/tasks/configure-pod-container/security-context/> | No |
| stdin | boolean | Whether this container should allocate a buffer for stdin in the container runtime. If this is not set, reads from stdin in the container will always result in EOF. Default is false. | No |
| stdinOnce | boolean | Whether the container runtime should close the stdin channel after it has been opened by a single attach. When stdin is true the stdin stream will remain open across multiple attach sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the first client attaches to stdin, and then remains open and accepts data until the client disconnects, at which time stdin is closed and remains closed until the container is restarted. If this flag is false, a container processes that reads from stdin will never receive an EOF. Default is false | No |
| terminationMessagePath | string | Optional: Path at which the file to which the container's termination message will be written is mounted into the container's filesystem. Message written is intended to be brief final status, such as an assertion failure message. Will be truncated by the node if greater than 4096 bytes. The total message length across all containers will be limited to 12kb. Defaults to /dev/termination-log. Cannot be updated. | No |
| terminationMessagePolicy | string | Indicate how the termination message should be populated. File will use the contents of terminationMessagePath to populate the container status message on both success and failure. FallbackToLogsOnError will use the last chunk of container log output if the termination message file is empty and the container exited with an error. The log output is limited to 2048 bytes or 80 lines, whichever is smaller. Defaults to File. Cannot be updated. | No |
| tty | boolean | Whether this container should allocate a TTY for itself, also requires 'stdin' to be true. Default is false. | No |
| volumeDevices | [ [io.k8s.api.core.v1.VolumeDevice](#io.k8s.api.core.v1.volumedevice) ] | volumeDevices is the list of block devices to be used by the container. This is a beta feature. | No |
| volumeMounts | [ [io.k8s.api.core.v1.VolumeMount](#io.k8s.api.core.v1.volumemount) ] | Pod volumes to mount into the container's filesystem. Cannot be updated. | No |
| workingDir | string | Container's working directory. If not specified, the container runtime's default will be used, which might be configured in the container image. Cannot be updated. | No |

#### io.k8s.api.core.v1.ContainerPort

ContainerPort represents a network port in a single container.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| containerPort | integer | Number of port to expose on the pod's IP address. This must be a valid port number, 0 < x < 65536. | Yes |
| hostIP | string | What host IP to bind the external port to. | No |
| hostPort | integer | Number of port to expose on the host. If specified, this must be a valid port number, 0 < x < 65536. If HostNetwork is specified, this must match ContainerPort. Most containers do not need this. | No |
| name | string | If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services. | No |
| protocol | string | Protocol for port. Must be UDP, TCP, or SCTP. Defaults to "TCP". | No |

#### io.k8s.api.core.v1.DownwardAPIProjection

Represents downward API info for projecting into a projected volume. Note that this is identical to a downwardAPI volume source without the default mode.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| items | [ [io.k8s.api.core.v1.DownwardAPIVolumeFile](#io.k8s.api.core.v1.downwardapivolumefile) ] | Items is a list of DownwardAPIVolume file | No |

#### io.k8s.api.core.v1.DownwardAPIVolumeFile

DownwardAPIVolumeFile represents information to create the file containing the pod field

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| fieldRef | [io.k8s.api.core.v1.ObjectFieldSelector](#io.k8s.api.core.v1.objectfieldselector) | Required: Selects a field of the pod: only annotations, labels, name and namespace are supported. | No |
| mode | integer | Optional: mode bits to use on this file, must be a value between 0 and 0777. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set. | No |
| path | string | Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..' | Yes |
| resourceFieldRef | [io.k8s.api.core.v1.ResourceFieldSelector](#io.k8s.api.core.v1.resourcefieldselector) | Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported. | No |

#### io.k8s.api.core.v1.DownwardAPIVolumeSource

DownwardAPIVolumeSource represents a volume containing downward API info. Downward API volumes support ownership management and SELinux relabeling.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| defaultMode | integer | Optional: mode bits to use on created files by default. Must be a value between 0 and 0777. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set. | No |
| items | [ [io.k8s.api.core.v1.DownwardAPIVolumeFile](#io.k8s.api.core.v1.downwardapivolumefile) ] | Items is a list of downward API volume file | No |

#### io.k8s.api.core.v1.EmptyDirVolumeSource

Represents an empty directory for a pod. Empty directory volumes support ownership management and SELinux relabeling.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| medium | string | What type of storage medium should back this directory. The default is "" which means to use the node's default medium. Must be an empty string (default) or Memory. More info: <https://kubernetes.io/docs/concepts/storage/volumes#emptydir> | No |
| sizeLimit | [io.k8s.apimachinery.pkg.api.resource.Quantity](#io.k8s.apimachinery.pkg.api.resource.quantity) | Total amount of local storage required for this EmptyDir volume. The size limit is also applicable for memory medium. The maximum usage on memory medium EmptyDir would be the minimum value between the SizeLimit specified here and the sum of memory limits of all containers in a pod. The default is nil which means that the limit is undefined. More info: <http://kubernetes.io/docs/user-guide/volumes#emptydir> | No |

#### io.k8s.api.core.v1.EnvFromSource

EnvFromSource represents the source of a set of ConfigMaps

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| configMapRef | [io.k8s.api.core.v1.ConfigMapEnvSource](#io.k8s.api.core.v1.configmapenvsource) | The ConfigMap to select from | No |
| prefix | string | An optional identifier to prepend to each key in the ConfigMap. Must be a C_IDENTIFIER. | No |
| secretRef | [io.k8s.api.core.v1.SecretEnvSource](#io.k8s.api.core.v1.secretenvsource) | The Secret to select from | No |

#### io.k8s.api.core.v1.EnvVar

EnvVar represents an environment variable present in a Container.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| name | string | Name of the environment variable. Must be a C_IDENTIFIER. | Yes |
| value | string | Variable references $(VAR_NAME) are expanded using the previous defined environment variables in the container and any service environment variables. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Defaults to "". | No |
| valueFrom | [io.k8s.api.core.v1.EnvVarSource](#io.k8s.api.core.v1.envvarsource) | Source for the environment variable's value. Cannot be used if value is not empty. | No |

#### io.k8s.api.core.v1.EnvVarSource

EnvVarSource represents a source for the value of an EnvVar.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| configMapKeyRef | [io.k8s.api.core.v1.ConfigMapKeySelector](#io.k8s.api.core.v1.configmapkeyselector) | Selects a key of a ConfigMap. | No |
| fieldRef | [io.k8s.api.core.v1.ObjectFieldSelector](#io.k8s.api.core.v1.objectfieldselector) | Selects a field of the pod: supports metadata.name, metadata.namespace, metadata.labels, metadata.annotations, spec.nodeName, spec.serviceAccountName, status.hostIP, status.podIP. | No |
| resourceFieldRef | [io.k8s.api.core.v1.ResourceFieldSelector](#io.k8s.api.core.v1.resourcefieldselector) | Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.ephemeral-storage) are currently supported. | No |
| secretKeyRef | [io.k8s.api.core.v1.SecretKeySelector](#io.k8s.api.core.v1.secretkeyselector) | Selects a key of a secret in the pod's namespace | No |

#### io.k8s.api.core.v1.Event

Event is a report of an event somewhere in the cluster.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| action | string | What action was taken/failed regarding to the Regarding object. | No |
| apiVersion | string | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: <https://git.k8s.io/community/contributors/devel/api-conventions.md#resources> | No |
| count | integer | The number of times this event has occurred. | No |
| eventTime | [io.k8s.apimachinery.pkg.apis.meta.v1.MicroTime](#io.k8s.apimachinery.pkg.apis.meta.v1.microtime) | Time when this Event was first observed. | No |
| firstTimestamp | [io.k8s.apimachinery.pkg.apis.meta.v1.Time](#io.k8s.apimachinery.pkg.apis.meta.v1.time) | The time at which the event was first recorded. (Time of server receipt is in TypeMeta.) | No |
| involvedObject | [io.k8s.api.core.v1.ObjectReference](#io.k8s.api.core.v1.objectreference) | The object that this event is about. | Yes |
| kind | string | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: <https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds> | No |
| lastTimestamp | [io.k8s.apimachinery.pkg.apis.meta.v1.Time](#io.k8s.apimachinery.pkg.apis.meta.v1.time) | The time at which the most recent occurrence of this event was recorded. | No |
| message | string | A human-readable description of the status of this operation. | No |
| metadata | [io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta](#io.k8s.apimachinery.pkg.apis.meta.v1.objectmeta) | Standard object's metadata. More info: <https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata> | Yes |
| reason | string | This should be a short, machine understandable string that gives the reason for the transition into the object's current status. | No |
| related | [io.k8s.api.core.v1.ObjectReference](#io.k8s.api.core.v1.objectreference) | Optional secondary object for more complex actions. | No |
| reportingComponent | string | Name of the controller that emitted this Event, e.g. `kubernetes.io/kubelet`. | No |
| reportingInstance | string | ID of the controller instance, e.g. `kubelet-xyzf`. | No |
| series | [io.k8s.api.core.v1.EventSeries](#io.k8s.api.core.v1.eventseries) | Data about the Event series this event represents or nil if it's a singleton Event. | No |
| source | [io.k8s.api.core.v1.EventSource](#io.k8s.api.core.v1.eventsource) | The component reporting this event. Should be a short machine understandable string. | No |
| type | string | Type of this event (Normal, Warning), new types could be added in the future | No |

#### io.k8s.api.core.v1.EventSeries

EventSeries contain information on series of events, i.e. thing that was/is happening continuously for some time.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| count | integer | Number of occurrences in this series up to the last heartbeat time | No |
| lastObservedTime | [io.k8s.apimachinery.pkg.apis.meta.v1.MicroTime](#io.k8s.apimachinery.pkg.apis.meta.v1.microtime) | Time of the last occurrence observed | No |
| state | string | State of this Series: Ongoing or Finished Deprecated. Planned removal for 1.18 | No |

#### io.k8s.api.core.v1.EventSource

EventSource contains information for an event.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| component | string | Component from which the event is generated. | No |
| host | string | Node name on which the event is generated. | No |

#### io.k8s.api.core.v1.ExecAction

ExecAction describes a "run in container" action.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| command | [ string ] | Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions ('\|', etc) won't work. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy. | No |

#### io.k8s.api.core.v1.FCVolumeSource

Represents a Fibre Channel volume. Fibre Channel volumes can only be mounted as read/write once. Fibre Channel volumes support ownership management and SELinux relabeling.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| fsType | string | Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. | No |
| lun | integer | Optional: FC target lun number | No |
| readOnly | boolean | Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. | No |
| targetWWNs | [ string ] | Optional: FC target worldwide names (WWNs) | No |
| wwids | [ string ] | Optional: FC volume world wide identifiers (wwids) Either wwids or combination of targetWWNs and lun must be set, but not both simultaneously. | No |

#### io.k8s.api.core.v1.FlexVolumeSource

FlexVolume represents a generic volume resource that is provisioned/attached using an exec based plugin.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| driver | string | Driver is the name of the driver to use for this volume. | Yes |
| fsType | string | Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". The default filesystem depends on FlexVolume script. | No |
| options | object | Optional: Extra command options if any. | No |
| readOnly | boolean | Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. | No |
| secretRef | [io.k8s.api.core.v1.LocalObjectReference](#io.k8s.api.core.v1.localobjectreference) | Optional: SecretRef is reference to the secret object containing sensitive information to pass to the plugin scripts. This may be empty if no secret object is specified. If the secret object contains more than one secret, all secrets are passed to the plugin scripts. | No |

#### io.k8s.api.core.v1.FlockerVolumeSource

Represents a Flocker volume mounted by the Flocker agent. One and only one of datasetName and datasetUUID should be set. Flocker volumes do not support ownership management or SELinux relabeling.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| datasetName | string | Name of the dataset stored as metadata -> name on the dataset for Flocker should be considered as deprecated | No |
| datasetUUID | string | UUID of the dataset. This is unique identifier of a Flocker dataset | No |

#### io.k8s.api.core.v1.GCEPersistentDiskVolumeSource

Represents a Persistent Disk resource in Google Compute Engine.

A GCE PD must exist before mounting to a container. The disk must also be in the same GCE project and zone as the kubelet. A GCE PD can only be mounted as read/write once or read-only many times. GCE PDs support ownership management and SELinux relabeling.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| fsType | string | Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: <https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk> | No |
| partition | integer | The partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as "1". Similarly, the volume partition for /dev/sda is "0" (or you can leave the property empty). More info: <https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk> | No |
| pdName | string | Unique name of the PD resource in GCE. Used to identify the disk in GCE. More info: <https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk> | Yes |
| readOnly | boolean | ReadOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. More info: <https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk> | No |

#### io.k8s.api.core.v1.GitRepoVolumeSource

Represents a volume that is populated with the contents of a git repository. Git repo volumes do not support ownership management. Git repo volumes support SELinux relabeling.

DEPRECATED: GitRepo is deprecated. To provision a container with a git repo, mount an EmptyDir into an InitContainer that clones the repo using git, then mount the EmptyDir into the Pod's container.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| directory | string | Target directory name. Must not contain or start with '..'.  If '.' is supplied, the volume directory will be the git repository.  Otherwise, if specified, the volume will contain the git repository in the subdirectory with the given name. | No |
| repository | string | Repository URL | Yes |
| revision | string | Commit hash for the specified revision. | No |

#### io.k8s.api.core.v1.GlusterfsVolumeSource

Represents a Glusterfs mount that lasts the lifetime of a pod. Glusterfs volumes do not support ownership management or SELinux relabeling.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| endpoints | string | EndpointsName is the endpoint name that details Glusterfs topology. More info: <https://releases.k8s.io/HEAD/examples/volumes/glusterfs/README.md#create-a-pod> | Yes |
| path | string | Path is the Glusterfs volume path. More info: <https://releases.k8s.io/HEAD/examples/volumes/glusterfs/README.md#create-a-pod> | Yes |
| readOnly | boolean | ReadOnly here will force the Glusterfs volume to be mounted with read-only permissions. Defaults to false. More info: <https://releases.k8s.io/HEAD/examples/volumes/glusterfs/README.md#create-a-pod> | No |

#### io.k8s.api.core.v1.HTTPGetAction

HTTPGetAction describes an action based on HTTP Get requests.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| host | string | Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead. | No |
| httpHeaders | [ [io.k8s.api.core.v1.HTTPHeader](#io.k8s.api.core.v1.httpheader) ] | Custom headers to set in the request. HTTP allows repeated headers. | No |
| path | string | Path to access on the HTTP server. | No |
| port | [io.k8s.apimachinery.pkg.util.intstr.IntOrString](#io.k8s.apimachinery.pkg.util.intstr.intorstring) | Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME. | Yes |
| scheme | string | Scheme to use for connecting to the host. Defaults to HTTP. | No |

#### io.k8s.api.core.v1.HTTPHeader

HTTPHeader describes a custom header to be used in HTTP probes

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| name | string | The header field name | Yes |
| value | string | The header field value | Yes |

#### io.k8s.api.core.v1.Handler

Handler defines a specific action that should be taken

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| exec | [io.k8s.api.core.v1.ExecAction](#io.k8s.api.core.v1.execaction) | One and only one of the following should be specified. Exec specifies the action to take. | No |
| httpGet | [io.k8s.api.core.v1.HTTPGetAction](#io.k8s.api.core.v1.httpgetaction) | HTTPGet specifies the http request to perform. | No |
| tcpSocket | [io.k8s.api.core.v1.TCPSocketAction](#io.k8s.api.core.v1.tcpsocketaction) | TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported | No |

#### io.k8s.api.core.v1.HostAlias

HostAlias holds the mapping between IP and hostnames that will be injected as an entry in the pod's hosts file.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| hostnames | [ string ] | Hostnames for the above IP address. | No |
| ip | string | IP address of the host file entry. | No |

#### io.k8s.api.core.v1.HostPathVolumeSource

Represents a host path mapped into a pod. Host path volumes do not support ownership management or SELinux relabeling.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| path | string | Path of the directory on the host. If the path is a symlink, it will follow the link to the real path. More info: <https://kubernetes.io/docs/concepts/storage/volumes#hostpath> | Yes |
| type | string | Type for HostPath Volume Defaults to "" More info: <https://kubernetes.io/docs/concepts/storage/volumes#hostpath> | No |

#### io.k8s.api.core.v1.ISCSIVolumeSource

Represents an ISCSI disk. ISCSI volumes can only be mounted as read/write once. ISCSI volumes support ownership management and SELinux relabeling.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| chapAuthDiscovery | boolean | whether support iSCSI Discovery CHAP authentication | No |
| chapAuthSession | boolean | whether support iSCSI Session CHAP authentication | No |
| fsType | string | Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: <https://kubernetes.io/docs/concepts/storage/volumes#iscsi> | No |
| initiatorName | string | Custom iSCSI Initiator Name. If initiatorName is specified with iscsiInterface simultaneously, new iSCSI interface <target portal>:<volume name> will be created for the connection. | No |
| iqn | string | Target iSCSI Qualified Name. | Yes |
| iscsiInterface | string | iSCSI Interface Name that uses an iSCSI transport. Defaults to 'default' (tcp). | No |
| lun | integer | iSCSI Target Lun number. | Yes |
| portals | [ string ] | iSCSI Target Portal List. The portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260). | No |
| readOnly | boolean | ReadOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. | No |
| secretRef | [io.k8s.api.core.v1.LocalObjectReference](#io.k8s.api.core.v1.localobjectreference) | CHAP Secret for iSCSI target and initiator authentication | No |
| targetPortal | string | iSCSI Target Portal. The Portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260). | Yes |

#### io.k8s.api.core.v1.KeyToPath

Maps a string key to a path within a volume.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| key | string | The key to project. | Yes |
| mode | integer | Optional: mode bits to use on this file, must be a value between 0 and 0777. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set. | No |
| path | string | The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'. | Yes |

#### io.k8s.api.core.v1.Lifecycle

Lifecycle describes actions that the management system should take in response to container lifecycle events. For the PostStart and PreStop lifecycle handlers, management of the container blocks until the action is complete, unless the container process fails, in which case the handler is aborted.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| postStart | [io.k8s.api.core.v1.Handler](#io.k8s.api.core.v1.handler) | PostStart is called immediately after a container is created. If the handler fails, the container is terminated and restarted according to its restart policy. Other management of the container blocks until the hook completes. More info: <https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks> | No |
| preStop | [io.k8s.api.core.v1.Handler](#io.k8s.api.core.v1.handler) | PreStop is called immediately before a container is terminated due to an API request or management event such as liveness probe failure, preemption, resource contention, etc. The handler is not called if the container crashes or exits. The reason for termination is passed to the handler. The Pod's termination grace period countdown begins before the PreStop hooked is executed. Regardless of the outcome of the handler, the container will eventually terminate within the Pod's termination grace period. Other management of the container blocks until the hook completes or until the termination grace period is reached. More info: <https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks> | No |

#### io.k8s.api.core.v1.LocalObjectReference

LocalObjectReference contains enough information to let you locate the referenced object inside the same namespace.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| name | string | Name of the referent. More info: <https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names> | No |

#### io.k8s.api.core.v1.NFSVolumeSource

Represents an NFS mount that lasts the lifetime of a pod. NFS volumes do not support ownership management or SELinux relabeling.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| path | string | Path that is exported by the NFS server. More info: <https://kubernetes.io/docs/concepts/storage/volumes#nfs> | Yes |
| readOnly | boolean | ReadOnly here will force the NFS export to be mounted with read-only permissions. Defaults to false. More info: <https://kubernetes.io/docs/concepts/storage/volumes#nfs> | No |
| server | string | Server is the hostname or IP address of the NFS server. More info: <https://kubernetes.io/docs/concepts/storage/volumes#nfs> | Yes |

#### io.k8s.api.core.v1.NodeAffinity

Node affinity is a group of node affinity scheduling rules.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| preferredDuringSchedulingIgnoredDuringExecution | [ [io.k8s.api.core.v1.PreferredSchedulingTerm](#io.k8s.api.core.v1.preferredschedulingterm) ] | The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node matches the corresponding matchExpressions; the node(s) with the highest sum are the most preferred. | No |
| requiredDuringSchedulingIgnoredDuringExecution | [io.k8s.api.core.v1.NodeSelector](#io.k8s.api.core.v1.nodeselector) | If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node. | No |

#### io.k8s.api.core.v1.NodeSelector

A node selector represents the union of the results of one or more label queries over a set of nodes; that is, it represents the OR of the selectors represented by the node selector terms.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| nodeSelectorTerms | [ [io.k8s.api.core.v1.NodeSelectorTerm](#io.k8s.api.core.v1.nodeselectorterm) ] | Required. A list of node selector terms. The terms are ORed. | Yes |

#### io.k8s.api.core.v1.NodeSelectorRequirement

A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| key | string | The label key that the selector applies to. | Yes |
| operator | string | Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt. | Yes |
| values | [ string ] | An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch. | No |

#### io.k8s.api.core.v1.NodeSelectorTerm

A null or empty node selector term matches no objects. The requirements of them are ANDed. The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| matchExpressions | [ [io.k8s.api.core.v1.NodeSelectorRequirement](#io.k8s.api.core.v1.nodeselectorrequirement) ] | A list of node selector requirements by node's labels. | No |
| matchFields | [ [io.k8s.api.core.v1.NodeSelectorRequirement](#io.k8s.api.core.v1.nodeselectorrequirement) ] | A list of node selector requirements by node's fields. | No |

#### io.k8s.api.core.v1.ObjectFieldSelector

ObjectFieldSelector selects an APIVersioned field of an object.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| apiVersion | string | Version of the schema the FieldPath is written in terms of, defaults to "v1". | No |
| fieldPath | string | Path of the field to select in the specified API version. | Yes |

#### io.k8s.api.core.v1.ObjectReference

ObjectReference contains enough information to let you inspect or modify the referred object.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| apiVersion | string | API version of the referent. | No |
| fieldPath | string | If referring to a piece of an object instead of an entire object, this string should contain a valid JSON/Go field access statement, such as desiredState.manifest.containers[2]. For example, if the object reference is to a container within a pod, this would take on a value like: "spec.containers{name}" (where "name" refers to the name of the container that triggered the event) or if no container name is specified "spec.containers[2]" (container with index 2 in this pod). This syntax is chosen only to have some well-defined way of referencing a part of an object. | No |
| kind | string | Kind of the referent. More info: <https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds> | No |
| name | string | Name of the referent. More info: <https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names> | No |
| namespace | string | Namespace of the referent. More info: <https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/> | No |
| resourceVersion | string | Specific resourceVersion to which this reference is made, if any. More info: <https://git.k8s.io/community/contributors/devel/api-conventions.md#concurrency-control-and-consistency> | No |
| uid | string | UID of the referent. More info: <https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids> | No |

#### io.k8s.api.core.v1.PersistentVolumeClaim

PersistentVolumeClaim is a user's request for and claim to a persistent volume

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| apiVersion | string | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: <https://git.k8s.io/community/contributors/devel/api-conventions.md#resources> | No |
| kind | string | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: <https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds> | No |
| metadata | [io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta](#io.k8s.apimachinery.pkg.apis.meta.v1.objectmeta) | Standard object's metadata. More info: <https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata> | No |
| spec | [io.k8s.api.core.v1.PersistentVolumeClaimSpec](#io.k8s.api.core.v1.persistentvolumeclaimspec) | Spec defines the desired characteristics of a volume requested by a pod author. More info: <https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims> | No |
| status | [io.k8s.api.core.v1.PersistentVolumeClaimStatus](#io.k8s.api.core.v1.persistentvolumeclaimstatus) | Status represents the current information/status of a persistent volume claim. Read-only. More info: <https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims> | No |

#### io.k8s.api.core.v1.PersistentVolumeClaimCondition

PersistentVolumeClaimCondition contails details about state of pvc

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| lastProbeTime | [io.k8s.apimachinery.pkg.apis.meta.v1.Time](#io.k8s.apimachinery.pkg.apis.meta.v1.time) | Last time we probed the condition. | No |
| lastTransitionTime | [io.k8s.apimachinery.pkg.apis.meta.v1.Time](#io.k8s.apimachinery.pkg.apis.meta.v1.time) | Last time the condition transitioned from one status to another. | No |
| message | string | Human-readable message indicating details about last transition. | No |
| reason | string | Unique, this should be a short, machine understandable string that gives the reason for condition's last transition. If it reports "ResizeStarted" that means the underlying persistent volume is being resized. | No |
| status | string |  | Yes |
| type | string |  | Yes |

#### io.k8s.api.core.v1.PersistentVolumeClaimSpec

PersistentVolumeClaimSpec describes the common attributes of storage devices and allows a Source for provider-specific attributes

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| accessModes | [ string ] | AccessModes contains the desired access modes the volume should have. More info: <https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1> | No |
| dataSource | [io.k8s.api.core.v1.TypedLocalObjectReference](#io.k8s.api.core.v1.typedlocalobjectreference) | This field requires the VolumeSnapshotDataSource alpha feature gate to be enabled and currently VolumeSnapshot is the only supported data source. If the provisioner can support VolumeSnapshot data source, it will create a new volume and data will be restored to the volume at the same time. If the provisioner does not support VolumeSnapshot data source, volume will not be created and the failure will be reported as an event. In the future, we plan to support more data source types and the behavior of the provisioner may change. | No |
| resources | [io.k8s.api.core.v1.ResourceRequirements](#io.k8s.api.core.v1.resourcerequirements) | Resources represents the minimum resources the volume should have. More info: <https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources> | No |
| selector | [io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector](#io.k8s.apimachinery.pkg.apis.meta.v1.labelselector) | A label query over volumes to consider for binding. | No |
| storageClassName | string | Name of the StorageClass required by the claim. More info: <https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1> | No |
| volumeMode | string | volumeMode defines what type of volume is required by the claim. Value of Filesystem is implied when not included in claim spec. This is a beta feature. | No |
| volumeName | string | VolumeName is the binding reference to the PersistentVolume backing this claim. | No |

#### io.k8s.api.core.v1.PersistentVolumeClaimStatus

PersistentVolumeClaimStatus is the current status of a persistent volume claim.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| accessModes | [ string ] | AccessModes contains the actual access modes the volume backing the PVC has. More info: <https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1> | No |
| capacity | object | Represents the actual resources of the underlying volume. | No |
| conditions | [ [io.k8s.api.core.v1.PersistentVolumeClaimCondition](#io.k8s.api.core.v1.persistentvolumeclaimcondition) ] | Current Condition of persistent volume claim. If underlying persistent volume is being resized then the Condition will be set to 'ResizeStarted'. | No |
| phase | string | Phase represents the current phase of PersistentVolumeClaim. | No |

#### io.k8s.api.core.v1.PersistentVolumeClaimVolumeSource

PersistentVolumeClaimVolumeSource references the user's PVC in the same namespace. This volume finds the bound PV and mounts that volume for the pod. A PersistentVolumeClaimVolumeSource is, essentially, a wrapper around another type of volume that is owned by someone else (the system).

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| claimName | string | ClaimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume. More info: <https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims> | Yes |
| readOnly | boolean | Will force the ReadOnly setting in VolumeMounts. Default false. | No |

#### io.k8s.api.core.v1.PhotonPersistentDiskVolumeSource

Represents a Photon Controller persistent disk resource.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| fsType | string | Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. | No |
| pdID | string | ID that identifies Photon Controller persistent disk | Yes |

#### io.k8s.api.core.v1.PodAffinity

Pod affinity is a group of inter pod affinity scheduling rules.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| preferredDuringSchedulingIgnoredDuringExecution | [ [io.k8s.api.core.v1.WeightedPodAffinityTerm](#io.k8s.api.core.v1.weightedpodaffinityterm) ] | The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred. | No |
| requiredDuringSchedulingIgnoredDuringExecution | [ [io.k8s.api.core.v1.PodAffinityTerm](#io.k8s.api.core.v1.podaffinityterm) ] | If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied. | No |

#### io.k8s.api.core.v1.PodAffinityTerm

Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| labelSelector | [io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector](#io.k8s.apimachinery.pkg.apis.meta.v1.labelselector) | A label query over a set of resources, in this case pods. | No |
| namespaces | [ string ] | namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace" | No |
| topologyKey | string | This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed. | Yes |

#### io.k8s.api.core.v1.PodAntiAffinity

Pod anti affinity is a group of inter pod anti affinity scheduling rules.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| preferredDuringSchedulingIgnoredDuringExecution | [ [io.k8s.api.core.v1.WeightedPodAffinityTerm](#io.k8s.api.core.v1.weightedpodaffinityterm) ] | The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling anti-affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred. | No |
| requiredDuringSchedulingIgnoredDuringExecution | [ [io.k8s.api.core.v1.PodAffinityTerm](#io.k8s.api.core.v1.podaffinityterm) ] | If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the anti-affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied. | No |

#### io.k8s.api.core.v1.PodDNSConfig

PodDNSConfig defines the DNS parameters of a pod in addition to those generated from DNSPolicy.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| nameservers | [ string ] | A list of DNS name server IP addresses. This will be appended to the base nameservers generated from DNSPolicy. Duplicated nameservers will be removed. | No |
| options | [ [io.k8s.api.core.v1.PodDNSConfigOption](#io.k8s.api.core.v1.poddnsconfigoption) ] | A list of DNS resolver options. This will be merged with the base options generated from DNSPolicy. Duplicated entries will be removed. Resolution options given in Options will override those that appear in the base DNSPolicy. | No |
| searches | [ string ] | A list of DNS search domains for host-name lookup. This will be appended to the base search paths generated from DNSPolicy. Duplicated search paths will be removed. | No |

#### io.k8s.api.core.v1.PodDNSConfigOption

PodDNSConfigOption defines DNS resolver options of a pod.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| name | string | Required. | No |
| value | string |  | No |

#### io.k8s.api.core.v1.PodSecurityContext

PodSecurityContext holds pod-level security attributes and common container settings. Some fields are also present in container.securityContext.  Field values of container.securityContext take precedence over field values of PodSecurityContext.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| fsGroup | long | A special supplemental group that applies to all containers in a pod. Some volume types allow the Kubelet to change the ownership of that volume to be owned by the pod:  1. The owning GID will be the FSGroup 2. The setgid bit is set (new files created in the volume will be owned by FSGroup) 3. The permission bits are OR'd with rw-rw----  If unset, the Kubelet will not modify the ownership and permissions of any volume. | No |
| runAsGroup | long | The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in SecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container. | No |
| runAsNonRoot | boolean | Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does. If unset or false, no such validation will be performed. May also be set in SecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. | No |
| runAsUser | long | The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in SecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container. | No |
| seLinuxOptions | [io.k8s.api.core.v1.SELinuxOptions](#io.k8s.api.core.v1.selinuxoptions) | The SELinux context to be applied to all containers. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in SecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container. | No |
| supplementalGroups | [ long ] | A list of groups applied to the first process run in each container, in addition to the container's primary GID.  If unspecified, no groups will be added to any container. | No |
| sysctls | [ [io.k8s.api.core.v1.Sysctl](#io.k8s.api.core.v1.sysctl) ] | Sysctls hold a list of namespaced sysctls used for the pod. Pods with unsupported sysctls (by the container runtime) might fail to launch. | No |
| windowsOptions | [io.k8s.api.core.v1.WindowsSecurityContextOptions](#io.k8s.api.core.v1.windowssecuritycontextoptions) | Windows security options. | No |

#### io.k8s.api.core.v1.PortworxVolumeSource

PortworxVolumeSource represents a Portworx volume resource.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| fsType | string | FSType represents the filesystem type to mount Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs". Implicitly inferred to be "ext4" if unspecified. | No |
| readOnly | boolean | Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. | No |
| volumeID | string | VolumeID uniquely identifies a Portworx volume | Yes |

#### io.k8s.api.core.v1.PreferredSchedulingTerm

An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| preference | [io.k8s.api.core.v1.NodeSelectorTerm](#io.k8s.api.core.v1.nodeselectorterm) | A node selector term, associated with the corresponding weight. | Yes |
| weight | integer | Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100. | Yes |

#### io.k8s.api.core.v1.Probe

Probe describes a health check to be performed against a container to determine whether it is alive or ready to receive traffic.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| exec | [io.k8s.api.core.v1.ExecAction](#io.k8s.api.core.v1.execaction) | One and only one of the following should be specified. Exec specifies the action to take. | No |
| failureThreshold | integer | Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1. | No |
| httpGet | [io.k8s.api.core.v1.HTTPGetAction](#io.k8s.api.core.v1.httpgetaction) | HTTPGet specifies the http request to perform. | No |
| initialDelaySeconds | integer | Number of seconds after the container has started before liveness probes are initiated. More info: <https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes> | No |
| periodSeconds | integer | How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1. | No |
| successThreshold | integer | Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness. Minimum value is 1. | No |
| tcpSocket | [io.k8s.api.core.v1.TCPSocketAction](#io.k8s.api.core.v1.tcpsocketaction) | TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported | No |
| timeoutSeconds | integer | Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: <https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes> | No |

#### io.k8s.api.core.v1.ProjectedVolumeSource

Represents a projected volume source

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| defaultMode | integer | Mode bits to use on created files by default. Must be a value between 0 and 0777. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set. | No |
| sources | [ [io.k8s.api.core.v1.VolumeProjection](#io.k8s.api.core.v1.volumeprojection) ] | list of volume projections | Yes |

#### io.k8s.api.core.v1.QuobyteVolumeSource

Represents a Quobyte mount that lasts the lifetime of a pod. Quobyte volumes do not support ownership management or SELinux relabeling.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| group | string | Group to map volume access to Default is no group | No |
| readOnly | boolean | ReadOnly here will force the Quobyte volume to be mounted with read-only permissions. Defaults to false. | No |
| registry | string | Registry represents a single or multiple Quobyte Registry services specified as a string as host:port pair (multiple entries are separated with commas) which acts as the central registry for volumes | Yes |
| tenant | string | Tenant owning the given Quobyte volume in the Backend Used with dynamically provisioned Quobyte volumes, value is set by the plugin | No |
| user | string | User to map volume access to Defaults to serivceaccount user | No |
| volume | string | Volume is a string that references an already created Quobyte volume by name. | Yes |

#### io.k8s.api.core.v1.RBDVolumeSource

Represents a Rados Block Device mount that lasts the lifetime of a pod. RBD volumes support ownership management and SELinux relabeling.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| fsType | string | Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: <https://kubernetes.io/docs/concepts/storage/volumes#rbd> | No |
| image | string | The rados image name. More info: <https://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it> | Yes |
| keyring | string | Keyring is the path to key ring for RBDUser. Default is /etc/ceph/keyring. More info: <https://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it> | No |
| monitors | [ string ] | A collection of Ceph monitors. More info: <https://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it> | Yes |
| pool | string | The rados pool name. Default is rbd. More info: <https://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it> | No |
| readOnly | boolean | ReadOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. More info: <https://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it> | No |
| secretRef | [io.k8s.api.core.v1.LocalObjectReference](#io.k8s.api.core.v1.localobjectreference) | SecretRef is name of the authentication secret for RBDUser. If provided overrides keyring. Default is nil. More info: <https://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it> | No |
| user | string | The rados user name. Default is admin. More info: <https://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it> | No |

#### io.k8s.api.core.v1.ResourceFieldSelector

ResourceFieldSelector represents container resources (cpu, memory) and their output format

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| containerName | string | Container name: required for volumes, optional for env vars | No |
| divisor | [io.k8s.apimachinery.pkg.api.resource.Quantity](#io.k8s.apimachinery.pkg.api.resource.quantity) | Specifies the output format of the exposed resources, defaults to "1" | No |
| resource | string | Required: resource to select | Yes |

#### io.k8s.api.core.v1.ResourceRequirements

ResourceRequirements describes the compute resource requirements.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| limits | object | Limits describes the maximum amount of compute resources allowed. More info: <https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/> | No |
| requests | object | Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: <https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/> | No |

#### io.k8s.api.core.v1.SELinuxOptions

SELinuxOptions are the labels to be applied to the container

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| level | string | Level is SELinux level label that applies to the container. | No |
| role | string | Role is a SELinux role label that applies to the container. | No |
| type | string | Type is a SELinux type label that applies to the container. | No |
| user | string | User is a SELinux user label that applies to the container. | No |

#### io.k8s.api.core.v1.ScaleIOVolumeSource

ScaleIOVolumeSource represents a persistent ScaleIO volume

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| fsType | string | Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Default is "xfs". | No |
| gateway | string | The host address of the ScaleIO API Gateway. | Yes |
| protectionDomain | string | The name of the ScaleIO Protection Domain for the configured storage. | No |
| readOnly | boolean | Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. | No |
| secretRef | [io.k8s.api.core.v1.LocalObjectReference](#io.k8s.api.core.v1.localobjectreference) | SecretRef references to the secret for ScaleIO user and other sensitive information. If this is not provided, Login operation will fail. | Yes |
| sslEnabled | boolean | Flag to enable/disable SSL communication with Gateway, default false | No |
| storageMode | string | Indicates whether the storage for a volume should be ThickProvisioned or ThinProvisioned. Default is ThinProvisioned. | No |
| storagePool | string | The ScaleIO Storage Pool associated with the protection domain. | No |
| system | string | The name of the storage system as configured in ScaleIO. | Yes |
| volumeName | string | The name of a volume already created in the ScaleIO system that is associated with this volume source. | No |

#### io.k8s.api.core.v1.SecretEnvSource

SecretEnvSource selects a Secret to populate the environment variables with.

The contents of the target Secret's Data field will represent the key-value pairs as environment variables.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| name | string | Name of the referent. More info: <https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names> | No |
| optional | boolean | Specify whether the Secret must be defined | No |

#### io.k8s.api.core.v1.SecretKeySelector

SecretKeySelector selects a key of a Secret.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| key | string | The key of the secret to select from.  Must be a valid secret key. | Yes |
| name | string | Name of the referent. More info: <https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names> | No |
| optional | boolean | Specify whether the Secret or its key must be defined | No |

#### io.k8s.api.core.v1.SecretProjection

Adapts a secret into a projected volume.

The contents of the target Secret's Data field will be presented in a projected volume as files using the keys in the Data field as the file names. Note that this is identical to a secret volume source without the default mode.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| items | [ [io.k8s.api.core.v1.KeyToPath](#io.k8s.api.core.v1.keytopath) ] | If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'. | No |
| name | string | Name of the referent. More info: <https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names> | No |
| optional | boolean | Specify whether the Secret or its key must be defined | No |

#### io.k8s.api.core.v1.SecretVolumeSource

Adapts a Secret into a volume.

The contents of the target Secret's Data field will be presented in a volume as files using the keys in the Data field as the file names. Secret volumes support ownership management and SELinux relabeling.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| defaultMode | integer | Optional: mode bits to use on created files by default. Must be a value between 0 and 0777. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set. | No |
| items | [ [io.k8s.api.core.v1.KeyToPath](#io.k8s.api.core.v1.keytopath) ] | If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'. | No |
| optional | boolean | Specify whether the Secret or its keys must be defined | No |
| secretName | string | Name of the secret in the pod's namespace to use. More info: <https://kubernetes.io/docs/concepts/storage/volumes#secret> | No |

#### io.k8s.api.core.v1.SecurityContext

SecurityContext holds security configuration that will be applied to a container. Some fields are present in both SecurityContext and PodSecurityContext.  When both are set, the values in SecurityContext take precedence.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| allowPrivilegeEscalation | boolean | AllowPrivilegeEscalation controls whether a process can gain more privileges than its parent process. This bool directly controls if the no_new_privs flag will be set on the container process. AllowPrivilegeEscalation is true always when the container is: 1) run as Privileged 2) has CAP_SYS_ADMIN | No |
| capabilities | [io.k8s.api.core.v1.Capabilities](#io.k8s.api.core.v1.capabilities) | The capabilities to add/drop when running containers. Defaults to the default set of capabilities granted by the container runtime. | No |
| privileged | boolean | Run container in privileged mode. Processes in privileged containers are essentially equivalent to root on the host. Defaults to false. | No |
| procMount | string | procMount denotes the type of proc mount to use for the containers. The default is DefaultProcMount which uses the container runtime defaults for readonly paths and masked paths. This requires the ProcMountType feature flag to be enabled. | No |
| readOnlyRootFilesystem | boolean | Whether this container has a read-only root filesystem. Default is false. | No |
| runAsGroup | long | The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. | No |
| runAsNonRoot | boolean | Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does. If unset or false, no such validation will be performed. May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. | No |
| runAsUser | long | The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. | No |
| seLinuxOptions | [io.k8s.api.core.v1.SELinuxOptions](#io.k8s.api.core.v1.selinuxoptions) | The SELinux context to be applied to the container. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. | No |
| windowsOptions | [io.k8s.api.core.v1.WindowsSecurityContextOptions](#io.k8s.api.core.v1.windowssecuritycontextoptions) | Windows security options. | No |

#### io.k8s.api.core.v1.ServiceAccountTokenProjection

ServiceAccountTokenProjection represents a projected service account token volume. This projection can be used to insert a service account token into the pods runtime filesystem for use against APIs (Kubernetes API Server or otherwise).

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| audience | string | Audience is the intended audience of the token. A recipient of a token must identify itself with an identifier specified in the audience of the token, and otherwise should reject the token. The audience defaults to the identifier of the apiserver. | No |
| expirationSeconds | long | ExpirationSeconds is the requested duration of validity of the service account token. As the token approaches expiration, the kubelet volume plugin will proactively rotate the service account token. The kubelet will start trying to rotate the token if the token is older than 80 percent of its time to live or if the token is older than 24 hours.Defaults to 1 hour and must be at least 10 minutes. | No |
| path | string | Path is the path relative to the mount point of the file to project the token into. | Yes |

#### io.k8s.api.core.v1.StorageOSVolumeSource

Represents a StorageOS persistent volume resource.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| fsType | string | Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. | No |
| readOnly | boolean | Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. | No |
| secretRef | [io.k8s.api.core.v1.LocalObjectReference](#io.k8s.api.core.v1.localobjectreference) | SecretRef specifies the secret to use for obtaining the StorageOS API credentials.  If not specified, default values will be attempted. | No |
| volumeName | string | VolumeName is the human-readable name of the StorageOS volume.  Volume names are only unique within a namespace. | No |
| volumeNamespace | string | VolumeNamespace specifies the scope of the volume within StorageOS.  If no namespace is specified then the Pod's namespace will be used.  This allows the Kubernetes name scoping to be mirrored within StorageOS for tighter integration. Set VolumeName to any name to override the default behaviour. Set to "default" if you are not using namespaces within StorageOS. Namespaces that do not pre-exist within StorageOS will be created. | No |

#### io.k8s.api.core.v1.Sysctl

Sysctl defines a kernel parameter to be set

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| name | string | Name of a property to set | Yes |
| value | string | Value of a property to set | Yes |

#### io.k8s.api.core.v1.TCPSocketAction

TCPSocketAction describes an action based on opening a socket

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| host | string | Optional: Host name to connect to, defaults to the pod IP. | No |
| port | [io.k8s.apimachinery.pkg.util.intstr.IntOrString](#io.k8s.apimachinery.pkg.util.intstr.intorstring) | Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME. | Yes |

#### io.k8s.api.core.v1.Toleration

The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| effect | string | Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute. | No |
| key | string | Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys. | No |
| operator | string | Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category. | No |
| tolerationSeconds | long | TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system. | No |
| value | string | Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string. | No |

#### io.k8s.api.core.v1.TypedLocalObjectReference

TypedLocalObjectReference contains enough information to let you locate the typed referenced object inside the same namespace.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| apiGroup | string | APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required. | No |
| kind | string | Kind is the type of resource being referenced | Yes |
| name | string | Name is the name of resource being referenced | Yes |

#### io.k8s.api.core.v1.Volume

Volume represents a named volume in a pod that may be accessed by any container in the pod.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| awsElasticBlockStore | [io.k8s.api.core.v1.AWSElasticBlockStoreVolumeSource](#io.k8s.api.core.v1.awselasticblockstorevolumesource) | AWSElasticBlockStore represents an AWS Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: <https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore> | No |
| azureDisk | [io.k8s.api.core.v1.AzureDiskVolumeSource](#io.k8s.api.core.v1.azurediskvolumesource) | AzureDisk represents an Azure Data Disk mount on the host and bind mount to the pod. | No |
| azureFile | [io.k8s.api.core.v1.AzureFileVolumeSource](#io.k8s.api.core.v1.azurefilevolumesource) | AzureFile represents an Azure File Service mount on the host and bind mount to the pod. | No |
| cephfs | [io.k8s.api.core.v1.CephFSVolumeSource](#io.k8s.api.core.v1.cephfsvolumesource) | CephFS represents a Ceph FS mount on the host that shares a pod's lifetime | No |
| cinder | [io.k8s.api.core.v1.CinderVolumeSource](#io.k8s.api.core.v1.cindervolumesource) | Cinder represents a cinder volume attached and mounted on kubelets host machine More info: <https://releases.k8s.io/HEAD/examples/mysql-cinder-pd/README.md> | No |
| configMap | [io.k8s.api.core.v1.ConfigMapVolumeSource](#io.k8s.api.core.v1.configmapvolumesource) | ConfigMap represents a configMap that should populate this volume | No |
| csi | [io.k8s.api.core.v1.CSIVolumeSource](#io.k8s.api.core.v1.csivolumesource) | CSI (Container Storage Interface) represents storage that is handled by an external CSI driver (Alpha feature). | No |
| downwardAPI | [io.k8s.api.core.v1.DownwardAPIVolumeSource](#io.k8s.api.core.v1.downwardapivolumesource) | DownwardAPI represents downward API about the pod that should populate this volume | No |
| emptyDir | [io.k8s.api.core.v1.EmptyDirVolumeSource](#io.k8s.api.core.v1.emptydirvolumesource) | EmptyDir represents a temporary directory that shares a pod's lifetime. More info: <https://kubernetes.io/docs/concepts/storage/volumes#emptydir> | No |
| fc | [io.k8s.api.core.v1.FCVolumeSource](#io.k8s.api.core.v1.fcvolumesource) | FC represents a Fibre Channel resource that is attached to a kubelet's host machine and then exposed to the pod. | No |
| flexVolume | [io.k8s.api.core.v1.FlexVolumeSource](#io.k8s.api.core.v1.flexvolumesource) | FlexVolume represents a generic volume resource that is provisioned/attached using an exec based plugin. | No |
| flocker | [io.k8s.api.core.v1.FlockerVolumeSource](#io.k8s.api.core.v1.flockervolumesource) | Flocker represents a Flocker volume attached to a kubelet's host machine. This depends on the Flocker control service being running | No |
| gcePersistentDisk | [io.k8s.api.core.v1.GCEPersistentDiskVolumeSource](#io.k8s.api.core.v1.gcepersistentdiskvolumesource) | GCEPersistentDisk represents a GCE Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: <https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk> | No |
| gitRepo | [io.k8s.api.core.v1.GitRepoVolumeSource](#io.k8s.api.core.v1.gitrepovolumesource) | GitRepo represents a git repository at a particular revision. DEPRECATED: GitRepo is deprecated. To provision a container with a git repo, mount an EmptyDir into an InitContainer that clones the repo using git, then mount the EmptyDir into the Pod's container. | No |
| glusterfs | [io.k8s.api.core.v1.GlusterfsVolumeSource](#io.k8s.api.core.v1.glusterfsvolumesource) | Glusterfs represents a Glusterfs mount on the host that shares a pod's lifetime. More info: <https://releases.k8s.io/HEAD/examples/volumes/glusterfs/README.md> | No |
| hostPath | [io.k8s.api.core.v1.HostPathVolumeSource](#io.k8s.api.core.v1.hostpathvolumesource) | HostPath represents a pre-existing file or directory on the host machine that is directly exposed to the container. This is generally used for system agents or other privileged things that are allowed to see the host machine. Most containers will NOT need this. More info: <https://kubernetes.io/docs/concepts/storage/volumes#hostpath> | No |
| iscsi | [io.k8s.api.core.v1.ISCSIVolumeSource](#io.k8s.api.core.v1.iscsivolumesource) | ISCSI represents an ISCSI Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: <https://releases.k8s.io/HEAD/examples/volumes/iscsi/README.md> | No |
| name | string | Volume's name. Must be a DNS_LABEL and unique within the pod. More info: <https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names> | Yes |
| nfs | [io.k8s.api.core.v1.NFSVolumeSource](#io.k8s.api.core.v1.nfsvolumesource) | NFS represents an NFS mount on the host that shares a pod's lifetime More info: <https://kubernetes.io/docs/concepts/storage/volumes#nfs> | No |
| persistentVolumeClaim | [io.k8s.api.core.v1.PersistentVolumeClaimVolumeSource](#io.k8s.api.core.v1.persistentvolumeclaimvolumesource) | PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace. More info: <https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims> | No |
| photonPersistentDisk | [io.k8s.api.core.v1.PhotonPersistentDiskVolumeSource](#io.k8s.api.core.v1.photonpersistentdiskvolumesource) | PhotonPersistentDisk represents a PhotonController persistent disk attached and mounted on kubelets host machine | No |
| portworxVolume | [io.k8s.api.core.v1.PortworxVolumeSource](#io.k8s.api.core.v1.portworxvolumesource) | PortworxVolume represents a portworx volume attached and mounted on kubelets host machine | No |
| projected | [io.k8s.api.core.v1.ProjectedVolumeSource](#io.k8s.api.core.v1.projectedvolumesource) | Items for all in one resources secrets, configmaps, and downward API | No |
| quobyte | [io.k8s.api.core.v1.QuobyteVolumeSource](#io.k8s.api.core.v1.quobytevolumesource) | Quobyte represents a Quobyte mount on the host that shares a pod's lifetime | No |
| rbd | [io.k8s.api.core.v1.RBDVolumeSource](#io.k8s.api.core.v1.rbdvolumesource) | RBD represents a Rados Block Device mount on the host that shares a pod's lifetime. More info: <https://releases.k8s.io/HEAD/examples/volumes/rbd/README.md> | No |
| scaleIO | [io.k8s.api.core.v1.ScaleIOVolumeSource](#io.k8s.api.core.v1.scaleiovolumesource) | ScaleIO represents a ScaleIO persistent volume attached and mounted on Kubernetes nodes. | No |
| secret | [io.k8s.api.core.v1.SecretVolumeSource](#io.k8s.api.core.v1.secretvolumesource) | Secret represents a secret that should populate this volume. More info: <https://kubernetes.io/docs/concepts/storage/volumes#secret> | No |
| storageos | [io.k8s.api.core.v1.StorageOSVolumeSource](#io.k8s.api.core.v1.storageosvolumesource) | StorageOS represents a StorageOS volume attached and mounted on Kubernetes nodes. | No |
| vsphereVolume | [io.k8s.api.core.v1.VsphereVirtualDiskVolumeSource](#io.k8s.api.core.v1.vspherevirtualdiskvolumesource) | VsphereVolume represents a vSphere volume attached and mounted on kubelets host machine | No |

#### io.k8s.api.core.v1.VolumeDevice

volumeDevice describes a mapping of a raw block device within a container.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| devicePath | string | devicePath is the path inside of the container that the device will be mapped to. | Yes |
| name | string | name must match the name of a persistentVolumeClaim in the pod | Yes |

#### io.k8s.api.core.v1.VolumeMount

VolumeMount describes a mounting of a Volume within a container.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| mountPath | string | Path within the container at which the volume should be mounted.  Must not contain ':'. | Yes |
| mountPropagation | string | mountPropagation determines how mounts are propagated from the host to container and the other way around. When not set, MountPropagationNone is used. This field is beta in 1.10. | No |
| name | string | This must match the Name of a Volume. | Yes |
| readOnly | boolean | Mounted read-only if true, read-write otherwise (false or unspecified). Defaults to false. | No |
| subPath | string | Path within the volume from which the container's volume should be mounted. Defaults to "" (volume's root). | No |
| subPathExpr | string | Expanded path within the volume from which the container's volume should be mounted. Behaves similarly to SubPath but environment variable references $(VAR_NAME) are expanded using the container's environment. Defaults to "" (volume's root). SubPathExpr and SubPath are mutually exclusive. This field is beta in 1.15. | No |

#### io.k8s.api.core.v1.VolumeProjection

Projection that may be projected along with other supported volume types

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| configMap | [io.k8s.api.core.v1.ConfigMapProjection](#io.k8s.api.core.v1.configmapprojection) | information about the configMap data to project | No |
| downwardAPI | [io.k8s.api.core.v1.DownwardAPIProjection](#io.k8s.api.core.v1.downwardapiprojection) | information about the downwardAPI data to project | No |
| secret | [io.k8s.api.core.v1.SecretProjection](#io.k8s.api.core.v1.secretprojection) | information about the secret data to project | No |
| serviceAccountToken | [io.k8s.api.core.v1.ServiceAccountTokenProjection](#io.k8s.api.core.v1.serviceaccounttokenprojection) | information about the serviceAccountToken data to project | No |

#### io.k8s.api.core.v1.VsphereVirtualDiskVolumeSource

Represents a vSphere volume resource.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| fsType | string | Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. | No |
| storagePolicyID | string | Storage Policy Based Management (SPBM) profile ID associated with the StoragePolicyName. | No |
| storagePolicyName | string | Storage Policy Based Management (SPBM) profile name. | No |
| volumePath | string | Path that identifies vSphere volume vmdk | Yes |

#### io.k8s.api.core.v1.WeightedPodAffinityTerm

The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| podAffinityTerm | [io.k8s.api.core.v1.PodAffinityTerm](#io.k8s.api.core.v1.podaffinityterm) | Required. A pod affinity term, associated with the corresponding weight. | Yes |
| weight | integer | weight associated with matching the corresponding podAffinityTerm, in the range 1-100. | Yes |

#### io.k8s.api.core.v1.WindowsSecurityContextOptions

WindowsSecurityContextOptions contain Windows-specific options and credentials.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| gmsaCredentialSpec | string | GMSACredentialSpec is where the GMSA admission webhook (<https://github.com/kubernetes-sigs/windows-gmsa>) inlines the contents of the GMSA credential spec named by the GMSACredentialSpecName field. This field is alpha-level and is only honored by servers that enable the WindowsGMSA feature flag. | No |
| gmsaCredentialSpecName | string | GMSACredentialSpecName is the name of the GMSA credential spec to use. This field is alpha-level and is only honored by servers that enable the WindowsGMSA feature flag. | No |

#### io.k8s.api.policy.v1beta1.PodDisruptionBudgetSpec

PodDisruptionBudgetSpec is a description of a PodDisruptionBudget.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| maxUnavailable | [io.k8s.apimachinery.pkg.util.intstr.IntOrString](#io.k8s.apimachinery.pkg.util.intstr.intorstring) | An eviction is allowed if at most "maxUnavailable" pods selected by "selector" are unavailable after the eviction, i.e. even in absence of the evicted pod. For example, one can prevent all voluntary evictions by specifying 0. This is a mutually exclusive setting with "minAvailable". | No |
| minAvailable | [io.k8s.apimachinery.pkg.util.intstr.IntOrString](#io.k8s.apimachinery.pkg.util.intstr.intorstring) | An eviction is allowed if at least "minAvailable" pods selected by "selector" will still be available after the eviction, i.e. even in the absence of the evicted pod.  So for example you can prevent all voluntary evictions by specifying "100%". | No |
| selector | [io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector](#io.k8s.apimachinery.pkg.apis.meta.v1.labelselector) | Label query over pods whose evictions are managed by the disruption budget. | No |

#### io.k8s.apimachinery.pkg.api.resource.Quantity

Quantity is a fixed-point representation of a number. It provides convenient marshaling/unmarshaling in JSON and YAML, in addition to String() and Int64() accessors.

The serialization format is:

<quantity>        ::= <signedNumber><suffix>
  (Note that <suffix> may be empty, from the "" case in <decimalSI>.)
<digit>           ::= 0 | 1 | ... | 9 <digits>          ::= <digit> | <digit><digits> <number>          ::= <digits> | <digits>.<digits> | <digits>. | .<digits> <sign>            ::= "+" | "-" <signedNumber>    ::= <number> | <sign><number> <suffix>          ::= <binarySI> | <decimalExponent> | <decimalSI> <binarySI>        ::= Ki | Mi | Gi | Ti | Pi | Ei
  (International System of units; See: <http://physics.nist.gov/cuu/Units/binary.html>)
<decimalSI>       ::= m | "" | k | M | G | T | P | E
  (Note that 1024 = 1Ki but 1000 = 1k; I didn't choose the capitalization.)
<decimalExponent> ::= "e" <signedNumber> | "E" <signedNumber>

No matter which of the three exponent forms is used, no quantity may represent a number greater than 2^63-1 in magnitude, nor may it have more than 3 decimal places. Numbers larger or more precise will be capped or rounded up. (E.g.: 0.1m will rounded up to 1m.) This may be extended in the future if we require larger or smaller quantities.

When a Quantity is parsed from a string, it will remember the type of suffix it had, and will use the same type again when it is serialized.

Before serializing, Quantity will be put in "canonical form". This means that Exponent/suffix will be adjusted up or down (with a corresponding increase or decrease in Mantissa) such that:
  a. No precision is lost
  b. No fractional digits will be emitted
  c. The exponent (or suffix) is as large as possible.
The sign will be omitted unless the number is negative.

Examples:
  1.5 will be serialized as "1500m"
  1.5Gi will be serialized as "1536Mi"

Note that the quantity will NEVER be internally represented by a floating point number. That is the whole point of this exercise.

Non-canonical values will still parse as long as they are well formed, but will be re-emitted in their canonical form. (So always use canonical form, or don't diff.)

This format is intended to make it difficult to use these numbers without writing some sort of special handling code in the hopes that that will cause implementors to also use a fixed point implementation.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| io.k8s.apimachinery.pkg.api.resource.Quantity | string | Quantity is a fixed-point representation of a number. It provides convenient marshaling/unmarshaling in JSON and YAML, in addition to String() and Int64() accessors.  The serialization format is:  <quantity>        ::= <signedNumber><suffix>   (Note that <suffix> may be empty, from the "" case in <decimalSI>.) <digit>           ::= 0 | 1 | ... | 9 <digits>          ::= <digit> | <digit><digits> <number>          ::= <digits> | <digits>.<digits> | <digits>. | .<digits> <sign>            ::= "+" | "-" <signedNumber>    ::= <number> | <sign><number> <suffix>          ::= <binarySI> | <decimalExponent> | <decimalSI> <binarySI>        ::= Ki | Mi | Gi | Ti | Pi | Ei   (International System of units; See: http://physics.nist.gov/cuu/Units/binary.html) <decimalSI>       ::= m | "" | k | M | G | T | P | E   (Note that 1024 = 1Ki but 1000 = 1k; I didn't choose the capitalization.) <decimalExponent> ::= "e" <signedNumber> | "E" <signedNumber>  No matter which of the three exponent forms is used, no quantity may represent a number greater than 2^63-1 in magnitude, nor may it have more than 3 decimal places. Numbers larger or more precise will be capped or rounded up. (E.g.: 0.1m will rounded up to 1m.) This may be extended in the future if we require larger or smaller quantities.  When a Quantity is parsed from a string, it will remember the type of suffix it had, and will use the same type again when it is serialized.  Before serializing, Quantity will be put in "canonical form". This means that Exponent/suffix will be adjusted up or down (with a corresponding increase or decrease in Mantissa) such that:   a. No precision is lost   b. No fractional digits will be emitted   c. The exponent (or suffix) is as large as possible. The sign will be omitted unless the number is negative.  Examples:   1.5 will be serialized as "1500m"   1.5Gi will be serialized as "1536Mi"  Note that the quantity will NEVER be internally represented by a floating point number. That is the whole point of this exercise.  Non-canonical values will still parse as long as they are well formed, but will be re-emitted in their canonical form. (So always use canonical form, or don't diff.)  This format is intended to make it difficult to use these numbers without writing some sort of special handling code in the hopes that that will cause implementors to also use a fixed point implementation. |  |

#### io.k8s.apimachinery.pkg.apis.meta.v1.CreateOptions

CreateOptions may be provided when creating an API object.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| dryRun | [ string ] |  | No |
| fieldManager | string |  | No |

#### io.k8s.apimachinery.pkg.apis.meta.v1.Fields

Fields stores a set of fields in a data structure like a Trie. To understand how this is used, see: <https://github.com/kubernetes-sigs/structured-merge-diff>

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| io.k8s.apimachinery.pkg.apis.meta.v1.Fields | object | Fields stores a set of fields in a data structure like a Trie. To understand how this is used, see: <https://github.com/kubernetes-sigs/structured-merge-diff> |  |

#### io.k8s.apimachinery.pkg.apis.meta.v1.Initializer

Initializer is information about an initializer that has not yet completed.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| name | string | name of the process that is responsible for initializing this object. | Yes |

#### io.k8s.apimachinery.pkg.apis.meta.v1.Initializers

Initializers tracks the progress of initialization.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| pending | [ [io.k8s.apimachinery.pkg.apis.meta.v1.Initializer](#io.k8s.apimachinery.pkg.apis.meta.v1.initializer) ] | Pending is a list of initializers that must execute in order before this object is visible. When the last pending initializer is removed, and no failing result is set, the initializers struct will be set to nil and the object is considered as initialized and visible to all clients. | Yes |
| result | [io.k8s.apimachinery.pkg.apis.meta.v1.Status](#io.k8s.apimachinery.pkg.apis.meta.v1.status) | If result is set with the Failure field, the object will be persisted to storage and then deleted, ensuring that other clients can observe the deletion. | No |

#### io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector

A label selector is a label query over a set of resources. The result of matchLabels and matchExpressions are ANDed. An empty label selector matches all objects. A null label selector matches no objects.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| matchExpressions | [ [io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelectorRequirement](#io.k8s.apimachinery.pkg.apis.meta.v1.labelselectorrequirement) ] | matchExpressions is a list of label selector requirements. The requirements are ANDed. | No |
| matchLabels | object | matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed. | No |

#### io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelectorRequirement

A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| key | string | key is the label key that the selector applies to. | Yes |
| operator | string | operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist. | Yes |
| values | [ string ] | values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch. | No |

#### io.k8s.apimachinery.pkg.apis.meta.v1.ListMeta

ListMeta describes metadata that synthetic resources must have, including lists and various status objects. A resource may have only one of {ObjectMeta, ListMeta}.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| continue | string | continue may be set if the user set a limit on the number of items returned, and indicates that the server has more data available. The value is opaque and may be used to issue another request to the endpoint that served this list to retrieve the next set of available objects. Continuing a consistent list may not be possible if the server configuration has changed or more than a few minutes have passed. The resourceVersion field returned when using this continue value will be identical to the value in the first response, unless you have received this token from an error message. | No |
| remainingItemCount | long | remainingItemCount is the number of subsequent items in the list which are not included in this list response. If the list request contained label or field selectors, then the number of remaining items is unknown and the field will be left unset and omitted during serialization. If the list is complete (either because it is not chunking or because this is the last chunk), then there are no more remaining items and this field will be left unset and omitted during serialization. Servers older than v1.15 do not set this field. The intended use of the remainingItemCount is *estimating* the size of a collection. Clients should not rely on the remainingItemCount to be set or to be exact.  This field is alpha and can be changed or removed without notice. | No |
| resourceVersion | string | String that identifies the server's internal version of this object that can be used by clients to determine when objects have changed. Value must be treated as opaque by clients and passed unmodified back to the server. Populated by the system. Read-only. More info: <https://git.k8s.io/community/contributors/devel/api-conventions.md#concurrency-control-and-consistency> | No |
| selfLink | string | selfLink is a URL representing this object. Populated by the system. Read-only. | No |

#### io.k8s.apimachinery.pkg.apis.meta.v1.ManagedFieldsEntry

ManagedFieldsEntry is a workflow-id, a FieldSet and the group version of the resource that the fieldset applies to.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| apiVersion | string | APIVersion defines the version of this resource that this field set applies to. The format is "group/version" just like the top-level APIVersion field. It is necessary to track the version of a field set because it cannot be automatically converted. | No |
| fields | [io.k8s.apimachinery.pkg.apis.meta.v1.Fields](#io.k8s.apimachinery.pkg.apis.meta.v1.fields) | Fields identifies a set of fields. | No |
| manager | string | Manager is an identifier of the workflow managing these fields. | No |
| operation | string | Operation is the type of operation which lead to this ManagedFieldsEntry being created. The only valid values for this field are 'Apply' and 'Update'. | No |
| time | [io.k8s.apimachinery.pkg.apis.meta.v1.Time](#io.k8s.apimachinery.pkg.apis.meta.v1.time) | Time is timestamp of when these fields were set. It should always be empty if Operation is 'Apply' | No |

#### io.k8s.apimachinery.pkg.apis.meta.v1.MicroTime

MicroTime is version of Time with microsecond level precision.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| io.k8s.apimachinery.pkg.apis.meta.v1.MicroTime | string | MicroTime is version of Time with microsecond level precision. |  |

#### io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta

ObjectMeta is metadata that all persisted resources must have, which includes all objects users must create.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| annotations | object | Annotations is an unstructured key value map stored with a resource that may be set by external tools to store and retrieve arbitrary metadata. They are not queryable and should be preserved when modifying objects. More info: <http://kubernetes.io/docs/user-guide/annotations> | No |
| clusterName | string | The name of the cluster which the object belongs to. This is used to distinguish resources with same name and namespace in different clusters. This field is not set anywhere right now and apiserver is going to ignore it if set in create or update request. | No |
| creationTimestamp | [io.k8s.apimachinery.pkg.apis.meta.v1.Time](#io.k8s.apimachinery.pkg.apis.meta.v1.time) | CreationTimestamp is a timestamp representing the server time when this object was created. It is not guaranteed to be set in happens-before order across separate operations. Clients may not set this value. It is represented in RFC3339 form and is in UTC.  Populated by the system. Read-only. Null for lists. More info: <https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata> | No |
| deletionGracePeriodSeconds | long | Number of seconds allowed for this object to gracefully terminate before it will be removed from the system. Only set when deletionTimestamp is also set. May only be shortened. Read-only. | No |
| deletionTimestamp | [io.k8s.apimachinery.pkg.apis.meta.v1.Time](#io.k8s.apimachinery.pkg.apis.meta.v1.time) | DeletionTimestamp is RFC 3339 date and time at which this resource will be deleted. This field is set by the server when a graceful deletion is requested by the user, and is not directly settable by a client. The resource is expected to be deleted (no longer visible from resource lists, and not reachable by name) after the time in this field, once the finalizers list is empty. As long as the finalizers list contains items, deletion is blocked. Once the deletionTimestamp is set, this value may not be unset or be set further into the future, although it may be shortened or the resource may be deleted prior to this time. For example, a user may request that a pod is deleted in 30 seconds. The Kubelet will react by sending a graceful termination signal to the containers in the pod. After that 30 seconds, the Kubelet will send a hard termination signal (SIGKILL) to the container and after cleanup, remove the pod from the API. In the presence of network partitions, this object may still exist after this timestamp, until an administrator or automated process can determine the resource is fully terminated. If not set, graceful deletion of the object has not been requested.  Populated by the system when a graceful deletion is requested. Read-only. More info: <https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata> | No |
| finalizers | [ string ] | Must be empty before the object is deleted from the registry. Each entry is an identifier for the responsible component that will remove the entry from the list. If the deletionTimestamp of the object is non-nil, entries in this list can only be removed. | No |
| generateName | string | GenerateName is an optional prefix, used by the server, to generate a unique name ONLY IF the Name field has not been provided. If this field is used, the name returned to the client will be different than the name passed. This value will also be combined with a unique suffix. The provided value has the same validation rules as the Name field, and may be truncated by the length of the suffix required to make the value unique on the server.  If this field is specified and the generated name exists, the server will NOT return a 409 - instead, it will either return 201 Created or 500 with Reason ServerTimeout indicating a unique name could not be found in the time allotted, and the client should retry (optionally after the time indicated in the Retry-After header).  Applied only if Name is not specified. More info: <https://git.k8s.io/community/contributors/devel/api-conventions.md#idempotency> | No |
| generation | long | A sequence number representing a specific generation of the desired state. Populated by the system. Read-only. | No |
| initializers | [io.k8s.apimachinery.pkg.apis.meta.v1.Initializers](#io.k8s.apimachinery.pkg.apis.meta.v1.initializers) | An initializer is a controller which enforces some system invariant at object creation time. This field is a list of initializers that have not yet acted on this object. If nil or empty, this object has been completely initialized. Otherwise, the object is considered uninitialized and is hidden (in list/watch and get calls) from clients that haven't explicitly asked to observe uninitialized objects.  When an object is created, the system will populate this list with the current set of initializers. Only privileged users may set or modify this list. Once it is empty, it may not be modified further by any user.  DEPRECATED - initializers are an alpha field and will be removed in v1.15. | No |
| labels | object | Map of string keys and values that can be used to organize and categorize (scope and select) objects. May match selectors of replication controllers and services. More info: <http://kubernetes.io/docs/user-guide/labels> | No |
| managedFields | [ [io.k8s.apimachinery.pkg.apis.meta.v1.ManagedFieldsEntry](#io.k8s.apimachinery.pkg.apis.meta.v1.managedfieldsentry) ] | ManagedFields maps workflow-id and version to the set of fields that are managed by that workflow. This is mostly for internal housekeeping, and users typically shouldn't need to set or understand this field. A workflow can be the user's name, a controller's name, or the name of a specific apply path like "ci-cd". The set of fields is always in the version that the workflow used when modifying the object.  This field is alpha and can be changed or removed without notice. | No |
| name | string | Name must be unique within a namespace. Is required when creating resources, although some resources may allow a client to request the generation of an appropriate name automatically. Name is primarily intended for creation idempotence and configuration definition. Cannot be updated. More info: <http://kubernetes.io/docs/user-guide/identifiers#names> | No |
| namespace | string | Namespace defines the space within each name must be unique. An empty namespace is equivalent to the "default" namespace, but "default" is the canonical representation. Not all objects are required to be scoped to a namespace - the value of this field for those objects will be empty.  Must be a DNS_LABEL. Cannot be updated. More info: <http://kubernetes.io/docs/user-guide/namespaces> | No |
| ownerReferences | [ [io.k8s.apimachinery.pkg.apis.meta.v1.OwnerReference](#io.k8s.apimachinery.pkg.apis.meta.v1.ownerreference) ] | List of objects depended by this object. If ALL objects in the list have been deleted, this object will be garbage collected. If this object is managed by a controller, then an entry in this list will point to this controller, with the controller field set to true. There cannot be more than one managing controller. | No |
| resourceVersion | string | An opaque value that represents the internal version of this object that can be used by clients to determine when objects have changed. May be used for optimistic concurrency, change detection, and the watch operation on a resource or set of resources. Clients must treat these values as opaque and passed unmodified back to the server. They may only be valid for a particular resource or set of resources.  Populated by the system. Read-only. Value must be treated as opaque by clients and . More info: <https://git.k8s.io/community/contributors/devel/api-conventions.md#concurrency-control-and-consistency> | No |
| selfLink | string | SelfLink is a URL representing this object. Populated by the system. Read-only. | No |
| uid | string | UID is the unique in time and space value for this object. It is typically generated by the server on successful creation of a resource and is not allowed to change on PUT operations.  Populated by the system. Read-only. More info: <http://kubernetes.io/docs/user-guide/identifiers#uids> | No |

#### io.k8s.apimachinery.pkg.apis.meta.v1.OwnerReference

OwnerReference contains enough information to let you identify an owning object. An owning object must be in the same namespace as the dependent, or be cluster-scoped, so there is no namespace field.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| apiVersion | string | API version of the referent. | Yes |
| blockOwnerDeletion | boolean | If true, AND if the owner has the "foregroundDeletion" finalizer, then the owner cannot be deleted from the key-value store until this reference is removed. Defaults to false. To set this field, a user needs "delete" permission of the owner, otherwise 422 (Unprocessable Entity) will be returned. | No |
| controller | boolean | If true, this reference points to the managing controller. | No |
| kind | string | Kind of the referent. More info: <https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds> | Yes |
| name | string | Name of the referent. More info: <http://kubernetes.io/docs/user-guide/identifiers#names> | Yes |
| uid | string | UID of the referent. More info: <http://kubernetes.io/docs/user-guide/identifiers#uids> | Yes |

#### io.k8s.apimachinery.pkg.apis.meta.v1.Status

Status is a return value for calls that don't return other objects.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| apiVersion | string | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: <https://git.k8s.io/community/contributors/devel/api-conventions.md#resources> | No |
| code | integer | Suggested HTTP return code for this status, 0 if not set. | No |
| details | [io.k8s.apimachinery.pkg.apis.meta.v1.StatusDetails](#io.k8s.apimachinery.pkg.apis.meta.v1.statusdetails) | Extended data associated with the reason.  Each reason may define its own extended details. This field is optional and the data returned is not guaranteed to conform to any schema except that defined by the reason type. | No |
| kind | string | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: <https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds> | No |
| message | string | A human-readable description of the status of this operation. | No |
| metadata | [io.k8s.apimachinery.pkg.apis.meta.v1.ListMeta](#io.k8s.apimachinery.pkg.apis.meta.v1.listmeta) | Standard list metadata. More info: <https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds> | No |
| reason | string | A machine-readable description of why this operation is in the "Failure" status. If this value is empty there is no information available. A Reason clarifies an HTTP status code but does not override it. | No |
| status | string | Status of the operation. One of: "Success" or "Failure". More info: <https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status> | No |

#### io.k8s.apimachinery.pkg.apis.meta.v1.StatusCause

StatusCause provides more information about an api.Status failure, including cases when multiple errors are encountered.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| field | string | The field of the resource that has caused this error, as named by its JSON serialization. May include dot and postfix notation for nested attributes. Arrays are zero-indexed.  Fields may appear more than once in an array of causes due to fields having multiple errors. Optional.  Examples:   "name" - the field "name" on the current resource   "items[0].name" - the field "name" on the first array entry in "items" | No |
| message | string | A human-readable description of the cause of the error.  This field may be presented as-is to a reader. | No |
| reason | string | A machine-readable description of the cause of the error. If this value is empty there is no information available. | No |

#### io.k8s.apimachinery.pkg.apis.meta.v1.StatusDetails

StatusDetails is a set of additional properties that MAY be set by the server to provide additional information about a response. The Reason field of a Status object defines what attributes will be set. Clients must ignore fields that do not match the defined type of each attribute, and should assume that any attribute may be empty, invalid, or under defined.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| causes | [ [io.k8s.apimachinery.pkg.apis.meta.v1.StatusCause](#io.k8s.apimachinery.pkg.apis.meta.v1.statuscause) ] | The Causes array includes more details associated with the StatusReason failure. Not all StatusReasons may provide detailed causes. | No |
| group | string | The group attribute of the resource associated with the status StatusReason. | No |
| kind | string | The kind attribute of the resource associated with the status StatusReason. On some operations may differ from the requested resource Kind. More info: <https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds> | No |
| name | string | The name attribute of the resource associated with the status StatusReason (when there is a single name which can be described). | No |
| retryAfterSeconds | integer | If specified, the time in seconds before the operation should be retried. Some errors may indicate the client must take an alternate action - for those errors this field may indicate how long to wait before taking the alternate action. | No |
| uid | string | UID of the resource. (when there is a single resource which can be described). More info: <http://kubernetes.io/docs/user-guide/identifiers#uids> | No |

#### io.k8s.apimachinery.pkg.apis.meta.v1.Time

Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| io.k8s.apimachinery.pkg.apis.meta.v1.Time | string | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. |  |

#### io.k8s.apimachinery.pkg.util.intstr.IntOrString

IntOrString is a type that can hold an int32 or a string.  When used in JSON or YAML marshalling and unmarshalling, it produces or consumes the inner type.  This allows you to have, for example, a JSON field that can accept a name or number.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| io.k8s.apimachinery.pkg.util.intstr.IntOrString | string | IntOrString is a type that can hold an int32 or a string.  When used in JSON or YAML marshalling and unmarshalling, it produces or consumes the inner type.  This allows you to have, for example, a JSON field that can accept a name or number. |  |
