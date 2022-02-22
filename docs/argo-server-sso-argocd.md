# Use ArgoCD Dex for authentication

It is possible to have the Argo Workflows Server use the Argo CD Dex instance for authentication, for instance if you use Okta with SAML which cannot integrate with Argo Workflows directly. In order to make this happen, you will need the following:

- You must be using at least Dex [v2.23.0](https://github.com/dexidp/dex/releases/tag/v2.23.0), because that's when `staticClients[].secretEnv` was added. That means ArgoCD 1.7.12 and above.
- A secret containing two keys, `client-id` and `client-secret` to be used by both Dex and Argo Workflows Server. `client-id` is `argo-workflows-sso` in this example, `client-secret` can be any random string. If ArgoCD and ArgoWorkflows are installed in different namespaces the secret must be present in both of them. Example:
  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: argo-workflows-sso
  data:
    # client-id is 'argo-workflows-sso'
    client-id: YXJnby13b3JrZmxvd3Mtc3Nv
    # client-secret is 'MY-SECRET-STRING-CAN-BE-UUID'
    client-secret: TVktU0VDUkVULVNUUklORy1DQU4tQkUtVVVJRA==
  ```
- `--auth-mode=sso` server argument added
- A Dex `staticClients` configured for `argo-workflows-sso`
- The `sso` configuration filled out in Argo Workflows Server to match

## Example manifests for authenticating against ArgoCD's Dex (Kustomize)

In ArgoCD, add an environment variable to Dex deployment and configuration:
```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: argocd-dex-server
spec:
  template:
    spec:
      containers:
        - name: dex
          env:
            - name: ARGO_WORKFLOWS_SSO_CLIENT_SECRET
              valueFrom:
                secretKeyRef:
                  name: argo-workflows-sso
                  key: client-secret
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-cm
data:
  # Kustomize sees the value of dex.config as a single string instead of yaml. It will not merge
  # Dex settings, but instead it will replace the entire configuration with the settings below,
  # so add these to the existing config instead of setting them in a separate file
  dex.config: |
    # Setting staticClients allows ArgoWorkflows to use ArgoCD's Dex installation for authentication
    staticClients:
      - id: argo-workflows-sso
        name: Argo Workflow
        redirectURIs:
          - https://argo-workflows.mydomain.com/oauth2/callback
        secretEnv: ARGO_WORKFLOWS_SSO_CLIENT_SECRET
```

In Argo Workflows add `--auth-mode=sso` argument to argo-server deployment.
```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: argo-server
spec:
  template:
    spec:
      containers:
        - name: argo-server
          args:
            - server
            - --auth-mode=sso
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:
  # SSO Configuration for the Argo server.
  # You must also start argo server with `--auth-mode sso`.
  # https://argoproj.github.io/argo-workflows/argo-server-auth-mode/
  sso: |
    # This is the root URL of the OIDC provider (required).
    issuer: https://argo-cd.mydomain.com/api/dex
    # This is name of the secret and the key in it that contain OIDC client
    # ID issued to the application by the provider (required).
    clientId:
      name: argo-workflows-sso
      key: client-id
    # This is name of the secret and the key in it that contain OIDC client
    # secret issued to the application by the provider (required).
    clientSecret:
      name: argo-workflows-sso
      key: client-secret
    # This is the redirect URL supplied to the provider (required). It must
    # be in the form <argo-server-root-url>/oauth2/callback. It must be
    # browser-accessible.
    redirectUrl: https://argo-workflows.mydomain.com/oauth2/callback
```

## Example Helm chart configuration for authenticating against ArgoCD's Dex

`argo-cd/values.yaml`:
```yaml
     dex:
       image:
         tag: v2.23.0
       env:
         - name: ARGO_WORKFLOWS_SSO_CLIENT_SECRET
           valueFrom:
             secretKeyRef:
               name: argo-workflows-sso
               key: client-secret
     server:
       config:
         dex.config: |
           staticClients:
           - id: argo-workflows-sso
             name: Argo Workflow
             redirectURIs:
               - https://argo-workflows.mydomain.com/oauth2/callback
             secretEnv: ARGO_WORKFLOWS_SSO_CLIENT_SECRET
```

`argo/values.yaml`:
```yaml
     server:
       extraArgs:
         - --auth-mode=sso
       sso:
         issuer: https://argo-cd.mydomain.com/api/dex
         # sessionExpiry defines how long your login is valid for in hours. (optional, default: 10h)
         sessionExpiry: 240h
         clientId:
           name: argo-workflows-sso
           key: client-id
         clientSecret:
           name: argo-workflows-sso
           key: client-secret
         redirectUrl: https://argo-workflows.mydomain.com/oauth2/callback
```
