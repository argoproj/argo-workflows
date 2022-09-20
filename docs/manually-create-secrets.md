# Kubernetes Secrets

As of Kubernetes v1.24, secrets are not automatically created for service accounts by default. [Find out how to create these yourself manually](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#manually-create-a-service-account-api-token).

Argo discovers your token by name, not annotation. They must be named `${serviceAccountName}.service-account-token`.
