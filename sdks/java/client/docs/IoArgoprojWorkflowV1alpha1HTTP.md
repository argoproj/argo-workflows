

# IoArgoprojWorkflowV1alpha1HTTP


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**body** | **String** | Body is content of the HTTP Request |  [optional]
**bodyFrom** | [**IoArgoprojWorkflowV1alpha1HTTPBodySource**](IoArgoprojWorkflowV1alpha1HTTPBodySource.md) |  |  [optional]
**headers** | [**List&lt;IoArgoprojWorkflowV1alpha1HTTPHeader&gt;**](IoArgoprojWorkflowV1alpha1HTTPHeader.md) | Headers are an optional list of headers to send with HTTP requests |  [optional]
**insecureSkipVerify** | **Boolean** | InsecureSkipVerify is a bool when if set to true will skip TLS verification for the HTTP client |  [optional]
**method** | **String** | Method is HTTP methods for HTTP Request |  [optional]
**successCondition** | **String** | SuccessCondition is an expression if evaluated to true is considered successful |  [optional]
**timeoutSeconds** | **Integer** | TimeoutSeconds is request timeout for HTTP Request. Default is 30 seconds |  [optional]
**url** | **String** | URL of the HTTP Request | 



