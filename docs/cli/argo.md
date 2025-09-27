## argo

argo is the command line interface to Argo

### Synopsis


You can use the CLI in the following modes:

#### Kubernetes API Mode (default)

Requests are sent directly to the Kubernetes API. No Argo Server is needed. Large workflows and the workflow archive are not supported.

Use when you have direct access to the Kubernetes API, and don't need large workflow or workflow archive support.

If you're using instance ID (which is very unlikely), you'll need to set it:

	ARGO_INSTANCEID=your-instanceid

#### Argo Server GRPC Mode

Requests are sent to the Argo Server API via GRPC (using HTTP/2). Large workflows and the workflow archive are supported. Network load-balancers that do not support HTTP/2 are not supported.

Use if you do not have access to the Kubernetes API (e.g. you're in another cluster), and you're running the Argo Server using a network load-balancer that support HTTP/2.

To enable, set ARGO_SERVER:

	ARGO_SERVER=localhost:2746 ;# The format is "host:port" - do not prefix with "http" or "https"

If you're have transport-layer security (TLS) enabled (i.e. you are running "argo server --secure" and therefore has HTTPS):

	ARGO_SECURE=true

If your server is running with self-signed certificates. Do not use in production:

	ARGO_INSECURE_SKIP_VERIFY=true

By default, the CLI uses your KUBECONFIG to determine default for ARGO_TOKEN and ARGO_NAMESPACE. You probably error with "no configuration has been provided". To prevent it:

	KUBECONFIG=/dev/null

You will then need to set:

	ARGO_NAMESPACE=argo

And:

	ARGO_TOKEN='Bearer ******' ;# Should always start with "Bearer " or "Basic ".

#### Argo Server HTTP1 Mode

As per GRPC mode, but uses HTTP. Can be used with ALB that does not support HTTP/2. The command "argo logs --since-time=2020...." will not work (due to time-type).

Use this when your network load-balancer does not support HTTP/2.

Use the same configuration as GRPC mode, but also set:

	ARGO_HTTP1=true

If your server is behind an ingress with a path (running "argo server --base-href /argo" or "ARGO_BASE_HREF=/argo argo server"):

	ARGO_BASE_HREF=/argo


```
argo [flags]
```

### Options

```
      --argo-base-href string          Path to use with HTTP client due to Base HREF. Defaults to the ARGO_BASE_HREF environment variable.
      --argo-http1                     If true, use the HTTP client. Defaults to the ARGO_HTTP1 environment variable.
      --argo-root-path string          API path prefix when Argo Server is behind ingress/proxy. Defaults to the ARGO_ROOT_PATH environment variable.
  -s, --argo-server host:port          API server host:port. e.g. localhost:2746. Defaults to the ARGO_SERVER environment variable.
      --as string                      Username to impersonate for the operation
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --as-uid string                  UID to impersonate for the operation
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --context string                 The name of the kubeconfig context to use
      --disable-compression            If true, opt-out of response compression for all requests to the server
      --gloglevel int                  Set the glog logging level
  -H, --header strings                 Sets additional header to all requests made by Argo CLI. (Can be repeated multiple times to add multiple headers, also supports comma separated headers) Used only when either ARGO_HTTP1 or --argo-http1 is set to true.
  -h, --help                           help for argo
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
  -k, --insecure-skip-verify           If true, the Argo Server's certificate will not be checked for validity. This will make your HTTPS connections insecure. Defaults to the ARGO_INSECURE_SKIP_VERIFY environment variable.
      --instanceid string              submit with a specific controller's instance id label. Default to the ARGO_INSTANCEID environment variable.
      --kubeconfig string              Path to a kube config. Only required if out-of-cluster
      --log-format string              The formatter to use for logs. One of: text|json (default "text")
      --loglevel string                Set the logging level. One of: debug|info|warn|error (default "info")
  -n, --namespace string               If present, the namespace scope for this CLI request
      --password string                Password for basic authentication to the API server
      --proxy-url string               If provided, this URL will be used to connect via proxy
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
  -e, --secure                         Whether or not the server is using TLS with the Argo Server. Defaults to the ARGO_SECURE environment variable. (default true)
      --server string                  The address and port of the Kubernetes API server
      --tls-server-name string         If provided, this name will be used to validate server certificate. If this is not provided, hostname used to contact the server is used.
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use
      --username string                Username for basic authentication to the API server
  -v, --verbose                        Enabled verbose logging, i.e. --loglevel debug
```

### SEE ALSO

* [argo archive](argo_archive.md)	 - manage the workflow archive
* [argo auth](argo_auth.md)	 - manage authentication settings
* [argo cluster-template](argo_cluster-template.md)	 - manipulate cluster workflow templates
* [argo completion](argo_completion.md)	 - output shell completion code for the specified shell (bash, zsh or fish)
* [argo cp](argo_cp.md)	 - copy artifacts from workflow
* [argo cron](argo_cron.md)	 - manage cron workflows
* [argo delete](argo_delete.md)	 - delete workflows
* [argo executor-plugin](argo_executor-plugin.md)	 - manage executor plugins
* [argo get](argo_get.md)	 - display details about a workflow
* [argo lint](argo_lint.md)	 - validate files or directories of manifests
* [argo list](argo_list.md)	 - list workflows
* [argo logs](argo_logs.md)	 - view logs of a pod or workflow
* [argo node](argo_node.md)	 - perform action on a node in a workflow
* [argo resubmit](argo_resubmit.md)	 - resubmit one or more workflows
* [argo resume](argo_resume.md)	 - resume zero or more workflows (opposite of suspend)
* [argo retry](argo_retry.md)	 - retry zero or more workflows
* [argo server](argo_server.md)	 - start the Argo Server
* [argo stop](argo_stop.md)	 - stop zero or more workflows allowing all exit handlers to run
* [argo submit](argo_submit.md)	 - submit a workflow
* [argo suspend](argo_suspend.md)	 - suspend zero or more workflows (opposite of resume)
* [argo sync](argo_sync.md)	 - manage sync limits
* [argo template](argo_template.md)	 - manipulate workflow templates
* [argo terminate](argo_terminate.md)	 - terminate zero or more workflows immediately
* [argo version](argo_version.md)	 - print version information
* [argo wait](argo_wait.md)	 - waits for workflows to complete
* [argo watch](argo_watch.md)	 - watch a workflow until it completes

