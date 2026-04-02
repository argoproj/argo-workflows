# Node Field Selectors

> v2.8 and after

## Introduction

The resume, stop and retry Argo CLI and API commands support a `--node-field-selector` parameter to allow the user to select a subset of nodes for the command to apply to.

In the case of the resume and stop commands these are the nodes that should be resumed or stopped.

In the case of the retry command it allows specifying nodes that should be restarted even if they were previously successful (and must be used in combination with `--restart-successful`)

The format of this when used with the CLI is:

```bash
--node-field-selector=FIELD=VALUE
```

## Possible options

The field can be any of:

| Field | Description|
|----------|------------|
| `displayName`| Display name of the node. This is the name of the node as it is displayed on the CLI or UI, without considering its ancestors (see example below). This is a useful shortcut if there is only one node with the same `displayName` |
| `id` | ID of the node, a unique identifier for the node which can be discovered by reading the status of the workflow. |
| `name`| Full name of the node. This is the full name of the node, including its ancestors (see example below). Using `name` is necessary when two or more nodes share the same `displayName` and disambiguation is required. |
| `templateName`| Template name of the node |
| `phase`| Phase status of the node - e.g. Running |
| `templateRef.name`| The name of the workflow template the node is referring to |
| `templateRef.template`| The template within the workflow template the node is referring to |
| `inputs.parameters.<NAME>.value`| The value of input parameter NAME |

The operator can be '=' or '!='. Multiple selectors can be combined with a comma, in which case they are anded together.

## Examples

To filter for nodes where the input parameter 'foo' is equal to 'bar':

```bash
--node-field-selector=inputs.parameters.foo.value=bar
```

To filter for nodes where the input parameter 'foo' is equal to 'bar' and phase is not running:

```bash
--node-field-selector=foo1=bar1,phase!=Running
```

Consider the following workflow:

```text
 ● appr-promotion-ffsv4    code-release
 ├─✔ start                 sample-template/email                 appr-promotion-ffsv4-3704914002  2s
 ├─● app1                  wftempl1/approval-and-promotion
 │ ├─✔ notification-email  sample-template/email                 appr-promotion-ffsv4-524476380   2s
 │ └─ǁ wait-approval       sample-template/waiting-for-approval
 ├─✔ app2                  wftempl2/promotion
 │ ├─✔ notification-email  sample-template/email                 appr-promotion-ffsv4-2580536603  2s
 │ ├─✔ pr-approval         sample-template/approval              appr-promotion-ffsv4-3445567645  2s
 │ └─✔ deployment          sample-template/promote               appr-promotion-ffsv4-970728982   1s
 └─● app3                  wftempl1/approval-and-promotion
   ├─✔ notification-email  sample-template/email                 appr-promotion-ffsv4-388318034   2s
   └─ǁ wait-approval       sample-template/waiting-for-approval
```

Here we have two steps with the same `displayName`: `wait-approval`. To select one to suspend, we need to use their
`name`, either `appr-promotion-ffsv4.app1.wait-approval` or `appr-promotion-ffsv4.app3.wait-approval`. If it is not clear
what the full name of a node is, it can be found using `kubectl`:

```bash
$ kubectl get wf appr-promotion-ffsv4 -o yaml

...
    appr-promotion-ffsv4-3235686597:
      boundaryID: appr-promotion-ffsv4-3079407832
      displayName: wait-approval                        # <- Display Name
      finishedAt: null
      id: appr-promotion-ffsv4-3235686597
      name: appr-promotion-ffsv4.app1.wait-approval     # <- Full Name
      phase: Running
      startedAt: "2021-01-20T17:00:25Z"
      templateRef:
        name: sample-template
        template: waiting-for-approval
      templateScope: namespaced/wftempl1
      type: Suspend
...
```
