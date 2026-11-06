# Argo Workflows threat model

Argo Workflows is a Kubernetes-native workflow engine. Users submit `Workflow` resources, and the
workflow controller turns each step into a pod in the workflow's namespace. One controller and one
argo-server usually serve many namespaces and teams, and both hold broad Kubernetes credentials. A flaw
can therefore let one tenant reach another tenant's namespace, data or credentials, or take down
workflows for the whole cluster.

Project security policy: <https://github.com/argoproj/argo-workflows/blob/main/SECURITY.md>
Operator security notes: `docs/security.md`, `docs/workflow-restrictions.md`, `docs/argo-server-auth-mode.md`,
`docs/argo-server-sso.md`, `docs/workflow-rbac.md`, `docs/managed-namespace.md`

## Components and where untrusted input enters

- **argo-server** (`server/`, started by `argo server` in `cmd/argo`): gRPC and the gRPC-gateway REST API
  under `/api/`, the embedded UI, artifact download and upload endpoints (`server/artifacts`), SSO
  login, callback and logout, and the event and webhook endpoint `/api/v1/events/` (`server/event`,
  `server/auth/webhook`). This is the main network-facing component. Treat every request, header,
  cookie, bearer token, OIDC response and webhook payload as attacker-controlled.
- **argo-server authentication and authorization** (`server/auth`, `server/auth/sso`,
  `server/auth/serviceaccount`). The mode is chosen per request:
    - `client`: the caller's own Kubernetes token, so the caller's Kubernetes RBAC applies.
    - `sso`: argo-server issues its own token after OIDC login. SSO RBAC maps users to service accounts
      with expressions on service account annotations, optionally delegated per namespace.
    - `server`: argo-server's own service account.

  Most reads go to the Kubernetes API with the caller's credentials. Some do not: reads from informers,
  from the workflow archive database, and from ConfigMaps (for example `server/sync`) use argo-server's
  own credentials. Those code paths must check the caller's access themselves, through the authorizer in
  `server/auth`. A check there that is missing, wrong or possible to bypass is the most
  common class of past advisory.
- **workflow-controller** (`workflow/controller`, `workflow/sync`, `workflow/cron`,
  `workflow/templateresolution`, `workflow/validate`, `workflow/artifactrepositories`): one process
  shared by every namespace it manages. It creates pods, PVCs, ConfigMaps and secrets on behalf of
  users and reads secrets and ConfigMaps. Workflow specs, parameters, pod annotations and task results
  written from workflow pods are attacker-controlled. Workflow authors' expressions are evaluated
  inside this process (`util/expr`, `util/template`, and `govaluate` in DAG and cron conditions), so
  they must not reach controller memory, environment, files or credentials, or other tenants' data.
- **Workflow restrictions** (`templateReferencing: Strict` or `Secure`, `docs/workflow-restrictions.md`):
  when enabled, users may only run workflows from existing `WorkflowTemplate` and
  `ClusterWorkflowTemplate` resources. Any field that lets such a user change the resulting pods
  (pod spec patches, security context, host networking, service accounts, artifact GC pods and so on)
  is a bypass.
- **argoexec** (`cmd/argoexec`, `workflow/executor`): runs in every workflow pod as the init, wait,
  supervisor or emissary process, and as the agent pod for HTTP templates and executor plugins. It
  downloads, unpacks and uploads artifacts and reports outputs through `WorkflowTaskResult` resources.
  The wait container holds a service account token that can patch the pod and write task results.
- **Artifact drivers** (`workflow/artifacts/{s3,gcs,azure,oss,hdfs,git,http,raw,resource,plugin}`):
  object keys, archive entries (tar, zip, symlinks), git repositories and HTTP responses are untrusted.
  Artifact repository credentials come from Kubernetes secrets and must never be written to logs,
  events, workflow status or error messages.
- **Workflow archive and offloading** (`persist/sqldb`, `util/sqldb`, `server/workflowarchive`,
  `workflow/hydrator`): Postgres or MySQL. User-supplied list options, label and field selectors and
  names reach SQL queries. Results must be filtered to the namespaces the caller can read.
- **Events** (`server/event`, `WorkflowEventBinding`): event payloads are evaluated with expressions
  and used as submission parameters. They must not be re-evaluated as templates or expressions.
- **Memoization and synchronization** (`workflow/controller/cache`, `workflow/sync`): cache ConfigMaps
  and ConfigMap- or database-backed semaphores and mutexes. A workflow must not read or poison another
  namespace's cache, or hold another namespace's locks.
- **Plugins** (`workflow/artifacts/plugin`, executor plugins in the agent pod): operator-installed code.
  The plugin code itself is trusted. Argo's handling of the plugin token, the sidecar wiring and plugin
  responses is not. For example, a plugin response must not be able to change state for a different
  workflow.
- **UI** (`ui/`, React and TypeScript): renders workflow specs, status messages, logs, artifact listings
  and artifact contents that come from workflow authors and workflow pods. Stored or reflected XSS
  through that data is in scope. Artifacts are served with a sandboxing CSP
  (`docs/artifact-visualization.md`), and getting script to run in the UI's origin is a finding.
- **CLI** (`cmd/argo`): talks to argo-server or the Kubernetes API from users' machines. A malicious or
  compromised argo-server is a valid attacker for CLI bugs, for example local file writes from
  `argo cp`.

## Attackers to consider

- Unauthenticated network clients of argo-server, including webhook senders.
- Holders of an invalid, expired or forged token in `client` or `sso` mode.
- Authenticated users with limited permissions: read-only users, users who can only use one namespace,
  and users who may only submit from templates under workflow restrictions. Any way for them to read
  or change workflows, templates, archived workflows, logs or artifacts outside their permissions is in
  scope.
- Workflow authors in one namespace. Inside their own namespace they can already run arbitrary pods
  (see below), so treat them as attackers against everything else: other namespaces, the controller,
  argo-server, other tenants' artifacts, caches and locks, and workflow restrictions.
- Anyone who controls an artifact's contents or location that the workflow author did not choose:
  another tenant's outputs, a shared bucket, an upstream git or HTTP server.
- Responders to outbound requests made by the controller or argo-server (OIDC providers, artifact
  stores, HTTP endpoints the controller is made to call).

## Trusted / out of scope

- **Arbitrary pods by design.** Argo Workflows is a pod controller, like a Kubernetes Job. Anyone who
  can create a `Workflow` in a namespace can run any container, with any service account and pod spec,
  in that namespace. That is not a finding unless it bypasses workflow restrictions.
- A workflow author attacking their own pod. For example, path traversal while unpacking an artifact the
  author created and controls is out of scope. The same bug is in scope when someone else controls the
  artifact.
- Network access from inside workflow pods or the agent pod, including HTTP templates reaching cloud
  metadata or the cluster network. Workflow authors can already do that from a container. Making the
  controller or argo-server send requests (SSRF) is in scope.
- Administrators, and anyone who can change `workflow-controller-configmap`, the controller's
  namespace, the controller's or argo-server's service accounts, or the archive database.
- Documented insecure configurations:
    - `--auth-mode=server` exposed to users who should not have argo-server's service account.
    - SSO without SSO RBAC, which maps every SSO user to argo-server's service account.
    - Letting users create workflows or pods in the controller's namespace.
    - Granting users cluster-wide rights to create workflows or pods.

  Report issues in these setups only if they bypass a documented restriction.
- Executor and artifact plugin code, as above.
- Vulnerabilities in dependencies (Go modules, npm packages, container images) that are not reachable
  through Argo Workflows code. Dependency CVEs reported only from a version number are out of scope.
- `test/`, `hack/`, `docs/`, `examples/`, `dev/` and `sdks/`.

## Known weak areas

Many past advisories were incomplete fixes, so look for siblings of these:

- Authorization checks on reads that use argo-server's own credentials: informer-backed workflow and
  template reads, archived workflow list and get (including name and field selectors), sync ConfigMaps,
  and info endpoints in `client` mode.
- Workflow restriction bypasses through any field that ends up in a pod, including `podSpecPatch` and
  pods the controller creates outside the main template, such as artifact GC pods.
- Path traversal and symlink escapes when unpacking artifacts or mapping object keys to local paths in
  `workflow/executor` and `workflow/artifacts`.
- Artifact repository credentials written to controller or executor logs.
- XSS in artifact listings and artifact contents served by argo-server.
- Controller crashes or stalls caused by malformed pod annotations, task results or workflow specs, and
  argo-server reading unbounded request bodies before authentication.

## How to exercise it

- Unit tests sit next to the code. Run them with `KUBECONFIG=/dev/null`, which the image already sets.
  Single test: `go test -run TestFoo ./server/workflow/`.
- Many argo-server tests build services with fake Kubernetes clients; `server/workflow/workflow_server_test.go`
  and `server/auth/gatekeeper_test.go` show how. Controller tests in `workflow/controller` run the
  operator against fake clients.
- The `argo`, `workflow-controller` and `argoexec` binaries are on `PATH`, built from `/src` with the UI
  embedded in `argo`. `argo lint --offline` checks workflow YAML files without a cluster.
- The UI's dependencies are installed in `ui/node_modules`, and `yarn --cwd ui test` runs its tests.
- There is no Kubernetes cluster, Docker daemon or database in the image. Prefer reproducers as Go unit
  tests. Tests that start Postgres or MySQL with testcontainers (in `persist/sqldb`, `util/sqldb` and
  `workflow/sync`) fail there. That is expected and not a finding. The scan has no network access, so
  tests that fetch remote content also fail.

## How we rate severity

We use CVSS 3.1 and publish advisories through GitHub Security Advisories. As guidance:

- **Critical**: unauthenticated remote code execution; authentication bypass on argo-server; reading
  workflows, templates, secrets or credentials with no valid token; workflow author expressions running
  code in the controller or reading its credentials.
- **High**:
    - An authenticated low-privilege user reading or changing workflows, templates, archived workflows,
      logs or artifacts in namespaces they cannot access.
    - A workflow author reaching another namespace, the controller's or argo-server's credentials, or
      another tenant's artifacts, caches or locks.
    - Any workflow restriction bypass.
    - A namespace tenant crash-looping or stalling the shared controller, or an unauthenticated client
      taking down argo-server.
    - Code execution or file writes outside the workspace in argoexec from artifact contents or keys the
      workflow author does not control.
    - Credentials written to logs, events or workflow status.
    - Stored XSS in the UI.
- **Medium**: information disclosure without a valid token that does not reveal workflow contents, such
  as version details or enumerating archived workflow names; open redirects; CSRF on state-changing
  endpoints; other denial of service by an authenticated user.
- **Low**: issues that need administrator-controlled configuration, or an insecure configuration that
  the docs warn against.

## How we would like reports

Keep reports short. Please include the affected component, the attacker's required access (none,
invalid token, read-only user, namespace user, workflow author, template-only user under workflow
restrictions), the auth mode, and a reproducer, ideally as a Go unit test. Please validate impact
before reporting, and don't report theoretical issues without a concrete path from attacker input to
impact.
