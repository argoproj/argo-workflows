## argo auth login

Log in to an Argo Server that uses SSO

### Synopsis

Log in to an Argo Server that uses SSO, and save the token for later commands.

By default, the CLI opens your browser at the Argo Server's SSO login page and listens on a loopback port for the result.
The Argo Server encrypts the token to a key that only this command holds.
The login must finish within 3 minutes, because the Argo Server's login state expires then.

With --paste, the browser login ends on the User Info page of the UI instead, and you paste the token from that page.
Use --paste when your browser runs on a different machine, for example over SSH, and add --no-browser to only print the login URL.

The CLI saves the token for the server given by --argo-server, --argo-base-href and --secure.
Later commands for that server use the saved token when ARGO_TOKEN isn't set.
"argo auth token" prints it, so other tools can use it too.

```
argo auth login [flags]
```

### Examples

```
# Log in with your browser
  argo auth login --argo-server argo.example.com:443

# Log in from a machine without a browser
  argo auth login --argo-server argo.example.com:443 --paste --no-browser

# Use the saved token with another tool
  curl -H "Authorization: $(argo auth token --argo-server argo.example.com:443)" https://argo.example.com/api/v1/userinfo
```

### Options

```
  -h, --help               help for login
      --no-browser         Print the login URL without opening a browser
      --paste              Paste the token from the UI instead of receiving it on a loopback port
      --timeout duration   How long to wait for the browser login to finish, at most 3m (default 3m0s)
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

* [argo auth](argo_auth.md)	 - manage authentication settings

