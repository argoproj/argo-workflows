

# VolumeProjection

Projection that may be projected along with other supported volume types. Exactly one of these fields must be set.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**clusterTrustBundle** | [**ClusterTrustBundleProjection**](ClusterTrustBundleProjection.md) |  |  [optional]
**configMap** | [**ConfigMapProjection**](ConfigMapProjection.md) |  |  [optional]
**downwardAPI** | [**DownwardAPIProjection**](DownwardAPIProjection.md) |  |  [optional]
**podCertificate** | [**PodCertificateProjection**](PodCertificateProjection.md) |  |  [optional]
**secret** | [**SecretProjection**](SecretProjection.md) |  |  [optional]
**serviceAccountToken** | [**ServiceAccountTokenProjection**](ServiceAccountTokenProjection.md) |  |  [optional]



