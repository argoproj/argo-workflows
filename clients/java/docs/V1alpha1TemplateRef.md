

# V1alpha1TemplateRef

TemplateRef is a reference of template resource.
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **String** | Name is the resource name of the template. |  [optional]
**runtimeResolution** | **Boolean** | RuntimeResolution skips validation at creation time. By enabling this option, you can create the referred workflow template before the actual runtime. |  [optional]
**template** | **String** | Template is the name of referred template in the resource. |  [optional]



