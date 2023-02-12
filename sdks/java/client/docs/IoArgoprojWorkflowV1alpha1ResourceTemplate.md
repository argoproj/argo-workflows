

# IoArgoprojWorkflowV1alpha1ResourceTemplate

ResourceTemplate is a template subtype to manipulate kubernetes resources

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**action** | **String** | Action is the action to perform to the resource. Must be one of: get, create, apply, delete, replace, patch | 
**failureCondition** | **String** | FailureCondition is a label selector expression which describes the conditions of the k8s resource in which the step was considered failed |  [optional]
**flags** | **List&lt;String&gt;** | Flags is a set of additional options passed to kubectl before submitting a resource I.e. to disable resource validation: flags: [  \&quot;--validate&#x3D;false\&quot;  # disable resource validation ] |  [optional]
**manifest** | **String** | Manifest contains the kubernetes manifest |  [optional]
**manifestFrom** | [**IoArgoprojWorkflowV1alpha1ManifestFrom**](IoArgoprojWorkflowV1alpha1ManifestFrom.md) |  |  [optional]
**mergeStrategy** | **String** | MergeStrategy is the strategy used to merge a patch. It defaults to \&quot;strategic\&quot; Must be one of: strategic, merge, json |  [optional]
**setOwnerReference** | **Boolean** | SetOwnerReference sets the reference to the workflow on the OwnerReference of generated resource. |  [optional]
**successCondition** | **String** | SuccessCondition is a label selector expression which describes the conditions of the k8s resource in which it is acceptable to proceed to the following step |  [optional]



