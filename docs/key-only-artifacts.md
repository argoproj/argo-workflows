# Key-Only Artifacts

> v3.0 and after

A key-only artifact is an input or output artifact where you only specify the key, omitting the bucket, secrets etc. When these are omitted, the bucket/secrets from the configured artifact repository is used.

This allows you to move the configuration of the artifact repository out of the workflow specification.

This is closely related to [artifact repository ref](artifact-repository-ref.md). You'll want to use them together for maximum benefit.

This should probably be your default if you're using v3.0:

* Reduces the size of workflows (improved performance).
* User owned artifact repository set-up configuration (simplified management).
* Decouples the artifact location configuration from the workflow. Allowing you to re-configure the artifact repository without changing your workflows or templates.

Example:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: key-only-artifacts-
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: generate
            template: generate
          - name: consume
            template: consume
            dependencies:
              - generate
    - name: generate
      container:
        image: argoproj/argosay:v2
        args: [ echo, hello, /mnt/file ]
      outputs:
        artifacts:
          - name: file
            path: /mnt/file
            s3:
              key: my-file
    - name: consume
      container:
        image: argoproj/argosay:v2
        args: [cat, /tmp/file]
      inputs:
        artifacts:
          - name: file
            path: /tmp/file
            s3:
              key: my-file
```

!!! WARNING
    The location data is not longer stored in `/status/nodes`. Any tooling that relies on this will need to be updated.
