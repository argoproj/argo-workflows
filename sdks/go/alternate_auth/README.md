# Alternate Authentication Example

This example demonstrates **alternate authentication patterns** for the Argo Workflows Go SDK.

## See Also

For basic authentication examples, see:
- **Kubeconfig Authentication**: See `../basic-workflow` - Standard kubectl-style authentication for local development
- **gRPC/Argo Server Authentication**: See `../grpc-client` - Remote Argo Server access

This example focuses on production and automation scenarios not covered in the basic examples.

## Supported Authentication Methods

1. **In-Cluster** - Service account authentication from within a pod
2. **Bearer Token** - Direct token authentication for automation

## Running the examples

### 1. In-Cluster Authentication

```bash
# This only works inside a Kubernetes pod
go run main.go -mode incluster
```

**Use when:**
- Running inside Kubernetes
- Using service accounts
- Building cluster-native applications

**Requirements:**
- Must run in a pod
- ServiceAccount with RBAC permissions

### 2. Bearer Token Authentication

```bash
# Export token
export KUBE_TOKEN=$(kubectl create token my-service-account)

# Run example
go run main.go -mode token
```

**Use when:**
- Automation and CI/CD
- Service-to-service authentication
- Don't have kubeconfig

**Get token:**
```bash
# Create token for service account
kubectl create token <service-account-name>

# Or get from secret
kubectl get secret <secret-name> -o jsonpath='{.data.token}' | base64 -d
```

## Expected Output

### In-Cluster Mode
```
=== Authentication Example ===
Mode: incluster

--- Authentication via In-Cluster Config ---
✓ Loaded in-cluster config
  API Server: https://kubernetes.default.svc
  Service Account: workflow-client
✓ Successfully authenticated and connected
  Found 3 workflow(s) in namespace 'default'

Usage:
  Best for: Applications running inside Kubernetes
  Requires: Pod with ServiceAccount having RBAC permissions
```

## Required RBAC Permissions

For in-cluster or token authentication, ensure your ServiceAccount has appropriate permissions:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: workflow-client
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: workflow-client-role
  namespace: default
rules:
- apiGroups: ["argoproj.io"]
  resources: ["workflows", "workflowtemplates", "cronworkflows"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: workflow-client-binding
  namespace: default
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: workflow-client-role
subjects:
- kind: ServiceAccount
  name: workflow-client
  namespace: default
```

## Choosing the Right Method

| Method | Use Case | Pros | Cons |
|--------|----------|------|------|
| **In-Cluster** | Apps in cluster | Native K8s, no external config | Only works in pods |
| **Token** | Automation, CI/CD | Simple, scriptable | Token management needed |
| **Kubeconfig** | Development, CLI tools | Easy, standard kubectl flow | See `../basic-workflow` |
| **gRPC** | Remote access | Works remotely, Argo features | See `../grpc-client` |

## Troubleshooting

### "Error loading in-cluster config"
- This only works inside a pod
- Verify ServiceAccount is mounted
- Check RBAC permissions

### "Unauthorized" errors
- Verify token is valid and not expired
- Check RBAC permissions for ServiceAccount
- Ensure namespace access is granted
