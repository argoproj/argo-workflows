package kubeconfig

import (
	"encoding/base64"
	"encoding/json"
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
		tc, _ := in.TransportConfig()
		auth, _ := exec.GetAuthenticator(in.ExecProvider)
		_ = auth.UpdateTransportConfig(tc)
		rt, _ := transport.New(tc)
		req := http.Request{Header: map[string][]string{}}
		_, _ = rt.RoundTrip(&req)
		token := req.Header.Get("Authorization")
		in.BearerToken = strings.TrimPrefix(token, "Bearer ")
	}
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
	return base64.StdEncoding.EncodeToString(configByte), nil
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
