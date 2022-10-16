

# IoArgoprojWorkflowV1alpha1CronWorkflow

CronWorkflow is the definition of a scheduled workflow resource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**apiVersion** | **String** | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#resources |  [optional]
**kind** | **String** | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  [optional]
**metadata** | [**io.kubernetes.client.openapi.models.V1ObjectMeta**](io.kubernetes.client.openapi.models.V1ObjectMeta.md) |  | 
**spec** | [**IoArgoprojWorkflowV1alpha1CronWorkflowSpec**](IoArgoprojWorkflowV1alpha1CronWorkflowSpec.md) |  | 
**status** | [**IoArgoprojWorkflowV1alpha1CronWorkflowStatus**](IoArgoprojWorkflowV1alpha1CronWorkflowStatus.md) |  |  [optional]



