# Default Workflow Spec

![alpha](assets/alpha.svg)

> v2.7 and after

It's possible to set default workflow specs which will be written to all workflows if the spec of interest is not set. This can be configurated through the 
workflow controller config [map](../workflow/config/config.go#L11) and the field [DefaultWorkflowSpec](../workflow/config/config.go#L69). 


In order to edit the Default workflow spec for a controller, edit the workflow config map: 


```bash 
kubectl edit cm/workflow-controller-configmap
```


As an example the time for a argo workflow to live after finish can be set, in the spec this field is known as ```secondsAfterCompletion``` in the ```ttlStrategy```. Example of how the config map could look with this filed can be found [here](./workflow-controller-configmap.yaml).

In order to test it a example workflow can be submited, in this case the [coinflip example](../examples/coinflip.yaml), the following can be run:

```bash 
argo submit ./examples/coinflip.yaml
```

to verify that the the defaultd are set run 

```bash
argo get [YOUR_ARGO_WORKFLOW_NAME]
```

You should then see the field, Ttl Strategy populated
```yaml
Ttl Strategy:
  Seconds After Completion:  10
```
