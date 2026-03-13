# Webhooks

Argo Workflows supports event-driven workflow execution through webhooks. 
While many clients can send events via the [events](events.md) API endpoint using a standard authorization header, some clients—such as those relying on signature verification for authentication—require additional configuration.

## Configuring Webhook Access

To enable webhook-based event triggering, you need to configure authentication and authorization within the namespace that will receive the event. This involves setting up roles, service accounts, and secrets.

### 1. Create Access Token Resources

In the target namespace, define the necessary access token resources for your client:

- **Role with permissions** to fetch workflow templates and create workflows:  
  [View Example YAML](https://raw.githubusercontent.com/argoproj/argo-workflows/main/manifests/quick-start/base/webhooks/submit-workflow-template-role.yaml)

- **Service account for the client**:  
  [View Example YAML](https://raw.githubusercontent.com/argoproj/argo-workflows/main/manifests/quick-start/base/webhooks/github.com-sa.yaml)

- **Role binding to associate the service account with the role**:  
  [View Example YAML](https://raw.githubusercontent.com/argoproj/argo-workflows/main/manifests/quick-start/base/webhooks/github.com-rolebinding.yaml)

### 2. Define Webhook Clients in a Secret

Create a Kubernetes Secret named `argo-workflows-webhook-clients` that lists the Service Accounts authorized to trigger workflows via webhooks.

- [View Example YAML](https://raw.githubusercontent.com/argoproj/argo-workflows/main/manifests/quick-start/base/webhooks/argo-workflows-webhook-clients-secret.yaml)

This secret helps Argo Workflows determine:

| Parameter | Description |
|-----------|-------------|
| **Webhook Type** | Specifies the type of webhook, e.g., `github` for GitHub events. |
| **Webhook Secret** | Matches the secret configured in the webhook provider (e.g., GitHub settings). |

## Testing Webhooks

To validate your webhook setup, you can use external tools to inspect incoming requests and debug issues:

- **[Beeceptor](https://beeceptor.com/)** – Set up an endpoint to capture and inspect webhook payloads.
- **[Webhook.site](https://webhook.site/)** – Test and debug webhooks with a live request logger.
