package kubeconfig

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
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

// get the default one from the filesystem
func DefaultRestConfig() (*restclient.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{})
	return config.ClientConfig()
}

func IsBasicAuthScheme(token string) bool {
	return strings.HasPrefix(token, BasicAuthScheme)
}

func IsBearerAuthScheme(token string) bool {
	return strings.HasPrefix(token, BearerAuthScheme)
}

func GetRestConfig(token string) (*restclient.Config, error) {
	if IsBasicAuthScheme(token) {
		token = strings.TrimSpace(strings.TrimPrefix(token, BasicAuthScheme))
		username, password, ok := decodeBasicAuthToken(token)
		if !ok {
			return nil, errors.New("error parsing basic authentication")
		}
		return GetBasicRestConfig(username, password)
	}
	if IsBearerAuthScheme(token) {
		token = strings.TrimSpace(strings.TrimPrefix(token, BearerAuthScheme))
		return GetBearerRestConfig(token)
	}
	return nil, errors.New("unsupported authentication scheme")
}

// convert a basic token (username, password) into a REST config
func GetBasicRestConfig(username, password string) (*restclient.Config, error) {
	restConfig, err := restConfigWithoutAuth()
	if err != nil {
		return nil, err
	}
	restConfig.Username = username
	restConfig.Password = password
	return restConfig, nil
}

// convert a bearer token into a REST config
func GetBearerRestConfig(token string) (*restclient.Config, error) {
	restConfig, err := restConfigWithoutAuth()
	if err != nil {
		return nil, err
	}
	restConfig.BearerToken = token
	return restConfig, nil
}

// populate everything except
// - username
// - password
// - bearerToken
// - client private key
func restConfigWithoutAuth() (*restclient.Config, error) {
	c, err := DefaultRestConfig()
	if err != nil {
		return nil, err
	}
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
	}, nil
}

// Return the AuthString include Auth type(Basic or Bearer)
func GetAuthString(ctx context.Context, in *restclient.Config, explicitKubeConfigPath string) (string, error) {
	// Checking Basic Auth
	if in.Username != "" {
		token, err := GetBasicAuthToken(in)
		return BasicAuthScheme + " " + token, err
	}

	token, err := GetBearerToken(ctx, in, explicitKubeConfigPath)
	return BearerAuthScheme + " " + token, err
}

func GetBasicAuthToken(in *restclient.Config) (string, error) {
	if in == nil {
		return "", fmt.Errorf("RestClient can't be nil")
	}

	return encodeBasicAuthToken(in.Username, in.Password), nil
}

// convert the REST config into a bearer token
func GetBearerToken(ctx context.Context, in *restclient.Config, explicitKubeConfigPath string) (string, error) {
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
			var clusterErr error
			cluster, clusterErr = ConfigToExecCluster(ctx, in)
			if clusterErr != nil {
				return "", clusterErr
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
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, in.Host, nil)
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
			token, err := RefreshTokenIfExpired(in, explicitKubeConfigPath, token)
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

// ConfigToExecCluster creates a clientauthentication.Cluster with the corresponding fields from the provided Config
func ConfigToExecCluster(ctx context.Context, config *restclient.Config) (*clientauthenticationapi.Cluster, error) {
	caData, err := dataFromSliceOrFile(config.CAData, config.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA bundle for execProvider: %w", err)
	}

	var proxyURL string
	if config.Proxy != nil {
		req, err := http.NewRequestWithContext(ctx, "", config.Host, nil)
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
		fileData, err := os.ReadFile(filepath.Clean(file))
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
	before, after, ok0 := strings.Cut(cs, ":")
	if !ok0 {
		return
	}
	return before, after, true
}

func ReloadKubeConfig(explicitPath string) clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	loadingRules.ExplicitPath = explicitPath
	overrides := clientcmd.ConfigOverrides{}
	return clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, &overrides, os.Stdin)
}

func RefreshTokenIfExpired(restConfig *restclient.Config, explicitPath, curentToken string) (string, error) {
	if restConfig.AuthProvider != nil {
		timestr := restConfig.AuthProvider.Config["expiry"]
		if timestr != "" {
			t, err := time.Parse(time.RFC3339, timestr)
			if err != nil {
				return "", fmt.Errorf("invalid expiry date in Kubeconfig. %w", err)
			}
			if time.Now().After(t) {
				err = RefreshAuthToken(restConfig)
				if err != nil {
					return "", err
				}
				config := ReloadKubeConfig(explicitPath)
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

func RefreshAuthToken(in *restclient.Config) error {
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
