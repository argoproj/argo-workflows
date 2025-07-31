## argo archive resubmit

resubmit one or more workflows

```
argo archive resubmit [WORKFLOW...] [flags]
```

### Examples

```
# Resubmit a workflow:

  argo archive resubmit uid

# Resubmit multiple workflows:

  argo archive resubmit uid another-uid

# Resubmit multiple workflows by label selector:

  argo archive resubmit -l workflows.argoproj.io/test=true

# Resubmit multiple workflows by field selector:

  argo archive resubmit --field-selector metadata.namespace=argo

# Resubmit and wait for completion:

  argo archive resubmit --wait uid

# Resubmit and watch until completion:

  argo archive resubmit --watch uid

# Resubmit and tail logs until completion:

  argo archive resubmit --log uid

```

### Options

```
      --field-selector string   Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.
  -h, --help                    help for resubmit
      --log                     log the workflow until it completes
      --memoized                re-use successful steps & outputs from the previous run
  -o, --output string           Output format. One of: name|json|yaml|wide
  -p, --parameter stringArray   input parameter to override on the original workflow spec
      --priority int32          workflow priority
  -l, --selector string         Selector (label query) to filter on, not including uninitialized ones, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)
  -w, --wait                    wait for the workflow to complete, only works when a single workflow is resubmitted
      --watch                   watch the workflow until it completes, only works when a single workflow is resubmitted
```

### Options inherited from parent commands

```
      --argo-base-href string          Path to use with HTTP client due to Base HREF. Defaults to the ARGO_BASE_HREF environment variable.
      --argo-http1                     If true, use the HTTP client. Defaults to the ARGO_HTTP1 environment variable.
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
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
  -k, --insecure-skip-verify           If true, the Argo Server's certificate will not be checked for validity. This will make your HTTPS connections insecure. Defaults to the ARGO_INSECURE_SKIP_VERIFY environment variable.
      --instanceid string              submit with a specific controller's instance id label. Default to the ARGO_INSTANCEID environment variable.
      --kubeconfig string              Path to a kube config. Only required if out-of-cluster
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

