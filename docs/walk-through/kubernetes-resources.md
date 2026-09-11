# Kubernetes Resources

In many cases, you will want to manage Kubernetes resources from Argo workflows. The resource template allows you to create, delete or updated any type of Kubernetes resource.

```yaml
# in a workflow. The resource template type accepts any k8s manifest
# (including CRDs) and can perform any `kubectl` action against it (e.g. create,
# apply, delete, patch).
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: k8s-jobs-
spec:
  entrypoint: pi-tmpl
  templates:
  - name: pi-tmpl
    resource:                   # indicates that this is a resource template
      action: create            # can be any kubectl action (e.g. create, delete, apply, patch)
      # The successCondition and failureCondition are optional expressions.
      # If failureCondition is true, the step is considered failed.
      # If successCondition is true, the step is considered successful.
      # They use kubernetes label selection syntax and can be applied against any field
      # of the resource (not just labels).
      # For successCondition: Multiple AND conditions can be represented by comma
      # delimited expressions.
      # For failureCondition: Multiple OR conditions can be represented by comma
      # delimited expressions.
      # For more details: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
      successCondition: status.succeeded > 0
      failureCondition: status.failed > 3
      manifest: |               #put your kubernetes spec here
        apiVersion: batch/v1
        kind: Job
        metadata:
          generateName: pi-job-
        spec:
          template:
            metadata:
              name: pi
            spec:
              containers:
              - name: pi
                image: perl
                command: ["perl",  "-Mbignum=bpi", "-wle", "print bpi(2000)"]
              restartPolicy: Never
          backoffLimit: 4
```

**Note:**
Currently only a single resource can be managed by a resource template so either a `generateName` or `name` must be provided in the resource's meta-data.

Resources created in this way are independent of the workflow. If you want the resource to be deleted when the workflow is deleted then you can use [Kubernetes garbage collection](https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/) with the workflow resource as an owner reference ([example](https://github.com/argoproj/argo-workflows/tree/main/examples/k8s-owner-reference.yaml)).

You can also collect data about the resource in output parameters (see more at [k8s-jobs.yaml](https://github.com/argoproj/argo-workflows/tree/main/examples/k8s-jobs.yaml))

**Note:**
When patching, the resource will accept another attribute, `mergeStrategy`, which can either be `strategic`, `merge`, or `json`. If this attribute is not supplied, it will default to `strategic`. Keep in mind that Custom Resources cannot be patched with `strategic`, so a different strategy must be chosen. For example, suppose you have the [`CronTab` CRD](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#create-a-customresourcedefinition) defined, and the following instance of a `CronTab`:

```yaml
apiVersion: "stable.example.com/v1"
kind: CronTab
spec:
  cronSpec: "* * * * */5"
  image: my-awesome-cron-image
```

This `CronTab` can be modified using the following Argo Workflow:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: k8s-patch-
spec:
  entrypoint: cront-tmpl
  templates:
  - name: cront-tmpl
    resource:
      action: patch
      mergeStrategy: merge                 # Must be one of [strategic merge json]
      manifest: |
        apiVersion: "stable.example.com/v1"
        kind: CronTab
        spec:
          cronSpec: "* * * * */10"
          image: my-awesome-cron-image
```
