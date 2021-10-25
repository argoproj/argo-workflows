# V1ProjectedVolumeSource

Represents a projected volume source
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**default_mode** | **int** | Mode bits to use on created files by default. Must be a value between 0 and 0777. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set. | [optional] 
**sources** | [**list[V1VolumeProjection]**](V1VolumeProjection.md) | list of volume projections | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


