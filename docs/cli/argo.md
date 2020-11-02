## argo

argo is the command line interface to Argo

### Synopsis

argo is the command line interface to Argo

```
argo [flags]
```

### Examples

```
If you're using the Argo Server (e.g. because you need large workflow support or workflow archive), please read https://github.com/argoproj/argo/blob/master/docs/cli.md.
```

### Options

```
  -s, --argo-server host:port          API server host:port. e.g. localhost:2746. Defaults to the ARGO_SERVER environment variable.
      --as string                      Username to impersonate for the operation
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --context string                 The name of the kubeconfig context to use
      --gloglevel int                  Set the glog logging level
  -h, --help                           help for argo
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
  -k, --insecure-skip-verify           If true, the Argo Server's certificate will not be checked for validity. This will make your HTTPS connections insecure. Defaults to the ARGO_INSECURE_SKIP_VERIFY environment variable.
      --instanceid string              submit with a specific controller's instance id label. Default to the ARGO_INSTANCEID environment variable.
      --kubeconfig string              Path to a kube config. Only required if out-of-cluster
      --loglevel string                Set the logging level. One of: debug|info|warn|error (default "info")
  -n, --namespace string               If present, the namespace scope for this CLI request
      --password string                Password for basic authentication to the API server
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
  -e, --secure                         Whether or not the server is using TLS with the Argo Server. Defaults to the ARGO_SECURE environment variable.
      --server string                  The address and port of the Kubernetes API server
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use
      --username string                Username for basic authentication to the API server
  -v, --verbose                        Enabled verbose logging, i.e. --loglevel debug
```

### SEE ALSO

* [argo archive](argo_archive.md)	 - 
* [argo auth](argo_auth.md)	 - 
* [argo cluster-template](argo_cluster-template.md)	 - manipulate cluster workflow templates
* [argo completion](argo_completion.md)	 - output shell completion code for the specified shell (bash or zsh)
* [argo cron](argo_cron.md)	 - manage cron workflows

NextScheduledRun assumes that the workflow-controller uses UTC as its timezone
* [argo delete](argo_delete.md)	 - delete workflows
* [argo get](argo_get.md)	 - display details about a workflow
* [argo lint](argo_lint.md)	 - validate files or directories of workflow manifests
* [argo list](argo_list.md)	 - list workflows
* [argo logs](argo_logs.md)	 - view logs of a pod or workflow
* [argo node](argo_node.md)	 - perform action on a node in a workflow
* [argo resubmit](argo_resubmit.md)	 - resubmit one or more workflows
* [argo resume](argo_resume.md)	 - resume zero or more workflows
* [argo retry](argo_retry.md)	 - retry zero or more workflows
* [argo server](argo_server.md)	 - Start the Argo Server
* [argo stop](argo_stop.md)	 - stop zero or more workflows
* [argo submit](argo_submit.md)	 - submit a workflow
* [argo suspend](argo_suspend.md)	 - suspend zero or more workflow
* [argo template](argo_template.md)	 - manipulate workflow templates
* [argo terminate](argo_terminate.md)	 - terminate zero or more workflows
* [argo version](argo_version.md)	 - Print version information
* [argo wait](argo_wait.md)	 - waits for workflows to complete
* [argo watch](argo_watch.md)	 - watch a workflow until it completes

