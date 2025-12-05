# Secrets

Argo supports the same secrets syntax and mechanisms as Kubernetes Pod specs, which allows access to secrets as environment variables or volume mounts. See the [Kubernetes documentation](https://kubernetes.io/docs/concepts/configuration/secret/) for more information.

/// tab | YAML

```yaml
# To run this example, first create the secret by running:
# kubectl create secret generic my-secret --from-literal=mypassword=S00perS3cretPa55word
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: secret-example-
spec:
  entrypoint: print-secrets
  # To access secrets as files, add a volume entry in spec.volumes[] and
  # then in the container template spec, add a mount using volumeMounts.
  volumes:
  - name: my-secret-vol
    secret:
      secretName: my-secret     # name of an existing k8s secret
  templates:
  - name: print-secrets
    container:
      image: alpine:3.7
      command: [sh, -c]
      args: ['
        echo "secret from env: $MYSECRETPASSWORD";
        echo "secret from file: `cat /secret/mountpath/mypassword`"
      ']
      # To access secrets as environment variables, use the k8s valueFrom and
      # secretKeyRef constructs.
      env:
      - name: MYSECRETPASSWORD  # name of env var
        valueFrom:
          secretKeyRef:
            name: my-secret     # name of an existing k8s secret
            key: mypassword     # 'key' subcomponent of the secret
      volumeMounts:
      - name: my-secret-vol     # mount file containing secret at /secret/mountpath
        mountPath: "/secret/mountpath"
```

///

/// tab | Python

```python
from hera.workflows import Container, Workflow
from hera.workflows.models import (
    EnvVar,
    EnvVarSource,
    SecretKeySelector,
    SecretVolumeSource,
    Volume,
    VolumeMount,
)

with Workflow(
    generate_name="secret-example-",
    entrypoint="print-secrets",
    volumes=[
        Volume(name="my-secret-vol", secret=SecretVolumeSource(secret_name="my-secret"))
    ],
) as w:
    Container(
        name="print-secrets",
        image="alpine:3.7",
        command=["sh", "-c"],
        args=[
            ' echo "secret from env: $MYSECRETPASSWORD"; echo "secret from file: `cat /secret/mountpath/mypassword`" '
        ],
        env=[
            EnvVar(
                name="MYSECRETPASSWORD",
                value_from=EnvVarSource(
                    secret_key_ref=SecretKeySelector(key="mypassword", name="my-secret")
                ),
            )
        ],
        volume_mounts=[
            VolumeMount(mount_path="/secret/mountpath", name="my-secret-vol")
        ],
    )
```

///
