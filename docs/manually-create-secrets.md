# Kubernetes Secrets

As of Kubernetes v1.24, secrets are no longer automatically created for service accounts.

You must create a secret
manually: [Find out how to create these yourself manually](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#manually-create-a-service-account-api-token)
.

You must make the secret discoverable. You have two options:

## Option 1 - Discovery By Name

Name your secret `${serviceAccountName}.service-account-token`.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: default.service-account-token
  annotations:
    kubernetes.io/service-account.name: default
type: kubernetes.io/service-account-token
```

This option is simpler than option 2, as you can combine creating the secret with making it discoverable by name.

## Option 2 - Discovery By Annotation

Annotate the service account with the secret name:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: default
  annotations:
    workflows.argoproj.io/service-account-token.name: my-token
```

This option is useful when the secret already exists, or the service account has a very long name.
