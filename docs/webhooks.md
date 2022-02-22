# Webhooks


> v2.11 and after

Many clients can send events via the [events](events.md) API endpoint using a standard authorization header. However, for clients that are unable to do so (e.g. because they use signature verification as proof of origin), additional configuration is required.

In the namespace that will receive the event, create [access token](access-token.md) resources for your client:

* A role with permissions to get workflow templates and to create a workflow: [example](https://raw.githubusercontent.com/argoproj/argo-workflows/master/manifests/quick-start/base/webhooks/submit-workflow-template-role.yaml)
* A service account for the client: [example](https://raw.githubusercontent.com/argoproj/argo-workflows/master/manifests/quick-start/base/webhooks/github.com-sa.yaml). 
* A binding of the account to the role: [example](https://raw.githubusercontent.com/argoproj/argo-workflows/master/manifests/quick-start/base/webhooks/github.com-rolebinding.yaml)

Additionally create:

* A secret named "argo-workflows-webhook-clients" listing the service accounts: [example](https://raw.githubusercontent.com/argoproj/argo-workflows/master/manifests/quick-start/base/webhooks/argo-workflows-webhook-clients-secret.yaml)

The secret "argo-workflows-webhook-clients" tells Argo:

* What type of webhook the account can be used for, e.g. "github" 
* What "secret" that webhook is configured for, e.g. in your [Github settings page](https://github.com/alexec/argo/settings/hooks) 

