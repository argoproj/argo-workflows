# ArtifactServiceApi

All URIs are relative to *http://localhost:2746*

Method | HTTP request | Description
------------- | ------------- | -------------
[**artifactServiceGetInputArtifact**](ArtifactServiceApi.md#artifactServiceGetInputArtifact) | **GET** /input-artifacts/{namespace}/{name}/{podName}/{artifactName} | Get an input artifact.
[**artifactServiceGetInputArtifactByManifest**](ArtifactServiceApi.md#artifactServiceGetInputArtifactByManifest) | **POST** /input-artifacts-by-manifest/{podName}/{artifactName} | Get an output artifact by a full workflow manifest.
[**artifactServiceGetInputArtifactByUID**](ArtifactServiceApi.md#artifactServiceGetInputArtifactByUID) | **GET** /input-artifacts-by-uid/{uid}/{podName}/{artifactName} | Get an input artifact by UID.
[**artifactServiceGetOutputArtifact**](ArtifactServiceApi.md#artifactServiceGetOutputArtifact) | **GET** /artifacts/{namespace}/{name}/{podName}/{artifactName} | Get an output artifact.
[**artifactServiceGetOutputArtifactByManifest**](ArtifactServiceApi.md#artifactServiceGetOutputArtifactByManifest) | **POST** /artifacts-by-manifest/{podName}/{artifactName} | Get an output artifact by a full workflow manifest.
[**artifactServiceGetOutputArtifactByUID**](ArtifactServiceApi.md#artifactServiceGetOutputArtifactByUID) | **GET** /artifacts-by-uid/{uid}/{podName}/{artifactName} | Get an output artifact by UID.


<a name="artifactServiceGetInputArtifact"></a>
# **artifactServiceGetInputArtifact**
> artifactServiceGetInputArtifact(namespace, name, podName, artifactName)

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
**200** | An artifact file. |  -  |
**0** | An unexpected error response. |  -  |

<a name="artifactServiceGetInputArtifactByManifest"></a>
# **artifactServiceGetInputArtifactByManifest**
> artifactServiceGetInputArtifactByManifest(podName, artifactName, body)

Get an output artifact by a full workflow manifest.

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
    String podName = "podName_example"; // String | 
    String artifactName = "artifactName_example"; // String | 
    IoArgoprojWorkflowV1alpha1ArtifactByManifestRequest body = new IoArgoprojWorkflowV1alpha1ArtifactByManifestRequest(); // IoArgoprojWorkflowV1alpha1ArtifactByManifestRequest | 
    try {
      apiInstance.artifactServiceGetInputArtifactByManifest(podName, artifactName, body);
    } catch (ApiException e) {
      System.err.println("Exception when calling ArtifactServiceApi#artifactServiceGetInputArtifactByManifest");
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
 **podName** | **String**|  |
 **artifactName** | **String**|  |
 **body** | [**IoArgoprojWorkflowV1alpha1ArtifactByManifestRequest**](IoArgoprojWorkflowV1alpha1ArtifactByManifestRequest.md)|  |

### Return type

null (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | An artifact file. |  -  |
**0** | An unexpected error response. |  -  |

<a name="artifactServiceGetInputArtifactByUID"></a>
# **artifactServiceGetInputArtifactByUID**
> artifactServiceGetInputArtifactByUID(namespace, uid, podName, artifactName)

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
**200** | An artifact file. |  -  |
**0** | An unexpected error response. |  -  |

<a name="artifactServiceGetOutputArtifact"></a>
# **artifactServiceGetOutputArtifact**
> artifactServiceGetOutputArtifact(namespace, name, podName, artifactName)

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
**200** | An artifact file. |  -  |
**0** | An unexpected error response. |  -  |

<a name="artifactServiceGetOutputArtifactByManifest"></a>
# **artifactServiceGetOutputArtifactByManifest**
> artifactServiceGetOutputArtifactByManifest(podName, artifactName, body)

Get an output artifact by a full workflow manifest.

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
    String podName = "podName_example"; // String | 
    String artifactName = "artifactName_example"; // String | 
    IoArgoprojWorkflowV1alpha1ArtifactByManifestRequest body = new IoArgoprojWorkflowV1alpha1ArtifactByManifestRequest(); // IoArgoprojWorkflowV1alpha1ArtifactByManifestRequest | 
    try {
      apiInstance.artifactServiceGetOutputArtifactByManifest(podName, artifactName, body);
    } catch (ApiException e) {
      System.err.println("Exception when calling ArtifactServiceApi#artifactServiceGetOutputArtifactByManifest");
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
 **podName** | **String**|  |
 **artifactName** | **String**|  |
 **body** | [**IoArgoprojWorkflowV1alpha1ArtifactByManifestRequest**](IoArgoprojWorkflowV1alpha1ArtifactByManifestRequest.md)|  |

### Return type

null (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | An artifact file. |  -  |
**0** | An unexpected error response. |  -  |

<a name="artifactServiceGetOutputArtifactByUID"></a>
# **artifactServiceGetOutputArtifactByUID**
> artifactServiceGetOutputArtifactByUID(uid, podName, artifactName)

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
**200** | An artifact file. |  -  |
**0** | An unexpected error response. |  -  |

