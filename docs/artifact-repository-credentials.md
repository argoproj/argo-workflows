# Artifact Repository Credentials

![alpha](assets/alpha.svg)

> v2.9 and after

You can reduce duplication in your templates by configuring repository credentials that can be accessed by any workflow. This can also remove sensitive information from your templates.

Configuration of[workflow-controller-configmap.yaml](workflow-controller-configmap.yaml):

```
artifactRepositoryCredentials:
  - name: my-artifactory-repository-credentials
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

Example [../examples/artifact-repository-credentials.yaml](../examples/artifact-repository-credentials.yaml)

```
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: artifactory-repository-credentials-
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: docker/whalesay:latest
        command: [sh, -c]
        args: ["cowsay hello world | tee /tmp/hello_world.txt"]
      outputs:
        artifacts:
          - name: hello_world
            path: /tmp/hello_world.txt
            credentialName: my-artifactory-repository-credentials
```
