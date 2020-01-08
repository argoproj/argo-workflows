# Workflow Executors

A workflow executor is a process that conforms to a specific interface that allows Argo to perform certain actions like monitoring pod logs, collecting artifacts, managing container lifecycles, etc..

The executor to be used in your workflows can be changed in [the configmap](./workflow-controller-configmap.yaml) under the `containerRuntimeExecutor` key.

## Docker (docker)

**default**

### Pros

* Most reliable and well-tested executor
* Supports all workflow examples
* Highly scalable as it communicates directly with the docker daemon for heavy lifting
* Output artifacts can be located on the base layer (e.g. /tmp)

### Cons

* Least secure as it required `docker.sock` of the host to be mounted which is often rejected by OPA.

## Kubelet (kubelet)

### Pros

* Secure since you cannot escape the privileges of the pod's service account
* Moderately scalable  Log retrieval and container operations are performed against the kubelet

### Cons

* Additional kubelet configuration may be required
* Output artifacts can only be saved on volumes (e.g. emptyDir) and not the base image layer (e.g. /tmp)

## Kubernetes API (k8sapi)

### Pros

* Secure since you cannot escape the privileges of the pod's service account
* No extra configuration is required

### Cons

* Least scalable since log retrieval and container operations are performed against the kubernetes api
* Output artifacts can only be saved on volumes (e.g. emptyDir) and not the base image layer (e.g. /tmp)

## Process Namespace Sharing (pns)

### Pros

* Secure since you cannot escape the privileges of the pod's service account
* Output artifacts can be located on the base layer (e.g. /tmp)
* Highly scalable.  Process polling is done over procfs rather than the Kubernetes/Kubelet API
* Process will no longer run with PID 1

### Cons

* Immature
* Cannot capture artifact directories from base image layer which has a volume mounted under it
