# Artifact Repository Ref

![GA](assets/ga.svg)

> v2.9 and after

You can reduce duplication in your templates by configuring repositories that can be accessed by any workflow. This can also remove sensitive information from your templates.

Create a suitable config map in either (a) your workflows namespace or (b) in the managed namespace:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: artifact-repositories
data:
  default: |
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

```yaml
spec:
  artifactRepositoryRef:
    namespace: my-ns # default is the  workflows' namespace
    configMap: my-cm # default is "artifact-repositories"
    key: my-key # default is "default"
```

Reference: [fields.md#artifactrepositoryref](fields.md#artifactrepositoryref).