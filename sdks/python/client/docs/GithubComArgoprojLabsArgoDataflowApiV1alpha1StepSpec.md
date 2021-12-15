# GithubComArgoprojLabsArgoDataflowApiV1alpha1StepSpec


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**affinity** | [**Affinity**](Affinity.md) |  | [optional] 
**cat** | [**GithubComArgoprojLabsArgoDataflowApiV1alpha1Cat**](GithubComArgoprojLabsArgoDataflowApiV1alpha1Cat.md) |  | [optional] 
**code** | [**GithubComArgoprojLabsArgoDataflowApiV1alpha1Code**](GithubComArgoprojLabsArgoDataflowApiV1alpha1Code.md) |  | [optional] 
**container** | [**GithubComArgoprojLabsArgoDataflowApiV1alpha1Container**](GithubComArgoprojLabsArgoDataflowApiV1alpha1Container.md) |  | [optional] 
**dedupe** | [**GithubComArgoprojLabsArgoDataflowApiV1alpha1Dedupe**](GithubComArgoprojLabsArgoDataflowApiV1alpha1Dedupe.md) |  | [optional] 
**expand** | [**GithubComArgoprojLabsArgoDataflowApiV1alpha1Expand**](GithubComArgoprojLabsArgoDataflowApiV1alpha1Expand.md) |  | [optional] 
**filter** | [**GithubComArgoprojLabsArgoDataflowApiV1alpha1Filter**](GithubComArgoprojLabsArgoDataflowApiV1alpha1Filter.md) |  | [optional] 
**flatten** | [**GithubComArgoprojLabsArgoDataflowApiV1alpha1Flatten**](GithubComArgoprojLabsArgoDataflowApiV1alpha1Flatten.md) |  | [optional] 
**git** | [**GithubComArgoprojLabsArgoDataflowApiV1alpha1Git**](GithubComArgoprojLabsArgoDataflowApiV1alpha1Git.md) |  | [optional] 
**group** | [**GithubComArgoprojLabsArgoDataflowApiV1alpha1Group**](GithubComArgoprojLabsArgoDataflowApiV1alpha1Group.md) |  | [optional] 
**image_pull_secrets** | [**[LocalObjectReference]**](LocalObjectReference.md) |  | [optional] 
**map** | [**GithubComArgoprojLabsArgoDataflowApiV1alpha1Map**](GithubComArgoprojLabsArgoDataflowApiV1alpha1Map.md) |  | [optional] 
**metadata** | [**GithubComArgoprojLabsArgoDataflowApiV1alpha1Metadata**](GithubComArgoprojLabsArgoDataflowApiV1alpha1Metadata.md) |  | [optional] 
**name** | **str** |  | [optional] 
**node_selector** | **{str: (str,)}** |  | [optional] 
**replicas** | **int** |  | [optional] 
**restart_policy** | **str** |  | [optional] 
**scale** | [**GithubComArgoprojLabsArgoDataflowApiV1alpha1Scale**](GithubComArgoprojLabsArgoDataflowApiV1alpha1Scale.md) |  | [optional] 
**service_account_name** | **str** |  | [optional] 
**sidecar** | [**GithubComArgoprojLabsArgoDataflowApiV1alpha1Sidecar**](GithubComArgoprojLabsArgoDataflowApiV1alpha1Sidecar.md) |  | [optional] 
**sinks** | [**[GithubComArgoprojLabsArgoDataflowApiV1alpha1Sink]**](GithubComArgoprojLabsArgoDataflowApiV1alpha1Sink.md) |  | [optional] 
**sources** | [**[GithubComArgoprojLabsArgoDataflowApiV1alpha1Source]**](GithubComArgoprojLabsArgoDataflowApiV1alpha1Source.md) |  | [optional] 
**terminator** | **bool** |  | [optional] 
**tolerations** | [**[Toleration]**](Toleration.md) |  | [optional] 
**volumes** | [**[Volume]**](Volume.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


