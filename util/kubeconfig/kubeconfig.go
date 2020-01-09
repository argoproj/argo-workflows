package kubeconfig

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"k8s.io/client-go/plugin/pkg/client/auth/exec"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport"
)

// get the default one from the filesystem
func DefaultRestConfig() (*restclient.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{})
	return config.ClientConfig()
}

// convert a bearer token into a REST config
func GetRestConfig(token string) (*restclient.Config, error) {
	restConfigBytes, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}
	restConfig := &restclient.Config{}
	err = json.Unmarshal(restConfigBytes, restConfig)
	if err != nil {
		return nil, err
	}
	return restConfig, nil
}

// convert the REST config into a bearer token
func GetBearerToken(in *restclient.Config) (string, error) {
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
		in.BearerToken = strings.TrimPrefix(token, "Bearer ")
	}
	return in.BearerToken, nil
}
