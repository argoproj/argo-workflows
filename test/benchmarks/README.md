# Benchmarks

## Benchmarking multiple-ref template creation

Benchmarks were performed manually. This should be automated and performed in pipeline. For now let's just describe a process. 

Use fresh cluster in latests version.

Argo server started with `server --auth-mode=server --auth-mode=client --kube-api-burst=200  --kube-api-qps=200`

Before each tests following procedure were followed:
1. Delete all workflow
2. Wait for all workflows pods to be removed
3. Restart controller and server

Benchmarking tool: [hey](https://github.com/rakyll/hey). It runs command in parallel by default 200 times using 50 workers. Those values can be modified using:
* `-n`: number of requests
* `-c`: number of workers


Typical call:

```sh
hey \
    -n 200 -c 50 \
    -m POST \
    -disable-keepalive \
    -T "application/json" \
    -d '{
        "serverDryRun": false,
        "workflow": {
            "metadata": {
                "generateName": "curl-echo-test-",
                "namespace": "argo-test",
                "labels": {
                    "workflows.argoproj.io/benchmark": "true"
                }
            },
            "spec": {
                "workflowTemplateRef": {"name": "20-echos"},
                "arguments": {},
                "podMetadata": {
                    "labels": {
                        "workflows.argoproj.io/benchmark": "true"
                    }
                }
            }
        }
        }' \
    https://localhost:2746/api/v1/workflows/argo-test
```
