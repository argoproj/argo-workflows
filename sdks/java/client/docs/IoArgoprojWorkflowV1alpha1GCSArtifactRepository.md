

# IoArgoprojWorkflowV1alpha1GCSArtifactRepository

GCSArtifactRepository defines the controller configuration for a GCS artifact repository

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**bucket** | **String** | Bucket is the name of the bucket |  [optional]
**keyFormat** | **String** | KeyFormat is defines the format of how to store keys. Can reference workflow variables |  [optional]
**serviceAccountKeySecret** | [**SecretKeySelector**](SecretKeySelector.md) |  |  [optional]



