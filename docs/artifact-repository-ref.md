# Artifact Repository Ref

![GA](assets/ga.svg)

> v2.9 and after

You can reduce duplication in your templates by configuring repositories that can be accessed by any workflow. This can also remove sensitive information from your templates.

Create a suitable config map in either (a) your workflows namespace or (b) in the managed namespace:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
    # if you want to use this config map by default - name it "artifact-repositories" 
  name: artifact-repositories
  annotations:
    # if you want to use a specific key, put that's key into this annotation 
    workflows.argoproj.io/default-artifact-repository: default-v1
data:
  default-v1: |
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
    configMap: my-cm # default is "artifact-repositories"
    key: my-key # default can be set by the annotation
```

Reference: [fields.md#artifactrepositoryref](fields.md#artifactrepositoryref).