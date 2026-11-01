# Dynamic Parameter Enums

> v4.2 and after

A parameter's `enum` gives users a fixed dropdown when they submit from the UI.
`enumFrom` fills that dropdown from live Kubernetes resources instead, for example the cluster's namespaces or the Deployments in one namespace.
The values are looked up when the submit form opens, so the list never goes stale and nobody has to maintain it by hand.

## Example

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: enum-from
spec:
  entrypoint: main
  arguments:
    parameters:
      - name: namespace
        value: default
        enumFrom:
          resource:
            apiVersion: v1
            kind: Namespace
          jsonPath: '{.metadata.name}'
      - name: deployment
        value: ''
        enumFrom:
          resource:
            apiVersion: apps/v1
            kind: Deployment
            namespace: '{{workflow.parameters.namespace}}'
          jsonPath: '{.metadata.name}'
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
        args: [echo, 'Selected {{workflow.parameters.namespace}}/{{workflow.parameters.deployment}}']
```

The full example is in [`examples/enum-from-workflow-template.yaml`](https://github.com/argoproj/argo-workflows/blob/main/examples/enum-from-workflow-template.yaml).

## Fields

| Field                 | Required | Description                                                                                                                                        |
|-----------------------|----------|----------------------------------------------------------------------------------------------------------------------------------------------------|
| `resource.apiVersion` | Yes      | API version of the resource to list, e.g. `v1` or `apps/v1`.                                                                                       |
| `resource.kind`       | Yes      | Kind of the resource to list, e.g. `Namespace` or `Deployment`. Custom resources work too.                                                         |
| `resource.namespace`  | No       | Namespace to list in. Defaults to the namespace the workflow is submitted to. Ignored for cluster-scoped kinds. May reference other parameters.    |
| `labelSelector`       | No       | Kubernetes label selector to filter the list, e.g. `tier=web`. May reference other parameters.                                                     |
| `jsonPath`            | Yes      | [`kubectl`-style JSONPath](https://kubernetes.io/docs/reference/kubectl/jsonpath/) applied to each item. Every value it returns becomes an option. |

Options are de-duplicated and sorted.
Items the JSONPath doesn't match are skipped.

A parameter can't have both `enum` and `enumFrom`.
`enumFrom` can't list Secrets: `argo lint` and submission reject it, and the Argo Server refuses it even if the template was created another way.

## Cascading Parameters

`resource.namespace` and `labelSelector` can reference other workflow parameters as `{{workflow.parameters.<name>}}`.
The UI re-resolves the dropdown whenever a referenced parameter changes.
Until the referenced parameter has a value, the dropdown shows a text box and a note saying which reference is still empty.

Only `{{workflow.parameters.<name>}}` references are substituted here.
Other variables, such as `{{workflow.namespace}}`, are not available when the form is shown.

## Where It Works

The UI uses `enumFrom` when you:

* submit a `WorkflowTemplate` or `ClusterWorkflowTemplate`, for both `spec.arguments.parameters` and the entrypoint template's input parameters
* resubmit or retry a `Workflow` whose own `spec.arguments.parameters` define `enumFrom`

Workflows submitted from a template reference the template instead of copying its `enumFrom`, so resubmitting or retrying them shows text boxes.

`enumFrom` only affects how parameters are asked for.
The controller ignores it when the workflow runs, and a plain `argo submit` or API call works as before.

If the values can't be resolved, for example because you lack permission or the kind doesn't exist, the UI shows the error and falls back to a text box, so you can still submit.

### CLI

`argo submit --interactive` asks for each workflow parameter in turn, and lists the options for `enum` and `enumFrom` parameters:

```console
$ argo submit -n argo --from workflowtemplate/enum-from --interactive

namespace
  1) argo
  2) default
  3) kube-system
  choose 1-3 [default]: 3

deployment
  1) coredns
  2) metrics-server
  choose 1-2: 1
Name:                enum-from-kzvkt
...
```

* Type an option's number or the option itself, or press Enter to keep the default shown in brackets.
* Parameters you pass with `-p` or `--parameter-file` are not asked for, and later lookups use their values.
* Parameters are asked for in the order they are declared, so declare a parameter before the ones that reference it.
* `--interactive` needs `--from`, because options are resolved from the stored `WorkflowTemplate`, `ClusterWorkflowTemplate` or `CronWorkflow`; a local file can't be used.
* If the options can't be resolved, the CLI shows why and asks you to type the value.
* Prompts are written to standard error, so `-o json` and `-o name` output stays clean.

This works through the Argo Server and with a direct Kubernetes connection.

## Load and Caching

To protect the Kubernetes API server on large clusters, the Argo Server:

* lists at most 500 resources per dropdown, and returns at most 500 options
* caches answers for 10 seconds, so re-opening a form or going back to a parameter makes no Kubernetes calls at all
* merges identical requests that arrive while one is still running
* gives up on a list after 15 seconds, and the UI falls back to a text box

If the list was cut off, the UI says so below the dropdown.
Use a `labelSelector` to narrow it down.

Answers, and the lists behind them, are cached per user: the key includes a hash of the caller's credentials and the namespace being submitted to.
Users never see each other's cached results.
The only thing shared between users is which resource types the cluster serves, which is cached for 5 minutes.
That information isn't user-specific, and every authenticated user can read it through API discovery anyway.
A lookup for a kind the cluster doesn't serve is never cached, so a newly installed CRD is found straight away.

Within one cache period, a change to someone's RBAC, to the template, or to the listed resources may not show yet.

You can change the limit and cache period with the [environment variables](environment-variables.md#argo-server) `ARGO_SERVER_RESOLVE_ENUM_MAX_ITEMS` and `ARGO_SERVER_RESOLVE_ENUM_CACHE_TTL` on the Argo Server.
Set `ARGO_SERVER_RESOLVE_ENUM_CACHE_TTL=0` to disable all caching.

## Security Model

Resolution happens on the Argo Server, which is the one component that sees both the template and the cluster.
The design keeps that from becoming a way to read things you otherwise couldn't:

1. **The lookup comes from the stored template, never from the browser.**
   The UI only sends which template and which parameter it wants.
   The server reads the `enumFrom` from the stored `WorkflowTemplate`, `ClusterWorkflowTemplate`, `CronWorkflow` or `Workflow` itself.
   So a user can only run lookups that someone allowed to write templates has defined.
2. **Both reads use your credentials.**
   The server reads the template and lists the resources with the same Kubernetes client it uses for every other request you make.
   In `client` and `sso` (with [SSO RBAC](argo-server-sso.md#sso-rbac)) [auth modes](argo-server-auth-mode.md), Kubernetes RBAC applies to you.
3. **Secrets are never listed**, whatever RBAC allows.
4. **The template must belong to this Argo instance.**
   Templates with a different [instance ID](scaling.md#instance-id) are refused, as for other API calls.

### RBAC

To see the options, a user needs:

* `get` on the template or workflow holding the parameter, which they already need to submit from it
* `list` on the resource named in `enumFrom`, in the namespace being listed
* API discovery, which every authenticated user has by default through the `system:discovery` cluster role

For example, to let users pick Deployments in the `apps` namespace:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: list-deployments
  namespace: apps
rules:
  - apiGroups: [apps]
    resources: [deployments]
    verbs: [list]
```

Listing namespaces needs a `ClusterRole`, because namespaces are cluster-scoped.

### `server` Auth Mode

!!! Warning
    In `server` auth mode, and in `sso` mode without SSO RBAC, the Argo Server uses its own service account for everyone.
    Every UI user then sees whatever that service account can list, through any `enumFrom` a template defines.

The JSONPath can return any field, not only names.
A template author could, for example, return the data of every ConfigMap the server can list.
In these modes, treat permission to create templates as permission to show their lookup results to every UI user, and keep the Argo Server's own RBAC narrow.

### Not an Allow List

`enumFrom` makes the right choice easy, but it doesn't enforce it.
The submitted value is not checked against the options: users can type any value into the fallback text box, and API and CLI submissions are not checked at all.
If a workflow must only act on certain resources, enforce that in the workflow's own service account RBAC.

## API

The UI calls this endpoint, which you can also use directly:

```bash
curl -X POST "$ARGO_SERVER/api/v1/workflows/argo/resolve-enum" \
  -H "Authorization: $ARGO_TOKEN" \
  -d '{
    "resourceKind": "WorkflowTemplate",
    "resourceName": "enum-from",
    "parameterName": "deployment",
    "parameters": {"namespace": "default"}
  }'
```

```json
{"values": ["my-app", "my-other-app"]}
```

`truncated` is `true` when more resources or options existed than the limit allows.

| Request field   | Description                                                                                         |
|-----------------|-----------------------------------------------------------------------------------------------------|
| `resourceKind`  | `WorkflowTemplate`, `ClusterWorkflowTemplate`, `CronWorkflow` or `Workflow`.                        |
| `resourceName`  | Name of that object. Namespaced kinds are read from the namespace in the URL.                       |
| `parameterName` | The parameter whose `enumFrom` to resolve.                                                          |
| `templateName`  | Optional. Look the parameter up in this template's `inputs.parameters` instead of `spec.arguments`. |
| `parameters`    | Current values of other parameters, substituted into `{{workflow.parameters.<name>}}` references.   |

Errors use the usual gRPC codes:

* `InvalidArgument` (HTTP 400) for a bad request, an unknown parameter, kind or API version, or an invalid JSONPath
* `FailedPrecondition` (HTTP 400) when a referenced parameter has no value yet
* `PermissionDenied` (HTTP 403) when RBAC forbids the list, or the `enumFrom` targets Secrets
* `NotFound` (HTTP 404) when the template or workflow doesn't exist
