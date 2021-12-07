

# IoArgoprojWorkflowV1alpha1SemaphoreRef

SemaphoreRef is a reference of Semaphore

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**configMapKeyRef** | [**io.kubernetes.client.openapi.models.V1ConfigMapKeySelector**](io.kubernetes.client.openapi.models.V1ConfigMapKeySelector.md) |  |  [optional]
**selectors** | [**List&lt;IoArgoprojWorkflowV1alpha1SyncSelector&gt;**](IoArgoprojWorkflowV1alpha1SyncSelector.md) | Selectors is a list of references to dynamic values (like parameters, labels, annotations) that can be added to semaphore key to make concurrency more customizable |  [optional]



