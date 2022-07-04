# Disaster Recovery (DR)

We only store data in your Kubernetes cluster. You should consider backing this up regularly.

Exporting example:

```bash
kubectl get wf,cwf,cwft,wftmpl -A -o yaml > backup.yaml
```

Importing example:

```bash
kubectl apply -f backup.yaml 
```

You should also back-up any SQL persistence you use regularly with whatever tool is provided with it.
