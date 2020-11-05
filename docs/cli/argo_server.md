## argo server

Start the Argo Server

### Synopsis

Start the Argo Server

```
argo server [flags]
```

### Examples

```

See https://argoproj.github.io/argo/argo-server.md
```

### Options

```
      --auth-mode stringArray            API server authentication mode. Any 1 or more length permutation of: client,server,sso (default [server])
      --basehref string                  Value for base href in index.html. Used if the server is running behind reverse proxy under subpath different from /. Defaults to the environment variable BASE_HREF. (default "/")
  -b, --browser                          enable automatic launching of the browser [local mode]
      --configmap string                 Name of K8s configmap to retrieve workflow controller configuration (default "workflow-controller-configmap")
      --event-operation-queue-size int   how many events operations that can be queued at once (default 16)
      --event-worker-count int           how many event workers to run (default 4)
  -h, --help                             help for server
      --hsts                             Whether or not we should add a HTTP Secure Transport Security header. This only has effect if secure is enabled. (default true)
      --managed-namespace string         namespace that watches, default to the installation namespace
      --namespaced                       run as namespaced mode
  -p, --port int                         Port to listen on (default 2746)
      --x-frame-options string           Set X-Frame-Options header in HTTP responses. (default "DENY")
```

### Options inherited from parent commands

```
      --argo-base-href string          An path to use with HTTP client (e.g. due to BASE_HREF). Defaults to the ARGO_BASE_HREF environment variable.
      --argo-http1                     If true, use the HTTP client. Defaults to the ARGO_HTTP1 environment variable.
  -s, --argo-server host:port          API server host:port. e.g. localhost:2746. Defaults to the ARGO_SERVER environment variable.
      --as string                      Username to impersonate for the operation
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --context string                 The name of the kubeconfig context to use
      --gloglevel int                  Set the glog logging level
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

