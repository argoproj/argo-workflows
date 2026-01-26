Description: Add Optional Argo Workflow–Level Configuration for Executor Plugins
Author: [ntny](https://github.com/ntny)
Component: General
Issues: 15234

This PR allows configuring the Argo Workflow Executor Plugin for a specific Argo Workflow directly within the Workflow spec.

    metadata:
      name: http-template
      namespace: default
    spec:
      podSpecPatch:
        nodeName: virtual-node
      entrypoint: main
      executorPlugins: # executor plugin settings
         - spec:
            sidecar:
              container:
                name: test-sidecar
                image: busybox:1.35
                ports:
                  - containerPort: 8080
                resources:
                  requests:
                    cpu: "100m"
                    memory: "128Mi"
                  limits:
                    cpu: "200m"
                    memory: "256Mi"
                securityContext:
                  runAsUser: 1000
                  runAsGroup: 1000
                  runAsNonRoot: true