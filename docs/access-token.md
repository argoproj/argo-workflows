# Access Token

If you want to automate tasks with Argo Server, you need an access token. 

Firstly, create a role with minimal permissions. This example role for jenkins only permission to update and list workflows:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: jenkins-role
rules:
  - apiGroups:
      - ""
    resources:
      - workflows
    verbs:
      - list
      - update
```

Create a service account for your service:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: jenkins
```

Bind the service account to the role:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: jenkins
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: jenkins
subjects:
  - kind: ServiceAccount
    name: jenkins
```

Create a secret:

```yaml
kind: Secret
apiVersion: v1
metadata:
  name: jenkins
  annotations:
    kubernetes.io/service-account.name: jenkins
type: kubernetes.io/service-account-token
```

This secret will be automatically populated with a token under data/token ([learn more](https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/));

```shell script
ARGO_TOKEN=$(kubectl get secret jenkins -o yaml | grep -o 'token:.*' | sed 's/token: //')
echo $ARGO_TOKEN
ZXlKaGJHY2lPaUpTVXpJMU5pSXNJbXRwWkNJNkltS...
```

Use that token in your API requests, e.g. to list workflows:

```shell script
curl https://localhost:2746/api/v1/workflows/argo -H "Authorisation: Bearer $ARGO_TOKEN"
# 200 OK
```

You should check you cannot do things you're not allowed!

```shell script
curl https://localhost:2746/api/v1/workflow-templates/argo -H "Authorisation: Bearer $ARGO_TOKEN"
# 403 error
```

