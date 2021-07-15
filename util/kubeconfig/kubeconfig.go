package kubeconfig

import (
	"encoding/base64"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
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
			return nil, errors.New("Error parsing Basic Authentication")
		}
		return GetBasicRestConfig(username, password)
	}
	if IsBearerAuthScheme(token) {
		token = strings.TrimSpace(strings.TrimPrefix(token, BearerAuthScheme))
		return GetBearerRestConfig(token)
	}
	return nil, errors.New("Unsupported authentication scheme")
}

// convert a basic token (username, password) into a REST config
func GetBasicRestConfig(username, password string) (*restclient.Config, error) {
	restConfig, err := DefaultRestConfig()
	if err != nil {
		return nil, err
	}
	restConfig.BearerToken = ""
	restConfig.BearerTokenFile = ""
	restConfig.Username = username
	restConfig.Password = password
	return restConfig, nil
}

// convert a bearer token into a REST config
func GetBearerRestConfig(token string) (*restclient.Config, error) {
	restConfig, err := DefaultRestConfig()
	if err != nil {
		return nil, err
	}
	restConfig.BearerToken = ""
	restConfig.BearerTokenFile = ""
	restConfig.Username = ""
	restConfig.Password = ""
	if token != "" {
		restConfig.BearerToken = token
	}
	return restConfig, nil
}

// Return the AuthString include Auth type(Basic or Bearer)
func GetAuthString(in *restclient.Config, explicitKubeConfigPath string) (string, error) {
	// Checking Basic Auth
	if in.Username != "" {
		token, err := GetBasicAuthToken(in)
		return BasicAuthScheme + " " + token, err
	}

	token, err := GetBearerToken(in, explicitKubeConfigPath)
	return BearerAuthScheme + " " + token, err
}

func GetBasicAuthToken(in *restclient.Config) (string, error) {
	if in == nil {
		return "", errors.Errorf("RestClient can't be nil")
	}

	return encodeBasicAuthToken(in.Username, in.Password), nil
}

// convert the REST config into a bearer token
func GetBearerToken(in *restclient.Config, explicitKubeConfigPath string) (string, error) {
	if len(in.BearerToken) > 0 {
		return in.BearerToken, nil
	}

	if in == nil {
		return "", errors.Errorf("RestClient can't be nil")
	}
	if in.ExecProvider != nil {
		tc, err := in.TransportConfig()
		if err != nil {
			return "", err
		}

		auth, err := exec.GetAuthenticator(in.ExecProvider)
		if err != nil {
			return "", err
		}

		// This function will return error because of TLS Cert missing,
		// This code is not making actual request. We can ignore it.
		_ = auth.UpdateTransportConfig(tc)

		rt, err := transport.New(tc)
		if err != nil {
			return "", err
		}
		req := http.Request{Header: map[string][]string{}}

		newT := NewUserAgentRoundTripper("dummy", rt)
		_, _ = newT.RoundTrip(&req)

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
	return "", errors.Errorf("could not find a token")
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
				return "", errors.Errorf("Invalid expiry date in Kubeconfig. %v", err)
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

	_, _ = rt.RoundTrip(&req)
	return nil
}
