

# V1ServiceAccountTokenProjection

ServiceAccountTokenProjection represents a projected service account token volume. This projection can be used to insert a service account token into the pods runtime filesystem for use against APIs (Kubernetes API Server or otherwise).
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**audience** | **String** |  |  [optional]
**expirationSeconds** | **String** |  |  [optional]
**path** | **String** | Path is the path relative to the mount point of the file to project the token into. |  [optional]



