

# EnvFromSource

EnvFromSource represents the source of a set of ConfigMaps

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**configMapRef** | [**ConfigMapEnvSource**](ConfigMapEnvSource.md) |  |  [optional]
**prefix** | **String** | An optional identifier to prepend to each key in the ConfigMap. Must be a C_IDENTIFIER. |  [optional]
**secretRef** | [**SecretEnvSource**](SecretEnvSource.md) |  |  [optional]



