package http

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"

	cc "golang.org/x/oauth2/clientcredentials"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func CreateClientWithCertificate(clientCert, clientKey []byte) (*http.Client, error) {
	cert, err := tls.X509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}
	return client, err
}

func CreateOauth2Client(ctx context.Context, clientID, clientSecret, tokenURL string, scopes []string, endpointParams []wfv1.OAuth2EndpointParam) *http.Client {
	values := url.Values{}
	for _, endpointParam := range endpointParams {
		values.Add(endpointParam.Key, endpointParam.Value)
	}
	conf := cc.Config{
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		TokenURL:       tokenURL,
		EndpointParams: values,
		Scopes:         scopes,
	}
	return conf.Client(ctx)
}
