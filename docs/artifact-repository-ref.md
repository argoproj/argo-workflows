# Artifact Repository Ref

![GA](assets/ga.svg)

> v2.9 and after

You can reduce duplication in your templates by configuring repositories that can be accessed by any workflow. This can also remove sensitive information from your templates.

Create a suitable config map in either (a) your workflows namespace or (b) in the Argo's namespace, the default name is `artifact-repositories`:

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: artifact-repositories
data:
  minio: |
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
```

You can override the repository for a workflow as follows:

```
spec:
  artifactRepositoryRef:
    key: minio
```

Reference: [fields.md#artifactrepositoryref](fields.md#artifactrepositoryref).