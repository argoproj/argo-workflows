

# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AWSLambdaTrigger


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**accessKey** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**functionName** | **String** | FunctionName refers to the name of the function to invoke. |  [optional]
**invocationType** | **String** | Choose from the following options.     * RequestResponse (default) - Invoke the function synchronously. Keep    the connection open until the function returns a response or times out.    The API response includes the function response and additional data.     * Event - Invoke the function asynchronously. Send events that fail multiple    times to the function&#39;s dead-letter queue (if it&#39;s configured). The API    response only includes a status code.     * DryRun - Validate parameter values and verify that the user or role    has permission to invoke the function. +optional |  [optional]
**parameters** | [**List&lt;GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter.md) |  |  [optional]
**payload** | [**List&lt;GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter.md) | Payload is the list of key-value extracted from an event payload to construct the request payload. |  [optional]
**region** | **String** |  |  [optional]
**roleARN** | **String** |  |  [optional]
**secretKey** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]



