---
name: Bug report
about: Create a report to help us improve
title: ''
labels: 'bug'
assignees: ''
---
Checklist:

 * [ ] I've included the version.
 * [ ] I've included reproduction steps.
 * [ ] I've included the workflow YAML.
 * [ ] I've included the logs.
 
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
argo get <workflowname>
```
- executor logs:
```
kubectl logs <failedpodname> -c init
kubectl logs <failedpodname> -c wait
```
- workflow-controller logs:
```
kubectl logs -n argo $(kubectl get pods -l app=workflow-controller -n argo -o name)
```

**Logs**

```
argo get <workflowname>
kubectl logs <failedpodname> -c init
kubectl logs <failedpodname> -c wait
kubectl logs -n argo $(kubectl get pods -l app=workflow-controller -n argo -o name)
```
