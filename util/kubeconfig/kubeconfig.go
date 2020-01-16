package kubeconfig

import (
	"github.com/pkg/errors"
	"io/ioutil"
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

	restConfig, err := DefaultRestConfig()
	if err != nil {
		return nil, err
	}
	restConfig.BearerToken = ""
	restConfig.BearerTokenFile = ""
	if token != "" {
		restConfig.BearerToken = token
	}
	return restConfig, nil
}

// convert the REST config into a bearer token
func GetBearerToken(in *restclient.Config) (string, error) {

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
		return formatToken(1, strings.TrimPrefix(token, "Bearer ")), nil
	}
	if in.AuthProvider != nil {
		if in.AuthProvider.Name == "gcp" {
			tc, err := in.TransportConfig()
			if err != nil {
				return "", nil
			}

			auth, err := restclient.GetAuthProvider(in.Host, in.AuthProvider, in.AuthConfigPersister)
			if err != nil {
				return "", nil
			}

			rt, err := transport.New(tc)
			if err != nil {
				return "", nil
			}
			rt = auth.WrapTransport(rt)
			req := http.Request{Header: map[string][]string{}}

			_, _ = rt.RoundTrip(&req)

			token := in.AuthProvider.Config["access-token"]
			return formatToken(1, strings.TrimPrefix(token, "Bearer ")), nil
		}
	}
	return in.BearerToken, nil
}

func tlsClientConfig(in *restclient.Config) (restclient.TLSClientConfig, error) {
	c := restclient.TLSClientConfig{
		Insecure:   in.TLSClientConfig.Insecure,
		ServerName: in.TLSClientConfig.ServerName,
		CertData:   in.TLSClientConfig.CertData,
		KeyData:    in.TLSClientConfig.KeyData,
		CAData:     in.TLSClientConfig.CAData,
	}
	if in.TLSClientConfig.CAFile != "" {
		data, err := ioutil.ReadFile(in.TLSClientConfig.CAFile)
		if err != nil {
			return c, err
		}
		c.CAData = data
	}
	if in.TLSClientConfig.CertFile != "" {
		data, err := ioutil.ReadFile(in.TLSClientConfig.CertFile)
		if err != nil {
			return c, err
		}
		c.CertData = data
	}
	if in.TLSClientConfig.KeyFile != "" {
		data, err := ioutil.ReadFile(in.TLSClientConfig.KeyFile)
		if err != nil {
			return c, err
		}
		c.KeyData = data
	}
	return c, nil
}
