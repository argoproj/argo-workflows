

# IoArgoprojWorkflowV1alpha1EnumFrom

EnumFrom lists Kubernetes resources to offer as a parameter's values when it is submitted from the UI or with `argo submit --interactive`. The Argo Server lists them with the submitting user's credentials. It only affects how the parameter is asked for: the submitted value is not checked against the list.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**jsonPath** | **String** | JSONPath extracts the values from each listed resource, e.g. &#x60;{.metadata.name}&#x60; | 
**labelSelector** | **String** | LabelSelector filters the listed resources, e.g. &#x60;tier&#x3D;web&#x60;. It may reference other workflow parameters, so the list can depend on another parameter&#39;s value. |  [optional]
**resource** | [**IoArgoprojWorkflowV1alpha1ResourceSelector**](IoArgoprojWorkflowV1alpha1ResourceSelector.md) |  | 



