package common

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"

	"golang.org/x/oauth2/clientcredentials"
	"k8s.io/client-go/kubernetes"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util"
)

// ApplyHTTPAuth applies authentication configuration to an HTTP request
func ApplyHTTPAuth(ctx context.Context, req *http.Request, auth *wfv1.HTTPAuth, clientSet kubernetes.Interface, namespace string) error {
	if auth == nil {
		return nil
	}

	// Apply Basic Authentication
	if auth.BasicAuth.UsernameSecret != nil || auth.BasicAuth.PasswordSecret != nil {
		if err := applyBasicAuth(ctx, req, &auth.BasicAuth, clientSet, namespace); err != nil {
			return fmt.Errorf("failed to apply basic auth: %w", err)
		}
	}

	// Apply Client Certificate Authentication
	if auth.ClientCert.ClientCertSecret != nil || auth.ClientCert.ClientKeySecret != nil {
		if err := applyClientCertAuth(ctx, req, &auth.ClientCert, clientSet, namespace); err != nil {
			return fmt.Errorf("failed to apply client cert auth: %w", err)
		}
	}

	// Apply OAuth2 Authentication
	if auth.OAuth2.ClientIDSecret != nil {
		if err := applyOAuth2Auth(ctx, req, &auth.OAuth2, clientSet, namespace); err != nil {
			return fmt.Errorf("failed to apply oauth2 auth: %w", err)
		}
	}

	return nil
}

// CreateHTTPClientWithAuth creates an HTTP client configured with authentication
func CreateHTTPClientWithAuth(ctx context.Context, auth *wfv1.HTTPAuth, insecureSkipVerify bool, clientSet kubernetes.Interface, namespace string) (*http.Client, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecureSkipVerify,
			},
		},
	}

	if auth == nil {
		return client, nil
	}

	// Configure client certificate if provided
	if auth.ClientCert.ClientCertSecret != nil && auth.ClientCert.ClientKeySecret != nil {
		if err := configureClientCertTransport(ctx, client, &auth.ClientCert, clientSet, namespace); err != nil {
			return nil, fmt.Errorf("failed to configure client cert transport: %w", err)
		}
	}

	return client, nil
}

func applyBasicAuth(ctx context.Context, req *http.Request, basicAuth *wfv1.BasicAuth, clientSet kubernetes.Interface, namespace string) error {
	var username, password string

	if basicAuth.UsernameSecret != nil {
		usernameBytes, err := util.GetSecrets(ctx, clientSet, namespace, basicAuth.UsernameSecret.Name, basicAuth.UsernameSecret.Key)
		if err != nil {
			return fmt.Errorf("failed to get username secret: %w", err)
		}
		username = string(usernameBytes)
	}

	if basicAuth.PasswordSecret != nil {
		passwordBytes, err := util.GetSecrets(ctx, clientSet, namespace, basicAuth.PasswordSecret.Name, basicAuth.PasswordSecret.Key)
		if err != nil {
			return fmt.Errorf("failed to get password secret: %w", err)
		}
		password = string(passwordBytes)
	}

	if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}

	return nil
}

func applyClientCertAuth(ctx context.Context, req *http.Request, clientCert *wfv1.ClientCertAuth, clientSet kubernetes.Interface, namespace string) error {
	// Client certificate authentication is handled at the transport level
	// This function is a placeholder for any request-level cert handling if needed
	return nil
}

func configureClientCertTransport(ctx context.Context, client *http.Client, clientCert *wfv1.ClientCertAuth, clientSet kubernetes.Interface, namespace string) error {
	if clientCert.ClientCertSecret == nil || clientCert.ClientKeySecret == nil {
		return fmt.Errorf("both client cert and key secrets must be provided")
	}

	certBytes, err := util.GetSecrets(ctx, clientSet, namespace, clientCert.ClientCertSecret.Name, clientCert.ClientCertSecret.Key)
	if err != nil {
		return fmt.Errorf("failed to get client cert secret: %w", err)
	}

	keyBytes, err := util.GetSecrets(ctx, clientSet, namespace, clientCert.ClientKeySecret.Name, clientCert.ClientKeySecret.Key)
	if err != nil {
		return fmt.Errorf("failed to get client key secret: %w", err)
	}

	cert, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		return fmt.Errorf("failed to load client certificate: %w", err)
	}

	transport := client.Transport.(*http.Transport)
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}
	transport.TLSClientConfig.Certificates = []tls.Certificate{cert}

	return nil
}

func applyOAuth2Auth(ctx context.Context, req *http.Request, oauth2Auth *wfv1.OAuth2Auth, clientSet kubernetes.Interface, namespace string) error {
	if oauth2Auth.ClientIDSecret == nil || oauth2Auth.ClientSecretSecret == nil || oauth2Auth.TokenURLSecret == nil {
		return fmt.Errorf("client ID, client secret, and token URL must be provided for OAuth2")
	}

	clientIDBytes, err := util.GetSecrets(ctx, clientSet, namespace, oauth2Auth.ClientIDSecret.Name, oauth2Auth.ClientIDSecret.Key)
	if err != nil {
		return fmt.Errorf("failed to get client ID secret: %w", err)
	}

	clientSecretBytes, err := util.GetSecrets(ctx, clientSet, namespace, oauth2Auth.ClientSecretSecret.Name, oauth2Auth.ClientSecretSecret.Key)
	if err != nil {
		return fmt.Errorf("failed to get client secret: %w", err)
	}

	tokenURLBytes, err := util.GetSecrets(ctx, clientSet, namespace, oauth2Auth.TokenURLSecret.Name, oauth2Auth.TokenURLSecret.Key)
	if err != nil {
		return fmt.Errorf("failed to get token URL secret: %w", err)
	}

	config := &clientcredentials.Config{
		ClientID:     string(clientIDBytes),
		ClientSecret: string(clientSecretBytes),
		TokenURL:     string(tokenURLBytes),
		Scopes:       oauth2Auth.Scopes,
	}

	// Add endpoint parameters
	if len(oauth2Auth.EndpointParams) > 0 {
		endpointParams := url.Values{}
		for _, param := range oauth2Auth.EndpointParams {
			endpointParams.Set(param.Key, param.Value)
		}
		config.EndpointParams = endpointParams
	}

	token, err := config.Token(ctx)
	if err != nil {
		return fmt.Errorf("failed to obtain OAuth2 token: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	return nil
}