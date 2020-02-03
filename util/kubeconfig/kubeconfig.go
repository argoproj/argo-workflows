package kubeconfig

import (
	"encoding/base64"
	"net/http"
	"os"
	"strings"

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

//Return the AuthString include Auth type(Basic or Bearer)
func GetAuthString(in *restclient.Config) (string, error) {
	//Checking Basic Auth
	if in.Username != "" {
		token, err := GetBasicAuthToken(in)
		return BasicAuthScheme + " " + token, err
	}

	token, err := GetBearerToken(in)
	return BearerAuthScheme + " " + token, err
}

func GetBasicAuthToken(in *restclient.Config) (string, error) {

	if in == nil {
		return "", errors.Errorf("RestClient can't be nil")
	}

	return encodeBasicAuthToken(in.Username, in.Password), nil
}

// convert the REST config into a bearer token
func GetBearerToken(in *restclient.Config) (string, error) {

	if len(in.BearerToken) > 0 {
		return in.BearerToken, nil
	}

	if token := getEnvToken(); token != "" {
		return token, nil
	}

	if in == nil {
		return "", errors.Errorf("RestClient can't be nil")
	}
	if in.ExecProvider != nil {
		tc, err := in.TransportConfig()
		if err != nil {
			return "", nil
		}

		auth, err := exec.GetAuthenticator(in.ExecProvider)
		if err != nil {
			return "", nil
		}

		err = auth.UpdateTransportConfig(tc)
		if err != nil {
			return "", nil
		}

		rt, err := transport.New(tc)
		if err != nil {
			return "", nil
		}
		req := http.Request{Header: map[string][]string{}}

		_, _ = rt.RoundTrip(&req)

		token := req.Header.Get("Authorization")
		return strings.TrimPrefix(token, "Bearer "), nil
	}
	if in.AuthProvider != nil {
		if in.AuthProvider.Name == "gcp" {
			tc, err := in.TransportConfig()
			if err != nil {
				return "", err
			}

			auth, err := restclient.GetAuthProvider(in.Host, in.AuthProvider, in.AuthConfigPersister)
			if err != nil {
				return "", err
			}

			rt, err := transport.New(tc)
			if err != nil {
				return "", err
			}
			rt = auth.WrapTransport(rt)
			req := http.Request{Header: map[string][]string{}}

			_, _ = rt.RoundTrip(&req)

			token := in.AuthProvider.Config["access-token"]
			return strings.TrimPrefix(token, "Bearer "), nil
		}
	}
	return "", errors.Errorf("could not find a token")
}

// Get the Auth token from environment variable
func getEnvToken() string {
	return os.Getenv("ARGO_TOKEN")
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
