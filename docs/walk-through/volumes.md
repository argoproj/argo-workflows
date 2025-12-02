# Volumes

The following example dynamically creates a volume and then uses the volume in a two step workflow.

/// tab | YAML

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: volumes-pvc-
spec:
  entrypoint: volumes-pvc-example
  volumeClaimTemplates:                 # define volume, same syntax as k8s Pod spec
  - metadata:
      name: workdir                     # name of volume claim
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 1Gi                  # Gi => 1024 * 1024 * 1024

  templates:
  - name: volumes-pvc-example
    steps:
    - - name: generate
        template: hello-world-to-file
    - - name: print
        template: print-message-from-file

  - name: hello-world-to-file
    container:
      image: busybox
      command: [sh, -c]
      args: ["echo generating message in volume; echo hello world | tee /mnt/vol/hello_world.txt"]
      # Mount workdir volume at /mnt/vol before invoking the container
      volumeMounts:                     # same syntax as k8s Pod spec
      - name: workdir
        mountPath: /mnt/vol

  - name: print-message-from-file
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo getting message from volume; find /mnt/vol; cat /mnt/vol/hello_world.txt"]
      # Mount workdir volume at /mnt/vol before invoking the container
      volumeMounts:                     # same syntax as k8s Pod spec
      - name: workdir
        mountPath: /mnt/vol

```

///

/// tab | Python

```python
from hera.workflows import Container, Steps, Workflow
from hera.workflows.models import (
    ObjectMeta,
    PersistentVolumeClaim,
    PersistentVolumeClaimSpec,
    VolumeMount,
    VolumeResourceRequirements,
)

with Workflow(
    generate_name="volumes-pvc-",
    entrypoint="volumes-pvc-example",
    volume_claim_templates=[
        PersistentVolumeClaim(
            metadata=ObjectMeta(name="workdir"),
            spec=PersistentVolumeClaimSpec(
                access_modes=["ReadWriteOnce"],
                resources=VolumeResourceRequirements(
                    requests={"storage": "1Gi"}
                ),
            ),
        )
    ],
) as w:
    hello_to_file = Container(
        name="hello-world-to-file",
        image="busybox",
        command=["sh", "-c"],
        args=[
            "echo generating message in volume; echo hello world | tee /mnt/vol/hello_world.txt"
        ],
        volume_mounts=[VolumeMount(mount_path="/mnt/vol", name="workdir")],
    )
    print_from_file = Container(
        name="print-message-from-file",
        image="alpine:latest",
        command=["sh", "-c"],
        args=[
            "echo getting message from volume; find /mnt/vol; cat /mnt/vol/hello_world.txt"
        ],
        volume_mounts=[VolumeMount(mount_path="/mnt/vol", name="workdir")],
    )
    with Steps(name="volumes-pvc-example") as steps:
        hello_to_file(name="generate")
        print_from_file(name="print")
```

///

Volumes are a very useful way to move large amounts of data from one step in a workflow to another. Depending on the system, some volumes may be accessible concurrently from multiple steps.

In some cases, you want to access an already existing volume rather than creating/destroying one dynamically.

```yaml
# Define Kubernetes PVC
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: my-existing-volume
spec:
  accessModes: [ "ReadWriteOnce" ]
  resources:
    requests:
      storage: 1Gi
```

/// tab | YAML

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: volumes-existing-
spec:
  entrypoint: volumes-existing-example
  volumes:
  # Pass my-existing-volume as an argument to the volumes-existing-example template
  # Same syntax as k8s Pod spec
  - name: workdir
    persistentVolumeClaim:
      claimName: my-existing-volume

  templates:
  - name: volumes-existing-example
    steps:
    - - name: generate
        template: hello-world-to-file
    - - name: print
        template: print-message-from-file

  - name: hello-world-to-file
    container:
      image: busybox
      command: [sh, -c]
      args: ["echo generating message in volume; echo hello world | tee /mnt/vol/hello_world.txt"]
      volumeMounts:
      - name: workdir
        mountPath: /mnt/vol

  - name: print-message-from-file
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo getting message from volume; find /mnt/vol; cat /mnt/vol/hello_world.txt"]
      volumeMounts:
      - name: workdir
        mountPath: /mnt/vol
```

///

/// tab | Python

```python
from hera.workflows import Container, Steps, Workflow
from hera.workflows.models import PersistentVolumeClaimVolumeSource, Volume, VolumeMount

with Workflow(
    generate_name="volumes-existing-",
    entrypoint="volumes-existing-example",
    volumes=[
        Volume(
            name="workdir",
            persistent_volume_claim=PersistentVolumeClaimVolumeSource(
                claim_name="my-existing-volume"
            ),
        )
    ],
) as w:
    hello_to_file = Container(
        name="hello-world-to-file",
        image="busybox",
        command=["sh", "-c"],
        args=[
            "echo generating message in volume; echo hello world | tee /mnt/vol/hello_world.txt"
        ],
        volume_mounts=[VolumeMount(mount_path="/mnt/vol", name="workdir")],
    )
    print_from_file = Container(
        name="print-message-from-file",
        image="alpine:latest",
        command=["sh", "-c"],
        args=[
            "echo getting message from volume; find /mnt/vol; cat /mnt/vol/hello_world.txt"
        ],
        volume_mounts=[VolumeMount(mount_path="/mnt/vol", name="workdir")],
    )
    with Steps(name="volumes-existing-example") as steps:
        hello_to_file(name="generate")
        print_from_file(name="print")

```

///

It's also possible to declare existing volumes at the template level, instead of the workflow level.
Workflows can generate volumes using a [`resource`](kubernetes-resources.md) step.

/// tab | YAML

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: template-level-volume-
spec:
  entrypoint: generate-and-use-volume
  templates:
  - name: generate-and-use-volume
    steps:
    - - name: generate-volume
        template: generate-volume
        arguments:
          parameters:
            - name: pvc-size
              # In a real-world example, this could be generated by a previous workflow step.
              value: '1Gi'
    - - name: generate
        template: hello-world-to-file
        arguments:
          parameters:
            - name: pvc-name
              value: '{{steps.generate-volume.outputs.parameters.pvc-name}}'
    - - name: print
        template: print-message-from-file
        arguments:
          parameters:
            - name: pvc-name
              value: '{{steps.generate-volume.outputs.parameters.pvc-name}}'

  - name: generate-volume
    inputs:
      parameters:
        - name: pvc-size
    resource:
      action: create
      setOwnerReference: true
      manifest: |
        apiVersion: v1
        kind: PersistentVolumeClaim
        metadata:
          generateName: pvc-example-
        spec:
          accessModes: ['ReadWriteOnce', 'ReadOnlyMany']
          resources:
            requests:
              storage: '{{inputs.parameters.pvc-size}}'
    outputs:
      parameters:
        - name: pvc-name
          valueFrom:
            jsonPath: '{.metadata.name}'

  - name: hello-world-to-file
    inputs:
      parameters:
        - name: pvc-name
    volumes:
      - name: workdir
        persistentVolumeClaim:
          claimName: '{{inputs.parameters.pvc-name}}'
    container:
      image: busybox
      command: [sh, -c]
      args: ["echo generating message in volume; echo hello world | tee /mnt/vol/hello_world.txt"]
      volumeMounts:
      - name: workdir
        mountPath: /mnt/vol

  - name: print-message-from-file
    inputs:
        parameters:
          - name: pvc-name
    volumes:
      - name: workdir
        persistentVolumeClaim:
          claimName: '{{inputs.parameters.pvc-name}}'
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo getting message from volume; find /mnt/vol; cat /mnt/vol/hello_world.txt"]
      volumeMounts:
      - name: workdir
        mountPath: /mnt/vol

```

///

/// tab | Python

```python
from hera.workflows import Container, Resource, Steps, Workflow
from hera.workflows.models import (
    Arguments,
    Inputs,
    Outputs,
    Parameter,
    PersistentVolumeClaimVolumeSource,
    ValueFrom,
    Volume,
    VolumeMount,
)

with Workflow(
    generate_name="template-level-volume-",
    entrypoint="generate-and-use-volume",
) as w:
    generate_volume = Resource(
        name="generate-volume",
        inputs=[Parameter(name="pvc-size")],
        outputs=[Parameter(name="pvc-name", value_from=ValueFrom(json_path="{.metadata.name}"))],
        action="create",
        manifest="apiVersion: v1\nkind: PersistentVolumeClaim\nmetadata:\n  generateName: pvc-example-\nspec:\n  accessModes: ['ReadWriteOnce', 'ReadOnlyMany']\n  resources:\n    requests:\n      storage: '{{inputs.parameters.pvc-size}}'\n",
        set_owner_reference=True,
    )
    hello_to_file = Container(
        name="hello-world-to-file",
        image="busybox",
        command=["sh", "-c"],
        args=["echo generating message in volume; echo hello world | tee /mnt/vol/hello_world.txt"],
        inputs=[Parameter(name="pvc-name")],
        volumes=[
            Volume(
                name="workdir",
                persistent_volume_claim=PersistentVolumeClaimVolumeSource(claim_name="{{inputs.parameters.pvc-name}}"),
            )
        ],
        volume_mounts=[VolumeMount(mount_path="/mnt/vol", name="workdir")],
    )
    print_from_file = Container(
        name="print-message-from-file",
        image="alpine:latest",
        command=["sh", "-c"],
        args=["echo getting message from volume; find /mnt/vol; cat /mnt/vol/hello_world.txt"],
        inputs=[Parameter(name="pvc-name")],
        volumes=[
            Volume(
                name="workdir",
                persistent_volume_claim=PersistentVolumeClaimVolumeSource(claim_name="{{inputs.parameters.pvc-name}}"),
            )
        ],
        volume_mounts=[VolumeMount(mount_path="/mnt/vol", name="workdir")],
    )
    with Steps(name="generate-and-use-volume") as steps:
        generate_volume_step = generate_volume(
            name="generate-volume",
            arguments={"pvc-size": "1Gi"},
        )
        hello_to_file(
            name="generate",
            arguments=generate_volume_step.get_parameter("pvc-name"),
        )
        print_from_file(
            name="print",
            arguments=generate_volume_step.get_parameter("pvc-name"),
        )
```

///
