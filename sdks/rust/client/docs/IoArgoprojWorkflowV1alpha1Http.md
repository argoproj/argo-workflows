# IoArgoprojWorkflowV1alpha1Http

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**body** | Option<**String**> | Body is content of the HTTP Request | [optional]
**body_from** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1HttpBodySource**](io.argoproj.workflow.v1alpha1.HTTPBodySource.md)> |  | [optional]
**headers** | Option<[**Vec<crate::models::IoArgoprojWorkflowV1alpha1HttpHeader>**](io.argoproj.workflow.v1alpha1.HTTPHeader.md)> | Headers are an optional list of headers to send with HTTP requests | [optional]
**insecure_skip_verify** | Option<**bool**> | InsecureSkipVerify is a bool when if set to true will skip TLS verification for the HTTP client | [optional]
**method** | Option<**String**> | Method is HTTP methods for HTTP Request | [optional]
**success_condition** | Option<**String**> | SuccessCondition is an expression if evaluated to true is considered successful | [optional]
**timeout_seconds** | Option<**i32**> | TimeoutSeconds is request timeout for HTTP Request. Default is 30 seconds | [optional]
**url** | **String** | URL of the HTTP Request | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


