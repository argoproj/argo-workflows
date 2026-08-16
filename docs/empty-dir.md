# Empty Dir

Not all [workflow executors](workflow-executors.md) can get output artifacts/parameters from the base layer (e.g. `/tmp`).
It is unlikely you can get output artifacts/parameters from the base layer if you run your workflow pods with a [security context](workflow-pod-security-context.md).

You can work around this constraint by mounting volumes onto your pod. The easiest way to do this is to use an `emptyDir` volume.

!!! Note
    This is only needed for output artifacts/parameters. Input artifacts/parameters are automatically mounted to an empty-dir if needed

This example shows how to mount an output volume:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: empty-dir-
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
        command: [sh, -c]
        args: ["cowsay hello world | tee /mnt/out/hello_world.txt"]
        volumeMounts:
          - name: out
            mountPath: /mnt/out
      volumes:
        - name: out
          emptyDir: { }
      outputs:
        parameters:
          - name: message
            valueFrom:
              path: /mnt/out/hello_world.txt
```
