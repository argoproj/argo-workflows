**Is this a BUG REPORT or FEATURE REQUEST?**:

**What happened**:

**What you expected to happen**:

**How to reproduce it (as minimally and precisely as possible)**:

**Anything else we need to know?**:

**Environment**:
- Argo version:
```
$ argo version
```
- Kubernetes version :
```
$ kubectl version -o yaml
```

**Other debugging information (if applicable)**:
- workflow result:
```
$ argo get <workflowname>
```
- executor logs:
```
$ kubectl logs <failedpodname> -c init
$ kubectl logs <failedpodname> -c wait
```
- workflow-controller logs:
```
$ kubectl logs -n kube-system $(kubectl get pods -l app=workflow-controller -n kube-system -o name)
```
