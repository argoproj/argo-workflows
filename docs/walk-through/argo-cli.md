# Argo CLI

In case you want to follow along with this walk-through, here's a quick overview of the most useful argo command line interface (CLI) commands.

```bash
argo submit hello-world.yaml    # submit a workflow spec to Kubernetes
argo list                       # list current workflows
argo get hello-world-xxx        # get info about a specific workflow
argo logs hello-world-xxx       # print the logs from a workflow
argo delete hello-world-xxx     # delete workflow
```

You can also run workflow specs directly using `kubectl` but the Argo CLI provides syntax checking, nicer output, and requires less typing.

```bash
kubectl create -f hello-world.yaml
kubectl get wf
kubectl get wf hello-world-xxx
kubectl get po --selector=workflows.argoproj.io/workflow=hello-world-xxx
kubectl logs hello-world-xxx-yyy -c main
kubectl delete wf hello-world-xxx
```
