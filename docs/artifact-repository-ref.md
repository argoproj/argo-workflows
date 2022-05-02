# Artifact Repository Ref

> v2.9 and after

You can reduce duplication in your templates by configuring repositories that can be accessed by any workflow. This can also remove sensitive information from your templates.

Create a suitable config map in either (a) your workflows namespace or (b) in the managed namespace:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  # If you want to use this config map by default, name it "artifact-repositories". Otherwise, you can provide a reference to a
  # different config map in `artifactRepositoryRef.configMap`.
  name: my-artifact-repository
  annotations:
    # v3.0 and after - if you want to use a specific key, put that key into this annotation.
    workflows.argoproj.io/default-artifact-repository: default-v1-s3-artifact-repository
data:
  default-v1-s3-artifact-repository: |
    s3:
      bucket: my-bucket
      endpoint: minio:9000
      insecure: true
      accessKeySecret:
        name: my-minio-cred
        key: accesskey
      secretKeySecret:
        name: my-minio-cred
        key: secretkey
  v2-s3-artifact-repository: |
    s3:
      ...
```

You can override the artifact repository for a workflow as follows:

```yaml
spec:
  artifactRepositoryRef:
    configMap: my-artifact-repository # default is "artifact-repositories"
    key: v2-s3-artifact-repository # default can be set by the `workflows.argoproj.io/default-artifact-repository` annotation in config map.
```

This feature gives maximum benefit when used with [key-only artifacts](key-only-artifacts.md).

[Reference](fields.md#artifactrepositoryref).
