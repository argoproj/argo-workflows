# Sidecars

A sidecar is another container that executes concurrently in the same pod as the main container and is useful in creating multi-container pods.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: sidecar-nginx-
spec:
  entrypoint: sidecar-nginx-example
  templates:
  - name: sidecar-nginx-example
    container:
      image: appropriate/curl
      command: [sh, -c]
      # Try to read from nginx web server until it comes up
      args: ["until `curl -G 'http://127.0.0.1/' >& /tmp/out`; do echo sleep && sleep 1; done && cat /tmp/out"]
    # Create a simple nginx web server
    sidecars:
    - name: nginx
      image: nginx:1.13
      command: [nginx, -g, daemon off;]
```

In the above example, we create a sidecar container that runs Nginx as a simple web server. The order in which containers come up is random, so in this example the main container polls the Nginx container until it is ready to service requests. This is a good design pattern when designing multi-container systems: always wait for any services you need to come up before running your main code.
