# Default Workflow Specs

![alpha](assets/alpha.svg)

> v2.7 and after

It's possible to set default workflow specs which will be written to all workflows if the spec of interest is not set. This can be configurated through the 
workflow controller config [map](https://github.com/argoproj/argo/blob/f2ca045e1cad03d5ec7566ff7200fd8ca575ec5d/workflow/config/config.go#L11) and the field [DefaultWorkflowSpec](https://github.com/argoproj/argo/blob/f2ca045e1cad03d5ec7566ff7200fd8ca575ec5d/workflow/config/config.go#L69). 


In order to edit the Default workflow spec for a controller, edit the workflow config map: 


```bash 
kubectl edit cm/workflow-controller-configmap
```

The default config map for the controller if you follow this example [here](https://github.com/argoproj/argo/blob/master/docs/CONTRIBUTING.md) for local setup will look the following: 


``` yaml 
# Please edit the object below. Lines beginning with a '#' will be ignored,
# and an empty file will abort the edit. If an error occurs while saving this file will be
# reopened with the relevant failures.
#
apiVersion: v1
data:
  config: |
    artifactRepository:
      archiveLogs: true
      s3:
        bucket: my-bucket
        endpoint: minio:9000
        insecure: true
        accessKeySecret:
          name: my-minio-cred
          key: accesskey
        secretKeySecret:
          name: my-minio-cred
          key: secretkey
    persistence:
      connectionPool:
        maxIdleConns: 100
        maxOpenConns: 0
      nodeStatusOffLoad: true
      archive: true
      postgresql:
        host: postgres
        port: 5432
        database: postgres
        tableName: argo_workflows
        userNameSecret:
          name: argo-postgres-config
          key: username
        passwordSecret:
          name: argo-postgres-config
          key: password
kind: ConfigMap
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"v1","data":{"config":"artifactRepository:\n  archiveLogs: true\n  s3:\n    bucket: my-bucket\n    endpoint: minio:9000\n    insecure: true\n    accessKeySecret:\n      name: my-minio$
  creationTimestamp: "2020-03-11T13:28:31Z"
  name: workflow-controller-configmap
  namespace: argo
  resourceVersion: "23477"
  selfLink: /api/v1/namespaces/argo/configmaps/workflow-controller-configmap
  uid: 05f70535-22db-49ca-a4a9-afd513b49737
```
As an example the time for a argo workflow to live after Finish will be set in the spec, this filed is known as ```secondsAfterCompletion``` in the ```ttlStrategy```. The updated configMap will then look like this: 


``` yaml 
# Please edit the object below. Lines beginning with a '#' will be ignored,
# and an empty file will abort the edit. If an error occurs while saving this file will be
# reopened with the relevant failures.
#
apiVersion: v1
data:
  config: |
    artifactRepository:
      archiveLogs: true
      s3:
        bucket: my-bucket
        endpoint: minio:9000
        insecure: true
        accessKeySecret:
          name: my-minio-cred
          key: accesskey
        secretKeySecret:
          name: my-minio-cred
          key: secretkey
    persistence:
      connectionPool:
        maxIdleConns: 100
        maxOpenConns: 0
      nodeStatusOffLoad: true
      archive: true
      postgresql:
        host: postgres
        port: 5432
        database: postgres
        tableName: argo_workflows
        userNameSecret:
          name: argo-postgres-config
          key: username
        passwordSecret:
          name: argo-postgres-config
          key: password
    workflowDefaults:
      ttlStrategy:
        secondsAfterCompletion: 10
kind: ConfigMap
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"v1","data":{"config":"artifactRepository:\n  archiveLogs: true\n  s3:\n    bucket: my-bucket\n    endpoint: minio:9000\n    insecure: true\n    accessKeySecret:\n      name: my-minio$
  creationTimestamp: "2020-03-11T13:28:31Z"
  name: workflow-controller-configmap
  namespace: argo
  resourceVersion: "23964"
  selfLink: /api/v1/namespaces/argo/configmaps/workflow-controller-configmap
  uid: 05f70535-22db-49ca-a4a9-afd513b49737
```

In order to test it a example workflow can be submited, in this case the [coinflip example](https://github.com/argoproj/argo/blob/master/examples/coinflip.yaml). 

```bash 
kubectl create -f examples/coinflip.yaml 
```

to verify that the the defaultd are set run 

```bash
kubectl -n argo describe workflow.argoproj.io/[YOUR_ARGO_WORKFLOW_NAME]
```

You should then see the filed, Ttl Strategy populated
```yaml
Ttl Strategy:
  Seconds After Completion:  10
```
