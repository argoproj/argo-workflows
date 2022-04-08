# ArtifactServiceApi

All URIs are relative to *http://localhost:2746*

Method | HTTP request | Description
------------- | ------------- | -------------
[**artifactServiceGetInputArtifact**](ArtifactServiceApi.md#artifactServiceGetInputArtifact) | **GET** /input-artifacts/{namespace}/{name}/{podName}/{artifactName} | Get an input artifact.
[**artifactServiceGetInputArtifactWithHttpInfo**](ArtifactServiceApi.md#artifactServiceGetInputArtifactWithHttpInfo) | **GET** /input-artifacts/{namespace}/{name}/{podName}/{artifactName} | Get an input artifact.
[**artifactServiceGetInputArtifactByUID**](ArtifactServiceApi.md#artifactServiceGetInputArtifactByUID) | **GET** /input-artifacts-by-uid/{uid}/{podName}/{artifactName} | Get an input artifact by UID.
[**artifactServiceGetInputArtifactByUIDWithHttpInfo**](ArtifactServiceApi.md#artifactServiceGetInputArtifactByUIDWithHttpInfo) | **GET** /input-artifacts-by-uid/{uid}/{podName}/{artifactName} | Get an input artifact by UID.
[**artifactServiceGetOutputArtifact**](ArtifactServiceApi.md#artifactServiceGetOutputArtifact) | **GET** /artifacts/{namespace}/{name}/{podName}/{artifactName} | Get an output artifact.
[**artifactServiceGetOutputArtifactWithHttpInfo**](ArtifactServiceApi.md#artifactServiceGetOutputArtifactWithHttpInfo) | **GET** /artifacts/{namespace}/{name}/{podName}/{artifactName} | Get an output artifact.
[**artifactServiceGetOutputArtifactByUID**](ArtifactServiceApi.md#artifactServiceGetOutputArtifactByUID) | **GET** /artifacts-by-uid/{uid}/{podName}/{artifactName} | Get an output artifact by UID.
[**artifactServiceGetOutputArtifactByUIDWithHttpInfo**](ArtifactServiceApi.md#artifactServiceGetOutputArtifactByUIDWithHttpInfo) | **GET** /artifacts-by-uid/{uid}/{podName}/{artifactName} | Get an output artifact by UID.



## artifactServiceGetInputArtifact

> void artifactServiceGetInputArtifact(namespace, name, podName, artifactName)

Get an input artifact.

### Example

```java
// Import classes:
import io.argoproj.workflow.ApiClient;
import io.argoproj.workflow.ApiException;
import io.argoproj.workflow.Configuration;
import io.argoproj.workflow.models.*;
import io.argoproj.workflow.apis.ArtifactServiceApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost:2746");

        ArtifactServiceApi apiInstance = new ArtifactServiceApi(defaultClient);
        String namespace = "namespace_example"; // String | 
        String name = "name_example"; // String | 
        String podName = "podName_example"; // String | 
        String artifactName = "artifactName_example"; // String | 
        try {
            apiInstance.artifactServiceGetInputArtifact(namespace, name, podName, artifactName);
        } catch (ApiException e) {
            System.err.println("Exception when calling ArtifactServiceApi#artifactServiceGetInputArtifact");
            System.err.println("Status code: " + e.getCode());
            System.err.println("Reason: " + e.getResponseBody());
            System.err.println("Response headers: " + e.getResponseHeaders());
            e.printStackTrace();
        }
    }
}
```

### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **String**|  |
 **name** | **String**|  |
 **podName** | **String**|  |
 **artifactName** | **String**|  |

### Return type


null (empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | An artifact file. |  -  |
| **0** | An unexpected error response. |  -  |

## artifactServiceGetInputArtifactWithHttpInfo

> ApiResponse<Void> artifactServiceGetInputArtifact artifactServiceGetInputArtifactWithHttpInfo(namespace, name, podName, artifactName)

Get an input artifact.

### Example

```java
// Import classes:
import io.argoproj.workflow.ApiClient;
import io.argoproj.workflow.ApiException;
import io.argoproj.workflow.ApiResponse;
import io.argoproj.workflow.Configuration;
import io.argoproj.workflow.models.*;
import io.argoproj.workflow.apis.ArtifactServiceApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost:2746");

        ArtifactServiceApi apiInstance = new ArtifactServiceApi(defaultClient);
        String namespace = "namespace_example"; // String | 
        String name = "name_example"; // String | 
        String podName = "podName_example"; // String | 
        String artifactName = "artifactName_example"; // String | 
        try {
            ApiResponse<Void> response = apiInstance.artifactServiceGetInputArtifactWithHttpInfo(namespace, name, podName, artifactName);
            System.out.println("Status code: " + response.getStatusCode());
            System.out.println("Response headers: " + response.getHeaders());
        } catch (ApiException e) {
            System.err.println("Exception when calling ArtifactServiceApi#artifactServiceGetInputArtifact");
            System.err.println("Status code: " + e.getCode());
            System.err.println("Response headers: " + e.getResponseHeaders());
            System.err.println("Reason: " + e.getResponseBody());
            e.printStackTrace();
        }
    }
}
```

### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **String**|  |
 **name** | **String**|  |
 **podName** | **String**|  |
 **artifactName** | **String**|  |

### Return type


ApiResponse<Void>

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | An artifact file. |  -  |
| **0** | An unexpected error response. |  -  |


## artifactServiceGetInputArtifactByUID

> void artifactServiceGetInputArtifactByUID(namespace, uid, podName, artifactName)

Get an input artifact by UID.

### Example

```java
// Import classes:
import io.argoproj.workflow.ApiClient;
import io.argoproj.workflow.ApiException;
import io.argoproj.workflow.Configuration;
import io.argoproj.workflow.models.*;
import io.argoproj.workflow.apis.ArtifactServiceApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost:2746");

        ArtifactServiceApi apiInstance = new ArtifactServiceApi(defaultClient);
        String namespace = "namespace_example"; // String | 
        String uid = "uid_example"; // String | 
        String podName = "podName_example"; // String | 
        String artifactName = "artifactName_example"; // String | 
        try {
            apiInstance.artifactServiceGetInputArtifactByUID(namespace, uid, podName, artifactName);
        } catch (ApiException e) {
            System.err.println("Exception when calling ArtifactServiceApi#artifactServiceGetInputArtifactByUID");
            System.err.println("Status code: " + e.getCode());
            System.err.println("Reason: " + e.getResponseBody());
            System.err.println("Response headers: " + e.getResponseHeaders());
            e.printStackTrace();
        }
    }
}
```

### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **String**|  |
 **uid** | **String**|  |
 **podName** | **String**|  |
 **artifactName** | **String**|  |

### Return type


null (empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | An artifact file. |  -  |
| **0** | An unexpected error response. |  -  |

## artifactServiceGetInputArtifactByUIDWithHttpInfo

> ApiResponse<Void> artifactServiceGetInputArtifactByUID artifactServiceGetInputArtifactByUIDWithHttpInfo(namespace, uid, podName, artifactName)

Get an input artifact by UID.

### Example

```java
// Import classes:
import io.argoproj.workflow.ApiClient;
import io.argoproj.workflow.ApiException;
import io.argoproj.workflow.ApiResponse;
import io.argoproj.workflow.Configuration;
import io.argoproj.workflow.models.*;
import io.argoproj.workflow.apis.ArtifactServiceApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost:2746");

        ArtifactServiceApi apiInstance = new ArtifactServiceApi(defaultClient);
        String namespace = "namespace_example"; // String | 
        String uid = "uid_example"; // String | 
        String podName = "podName_example"; // String | 
        String artifactName = "artifactName_example"; // String | 
        try {
            ApiResponse<Void> response = apiInstance.artifactServiceGetInputArtifactByUIDWithHttpInfo(namespace, uid, podName, artifactName);
            System.out.println("Status code: " + response.getStatusCode());
            System.out.println("Response headers: " + response.getHeaders());
        } catch (ApiException e) {
            System.err.println("Exception when calling ArtifactServiceApi#artifactServiceGetInputArtifactByUID");
            System.err.println("Status code: " + e.getCode());
            System.err.println("Response headers: " + e.getResponseHeaders());
            System.err.println("Reason: " + e.getResponseBody());
            e.printStackTrace();
        }
    }
}
```

### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **String**|  |
 **uid** | **String**|  |
 **podName** | **String**|  |
 **artifactName** | **String**|  |

### Return type


ApiResponse<Void>

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | An artifact file. |  -  |
| **0** | An unexpected error response. |  -  |


## artifactServiceGetOutputArtifact

> void artifactServiceGetOutputArtifact(namespace, name, podName, artifactName)

Get an output artifact.

### Example

```java
// Import classes:
import io.argoproj.workflow.ApiClient;
import io.argoproj.workflow.ApiException;
import io.argoproj.workflow.Configuration;
import io.argoproj.workflow.models.*;
import io.argoproj.workflow.apis.ArtifactServiceApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost:2746");

        ArtifactServiceApi apiInstance = new ArtifactServiceApi(defaultClient);
        String namespace = "namespace_example"; // String | 
        String name = "name_example"; // String | 
        String podName = "podName_example"; // String | 
        String artifactName = "artifactName_example"; // String | 
        try {
            apiInstance.artifactServiceGetOutputArtifact(namespace, name, podName, artifactName);
        } catch (ApiException e) {
            System.err.println("Exception when calling ArtifactServiceApi#artifactServiceGetOutputArtifact");
            System.err.println("Status code: " + e.getCode());
            System.err.println("Reason: " + e.getResponseBody());
            System.err.println("Response headers: " + e.getResponseHeaders());
            e.printStackTrace();
        }
    }
}
```

### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **String**|  |
 **name** | **String**|  |
 **podName** | **String**|  |
 **artifactName** | **String**|  |

### Return type


null (empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | An artifact file. |  -  |
| **0** | An unexpected error response. |  -  |

## artifactServiceGetOutputArtifactWithHttpInfo

> ApiResponse<Void> artifactServiceGetOutputArtifact artifactServiceGetOutputArtifactWithHttpInfo(namespace, name, podName, artifactName)

Get an output artifact.

### Example

```java
// Import classes:
import io.argoproj.workflow.ApiClient;
import io.argoproj.workflow.ApiException;
import io.argoproj.workflow.ApiResponse;
import io.argoproj.workflow.Configuration;
import io.argoproj.workflow.models.*;
import io.argoproj.workflow.apis.ArtifactServiceApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost:2746");

        ArtifactServiceApi apiInstance = new ArtifactServiceApi(defaultClient);
        String namespace = "namespace_example"; // String | 
        String name = "name_example"; // String | 
        String podName = "podName_example"; // String | 
        String artifactName = "artifactName_example"; // String | 
        try {
            ApiResponse<Void> response = apiInstance.artifactServiceGetOutputArtifactWithHttpInfo(namespace, name, podName, artifactName);
            System.out.println("Status code: " + response.getStatusCode());
            System.out.println("Response headers: " + response.getHeaders());
        } catch (ApiException e) {
            System.err.println("Exception when calling ArtifactServiceApi#artifactServiceGetOutputArtifact");
            System.err.println("Status code: " + e.getCode());
            System.err.println("Response headers: " + e.getResponseHeaders());
            System.err.println("Reason: " + e.getResponseBody());
            e.printStackTrace();
        }
    }
}
```

### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **String**|  |
 **name** | **String**|  |
 **podName** | **String**|  |
 **artifactName** | **String**|  |

### Return type


ApiResponse<Void>

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | An artifact file. |  -  |
| **0** | An unexpected error response. |  -  |


## artifactServiceGetOutputArtifactByUID

> void artifactServiceGetOutputArtifactByUID(uid, podName, artifactName)

Get an output artifact by UID.

### Example

```java
// Import classes:
import io.argoproj.workflow.ApiClient;
import io.argoproj.workflow.ApiException;
import io.argoproj.workflow.Configuration;
import io.argoproj.workflow.models.*;
import io.argoproj.workflow.apis.ArtifactServiceApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost:2746");

        ArtifactServiceApi apiInstance = new ArtifactServiceApi(defaultClient);
        String uid = "uid_example"; // String | 
        String podName = "podName_example"; // String | 
        String artifactName = "artifactName_example"; // String | 
        try {
            apiInstance.artifactServiceGetOutputArtifactByUID(uid, podName, artifactName);
        } catch (ApiException e) {
            System.err.println("Exception when calling ArtifactServiceApi#artifactServiceGetOutputArtifactByUID");
            System.err.println("Status code: " + e.getCode());
            System.err.println("Reason: " + e.getResponseBody());
            System.err.println("Response headers: " + e.getResponseHeaders());
            e.printStackTrace();
        }
    }
}
```

### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **uid** | **String**|  |
 **podName** | **String**|  |
 **artifactName** | **String**|  |

### Return type


null (empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | An artifact file. |  -  |
| **0** | An unexpected error response. |  -  |

## artifactServiceGetOutputArtifactByUIDWithHttpInfo

> ApiResponse<Void> artifactServiceGetOutputArtifactByUID artifactServiceGetOutputArtifactByUIDWithHttpInfo(uid, podName, artifactName)

Get an output artifact by UID.

### Example

```java
// Import classes:
import io.argoproj.workflow.ApiClient;
import io.argoproj.workflow.ApiException;
import io.argoproj.workflow.ApiResponse;
import io.argoproj.workflow.Configuration;
import io.argoproj.workflow.models.*;
import io.argoproj.workflow.apis.ArtifactServiceApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost:2746");

        ArtifactServiceApi apiInstance = new ArtifactServiceApi(defaultClient);
        String uid = "uid_example"; // String | 
        String podName = "podName_example"; // String | 
        String artifactName = "artifactName_example"; // String | 
        try {
            ApiResponse<Void> response = apiInstance.artifactServiceGetOutputArtifactByUIDWithHttpInfo(uid, podName, artifactName);
            System.out.println("Status code: " + response.getStatusCode());
            System.out.println("Response headers: " + response.getHeaders());
        } catch (ApiException e) {
            System.err.println("Exception when calling ArtifactServiceApi#artifactServiceGetOutputArtifactByUID");
            System.err.println("Status code: " + e.getCode());
            System.err.println("Response headers: " + e.getResponseHeaders());
            System.err.println("Reason: " + e.getResponseBody());
            e.printStackTrace();
        }
    }
}
```

### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **uid** | **String**|  |
 **podName** | **String**|  |
 **artifactName** | **String**|  |

### Return type


ApiResponse<Void>

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | An artifact file. |  -  |
| **0** | An unexpected error response. |  -  |

