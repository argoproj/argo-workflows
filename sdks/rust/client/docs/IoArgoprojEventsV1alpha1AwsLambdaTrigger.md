# IoArgoprojEventsV1alpha1AwsLambdaTrigger

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**access_key** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**function_name** | Option<**String**> | FunctionName refers to the name of the function to invoke. | [optional]
**invocation_type** | Option<**String**> | Choose from the following options.     * RequestResponse (default) - Invoke the function synchronously. Keep    the connection open until the function returns a response or times out.    The API response includes the function response and additional data.     * Event - Invoke the function asynchronously. Send events that fail multiple    times to the function's dead-letter queue (if it's configured). The API    response only includes a status code.     * DryRun - Validate parameter values and verify that the user or role    has permission to invoke the function. +optional | [optional]
**parameters** | Option<[**Vec<crate::models::IoArgoprojEventsV1alpha1TriggerParameter>**](io.argoproj.events.v1alpha1.TriggerParameter.md)> |  | [optional]
**payload** | Option<[**Vec<crate::models::IoArgoprojEventsV1alpha1TriggerParameter>**](io.argoproj.events.v1alpha1.TriggerParameter.md)> | Payload is the list of key-value extracted from an event payload to construct the request payload. | [optional]
**region** | Option<**String**> |  | [optional]
**role_arn** | Option<**String**> |  | [optional]
**secret_key** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


