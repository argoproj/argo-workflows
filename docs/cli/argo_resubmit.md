## argo resubmit

resubmit one or more workflows

### Synopsis

resubmit one or more workflows

```
argo resubmit [WORKFLOW...] [flags]
```

### Examples

```
# Resubmit a workflow:

  argo resubmit my-wf

# Resubmit and wait for completion:

  argo resubmit --wait my-wf.yaml

# Resubmit and watch until completion:

  argo resubmit --watch my-wf.yaml

# Resubmit and tail logs until completion:

  argo resubmit --log my-wf.yaml

# Resubmit the latest workflow:

  argo resubmit @latest

```

### Options

```
  -h, --help             help for resubmit
      --log              log the workflow until it completes
      --memoized         re-use successful steps & outputs from the previous run (experimental)
  -o, --output string    Output format. One of: name|json|yaml|wide
      --priority int32   workflow priority
  -w, --wait             wait for the workflow to complete
      --watch            watch the workflow until it completes
```

### Options inherited from parent commands

```
  -s, --argo-server host:port          API server host:port. e.g. localhost:2746. Defaults to the ARGO_SERVER environment variable.
      --as string                      Username to impersonate for the operation
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --context string                 The name of the kubeconfig context to use
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

* [argo](argo.md)	 - argo is the command line interface to Argo

