package kubeconfig

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"k8s.io/client-go/plugin/pkg/client/auth/exec"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport"

	"github.com/argoproj/argo/pkg/apis/workflow"
)

// get the default one from the filesystem
func DefaultRestConfig() (*restclient.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{})
	return config.ClientConfig()
}

// convert a bearer token into a REST config
func GetRestConfig(token string) (*restclient.Config, error) {
	version, tokenBody, err := parseToken(token)
	if err != nil {
		return nil, err
	}
	switch version {
	case tokenVersion0:
		restConfigBytes, err := base64.StdEncoding.DecodeString(tokenBody)
		if err != nil {
			return nil, err
		}
		restConfig := &restclient.Config{}
		err = json.Unmarshal(restConfigBytes, restConfig)
		if err != nil {
			return nil, err
		}
		return restConfig, nil
	case tokenVersion1:
		restConfig, err := DefaultRestConfig()
		if err != nil {
			return nil, err
		}
		restConfig.BearerToken = tokenBody
		restConfig.BearerTokenFile = ""
		return restConfig, nil
	case tokenVersion2:
		value, err := getV2Token()
		if err != nil {
			return nil, err
		}
		if tokenBody != value {
			return nil, fmt.Errorf("v2 token invalid")
		}
		restConfig, err := DefaultRestConfig()
		if err != nil {
			return nil, err
		}
		return restConfig, nil
	}
	return nil, fmt.Errorf("invalid token tokenVersion")
}

// convert the REST config into a bearer token
func GetBearerToken(in *restclient.Config) (string, error) {
	switch getDefaultTokenVersion() {
	case tokenVersion0:
		tlsClientConfig, err := tlsClientConfig(in)
		if err != nil {
			return "", err
		}
		clientConfig := &workflow.ClientConfig{
			Host:    in.Host,
			APIPath: in.APIPath,
			ContentConfig: restclient.ContentConfig{
				AcceptContentTypes: in.ContentConfig.AcceptContentTypes,
				ContentType:        in.ContentConfig.ContentType,
				GroupVersion:       in.ContentConfig.GroupVersion,
			},
			Username:        in.Username,
			Password:        in.Password,
			BearerToken:     in.BearerToken,
			Impersonate:     in.Impersonate,
			AuthProvider:    in.AuthProvider,
			TLSClientConfig: tlsClientConfig,
			UserAgent:       in.UserAgent,
			QPS:             in.QPS,
			Burst:           in.Burst,
			Timeout:         in.Timeout,
		}
		configByte, err := json.Marshal(clientConfig)
		if err != nil {
			return "", err
		}
		return formatToken(0, base64.StdEncoding.EncodeToString(configByte)), nil
	case tokenVersion1:
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
			return "", nil
		}
	case tokenVersion2:
		return getV2Token()
	}
	return "", fmt.Errorf("invalid token version")
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
