## argo server

start the Argo Server

```
argo server [flags]
```

### Examples

```

See https://argo-workflows.readthedocs.io/en/latest/argo-server/
```

### Options

```
      --access-control-allow-origin string   Set Access-Control-Allow-Origin header in HTTP responses.
      --allowed-link-protocol stringArray    Allowed protocols for links feature. Defaults to the environment variable ALLOWED_LINK_PROTOCOL: http,https (default [http,https])
      --api-rate-limit uint                  Set limit per IP for api ratelimiter (default 1000)
      --auth-mode stringArray                API server authentication mode. Any 1 or more length permutation of: client,server,sso (default [client])
      --basehref string                      Value for base href in index.html. Used if the server is running behind reverse proxy under subpath different from /. Defaults to the environment variable BASE_HREF. (default "/")
  -b, --browser                              enable automatic launching of the browser [local mode]
      --configmap string                     Name of K8s configmap to retrieve workflow controller configuration (default "workflow-controller-configmap")
      --event-async-dispatch                 dispatch event async
      --event-operation-queue-size int       how many events operations that can be queued at once (default 16)
      --event-worker-count int               how many event workers to run (default 4)
  -h, --help                                 help for server
      --hsts                                 Whether or not we should add a HTTP Secure Transport Security header. This only has effect if secure is enabled. (default true)
      --kube-api-burst int                   Burst to use while talking with kube-apiserver. (default 30)
      --kube-api-qps float32                 QPS to use while talking with kube-apiserver. (default 20)
      --log-format string                    The formatter to use for logs. One of: text|json (default "text")
      --managed-namespace string             namespace that watches, default to the installation namespace
      --namespaced                           run as namespaced mode
  -p, --port int                             Port to listen on (default 2746)
  -e, --secure                               Whether or not we should listen on TLS. (default true)
      --tls-certificate-secret-name string   The name of a Kubernetes secret that contains the server certificates
      --x-frame-options string               Set X-Frame-Options header in HTTP responses. (default "DENY")
```

### Options inherited from parent commands

```
      --argo-base-href string          Path to use with HTTP client due to BASE_HREF. Defaults to the ARGO_BASE_HREF environment variable.
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
      --server string                  The address and port of the Kubernetes API server
      --tls-server-name string         If provided, this name will be used to validate server certificate. If this is not provided, hostname used to contact the server is used.
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use
      --username string                Username for basic authentication to the API server
  -v, --verbose                        Enabled verbose logging, i.e. --loglevel debug
```

### SEE ALSO

* [argo](argo.md)	 - argo is the command line interface to Argo

