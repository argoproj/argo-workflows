package webhdfs

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	fpath "path"
	"strconv"

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

func buildUrl(endpoint, path string, overwrite *bool, operation WebhdfsOperation) (*url.URL, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	u.Path = fpath.Join(u.Path, path)
	q := u.Query()
	q.Set("op", string(operation))
	if overwrite != nil {
		q.Set("overwrite", strconv.FormatBool(*overwrite))
	}
	u.RawQuery = q.Encode()
	return u, err
}

func CreateOauth2Client(clientID, clientSecret, tokenURL string, endpointParams []wfv1.EndpointParam) *http.Client {
	values := url.Values{}
	for _, endpointParam := range endpointParams {
		values.Add(endpointParam.Key, endpointParam.Value)
	}
	ctx := context.Background()
	conf := cc.Config{
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		TokenURL:       tokenURL,
		EndpointParams: values,
	}
	return conf.Client(ctx)
}
