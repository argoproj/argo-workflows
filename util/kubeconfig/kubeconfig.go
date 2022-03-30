package kubeconfig

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	clientauthenticationapi "k8s.io/client-go/pkg/apis/clientauthentication"
	"k8s.io/client-go/plugin/pkg/client/auth/exec"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport"
)

const (
	BasicAuthScheme  = "Basic"
	BearerAuthScheme = "Bearer"
)

func DefaultRestConfig() (*restclient.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{})
	return config.ClientConfig()
}

func isBasicAuthScheme(token string) bool {
	return strings.HasPrefix(token, BasicAuthScheme)
}

func isBearerAuthScheme(token string) bool {
	return strings.HasPrefix(token, BearerAuthScheme)
}

func GetRestConfig(c *restclient.Config, token string) (*restclient.Config, error) {
	if isBasicAuthScheme(token) {
		token = strings.TrimSpace(strings.TrimPrefix(token, BasicAuthScheme))
		username, password, ok := decodeBasicAuthToken(token)
		if !ok {
			return nil, errors.New("Error parsing Basic Authentication")
		}
		return getBasicRestConfig(c, username, password), nil
	}
	if isBearerAuthScheme(token) {
		token = strings.TrimSpace(strings.TrimPrefix(token, BearerAuthScheme))
		return getBearerRestConfig(c, token), nil
	}
	return nil, errors.New("Unsupported authentication scheme")
}

// convert a basic token (username, password) into a REST config
func getBasicRestConfig(c *restclient.Config, username, password string) *restclient.Config {
	restConfig := restConfigWithoutAuth(c)
	restConfig.Username = username
	restConfig.Password = password
	return restConfig
}

// convert a bearer token into a REST config
func getBearerRestConfig(c *restclient.Config, token string) *restclient.Config {
	restConfig := restConfigWithoutAuth(c)
	restConfig.BearerToken = token
	return restConfig
}

// populate everything except
// - username
// - password
// - bearerToken
// - client private key
func restConfigWithoutAuth(c *restclient.Config) *restclient.Config {
	t := c.TLSClientConfig
	return &restclient.Config{
		Host:          c.Host,
		APIPath:       c.APIPath,
		ContentConfig: c.ContentConfig,
		TLSClientConfig: restclient.TLSClientConfig{
			Insecure:   t.Insecure,
			ServerName: t.ServerName,
			CertFile:   t.CertFile,
			CAFile:     t.CAFile,
			CertData:   t.CertData,
			CAData:     t.CAData,
			NextProtos: c.NextProtos,
		},
		UserAgent:          c.UserAgent,
		DisableCompression: c.DisableCompression,
		Transport:          c.Transport,
		WrapTransport:      c.WrapTransport,
		QPS:                c.QPS,
		Burst:              c.Burst,
		RateLimiter:        c.RateLimiter,
		WarningHandler:     c.WarningHandler,
		Timeout:            c.Timeout,
		Dial:               c.Dial,
		Proxy:              c.Proxy,
	}
}

// Return the AuthString include Auth type(Basic or Bearer)
func GetAuthString(in *restclient.Config, explicitKubeConfigPath string) (string, error) {
	// Checking Basic Auth
	if in.Username != "" {
		token, err := getBasicAuthToken(in)
		return BasicAuthScheme + " " + token, err
	}

	token, err := getBearerToken(in, explicitKubeConfigPath)
	return BearerAuthScheme + " " + token, err
}

func getBasicAuthToken(in *restclient.Config) (string, error) {
	if in == nil {
		return "", fmt.Errorf("RestClient can't be nil")
	}

	return encodeBasicAuthToken(in.Username, in.Password), nil
}

// convert the REST config into a bearer token
func getBearerToken(in *restclient.Config, explicitKubeConfigPath string) (string, error) {
	if len(in.BearerToken) > 0 {
		return in.BearerToken, nil
	}

	if in == nil {
		return "", fmt.Errorf("RestClient can't be nil")
	}
	if in.ExecProvider != nil {
		tc, err := in.TransportConfig()
		if err != nil {
			return "", err
		}

		var cluster *clientauthenticationapi.Cluster
		if in.ExecProvider.ProvideClusterInfo {
			var err error
			cluster, err = configToExecCluster(in)
			if err != nil {
				return "", err
			}
		}
		auth, err := exec.GetAuthenticator(in.ExecProvider, cluster)
		if err != nil {
			return "", err
		}

		// This function will return error because of TLS Cert missing,
		// This code is not making actual request. We can ignore it.
		_ = auth.UpdateTransportConfig(tc)

		tp, err := transport.New(tc)
		if err != nil {
			return "", err
		}
		req, err := http.NewRequest("GET", in.Host, nil)
		if err != nil {
			return "", err
		}
		resp, err := tc.WrapTransport(tp).RoundTrip(req)
		if err != nil {
			return "", err
		}
		if err := resp.Body.Close(); err != nil {
			return "", err
		}

		token := req.Header.Get("Authorization")
		return strings.TrimPrefix(token, "Bearer "), nil
	}
	if in.AuthProvider != nil {
		if in.AuthProvider.Name == "gcp" {
			token := in.AuthProvider.Config["access-token"]
			token, err := refreshTokenIfExpired(in, explicitKubeConfigPath, token)
			if err != nil {
				return "", err
			}
			return strings.TrimPrefix(token, "Bearer "), nil
		}
	}
	return "", fmt.Errorf("could not find a token")
}

/*https://pkg.go.dev/k8s.io/client-go@v0.20.4/pkg/apis/clientauthentication#Cluster
I am following this example: https://github.com/kubernetes/client-go/blob/v0.20.4/rest/transport.go#L99 and https://github.com/kubernetes/client-go/blob/v0.20.4/rest/exec.go */

// configToExecCluster creates a clientauthentication.Cluster with the corresponding fields from the provided Config
func configToExecCluster(config *restclient.Config) (*clientauthenticationapi.Cluster, error) {
	caData, err := dataFromSliceOrFile(config.CAData, config.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA bundle for execProvider: %v", err)
	}

	var proxyURL string
	if config.Proxy != nil {
		req, err := http.NewRequest("", config.Host, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create proxy URL request for execProvider: %w", err)
		}
		url, err := config.Proxy(req)
		if err != nil {
			return nil, fmt.Errorf("failed to get proxy URL for execProvider: %w", err)
		}
		if url != nil {
			proxyURL = url.String()
		}
	}

	return &clientauthenticationapi.Cluster{
		Server:                   config.Host,
		TLSServerName:            config.ServerName,
		InsecureSkipTLSVerify:    config.Insecure,
		CertificateAuthorityData: caData,
		ProxyURL:                 proxyURL,
		Config:                   config.ExecProvider.Config,
	}, nil
}

// dataFromSliceOrFile returns data from the slice (if non-empty), or from the file,
// or an error if an error occurred reading the file
func dataFromSliceOrFile(data []byte, file string) ([]byte, error) {
	if len(data) > 0 {
		return data, nil
	}

	if len(file) > 0 {
		fileData, err := ioutil.ReadFile(filepath.Clean(file))
		if err != nil {
			return []byte{}, err
		}
		return fileData, nil
	}
	return nil, nil
}

func encodeBasicAuthToken(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func decodeBasicAuthToken(auth string) (username, password string, ok bool) {
	c, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}

func reloadKubeConfig(explicitPath string) clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	loadingRules.ExplicitPath = explicitPath
	overrides := clientcmd.ConfigOverrides{}
	return clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, &overrides, os.Stdin)
}

func refreshTokenIfExpired(restConfig *restclient.Config, explicitPath, curentToken string) (string, error) {
	if restConfig.AuthProvider != nil {
		timestr := restConfig.AuthProvider.Config["expiry"]
		if timestr != "" {
			t, err := time.Parse(time.RFC3339, timestr)
			if err != nil {
				return "", fmt.Errorf("Invalid expiry date in Kubeconfig. %v", err)
			}
			if time.Now().After(t) {
				err = refreshAuthToken(restConfig)
				if err != nil {
					return "", err
				}
				config := reloadKubeConfig(explicitPath)
				restConfig, err = config.ClientConfig()
				if err != nil {
					return "", err
				}
				return restConfig.AuthProvider.Config["access-token"], nil
			}
		}
	}
	return curentToken, nil
}

func refreshAuthToken(in *restclient.Config) error {
	tc, err := in.TransportConfig()
	if err != nil {
		return err
	}

	auth, err := restclient.GetAuthProvider(in.Host, in.AuthProvider, in.AuthConfigPersister)
	if err != nil {
		return err
	}

	rt, err := transport.New(tc)
	if err != nil {
		return err
	}
	rt = auth.WrapTransport(rt)
	req := http.Request{Header: map[string][]string{}}

	resp, err := rt.RoundTrip(&req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}
