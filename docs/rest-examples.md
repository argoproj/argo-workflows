# API Examples

Document contains couple of examples of workflow JSON's to submit via argo-server REST API.

> v2.5 and after

Assuming

* the namespace of argo-server is argo
* authentication is turned off (otherwise provide Authorization header)
* argo-server is available on localhost:2746

## Submitting workflow

```bash
curl --request POST \
  --url https://localhost:2746/api/v1/workflows/argo \
  --header 'content-type: application/json' \
  --data '{
  "namespace": "argo",
  "serverDryRun": false,
  "workflow": {
      "metadata": {
        "generateName": "hello-world-",
        "namespace": "argo",
        "labels": {
          "workflows.argoproj.io/completed": "false"
         }
      },
     "spec": {
       "templates": [
        {
         "name": "hello-world",
         "arguments": {},
         "inputs": {},
         "outputs": {},
         "metadata": {},
         "container": {
          "name": "",
          "image": "busybox",
          "command": [
            "echo"
          ],
          "args": [
            "hello world"
          ],
          "resources": {}
        }
      }
    ],
    "entrypoint": "hello-world",
    "arguments": {}
  }
}
}'
```

## Getting workflows for namespace argo

```bash
curl --request GET \
  --url https://localhost:2746/api/v1/workflows/argo
```

## Getting single workflow for namespace argo

```bash
curl --request GET \
  --url https://localhost:2746/api/v1/workflows/argo/abc-dthgt
```

## Deleting single workflow for namespace argo

```bash
curl --request DELETE \
  --url https://localhost:2746/api/v1/workflows/argo/abc-dthgt
```
