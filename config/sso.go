package config

import (
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SSOConfig struct {
	Issuer       string                  `json:"issuer"`
	IssuerAlias  string                  `json:"issuerAlias,omitempty"`
	ClientID     apiv1.SecretKeySelector `json:"clientId"`
	ClientSecret apiv1.SecretKeySelector `json:"clientSecret"`
	RedirectURL  string                  `json:"redirectUrl"`
	RBAC         *RBACConfig             `json:"rbac,omitempty"`
	// additional scopes (on top of "openid")
	Scopes        []string        `json:"scopes,omitempty"`
	SessionExpiry metav1.Duration `json:"sessionExpiry,omitempty"`
	// customGroupClaimName will override the groups claim name
	CustomGroupClaimName string   `json:"customGroupClaimName,omitempty"`
	UserInfoPath         string   `json:"userInfoPath,omitempty"`
	FilterGroupsRegex    []string `json:"filterGroupsRegex,omitempty"`
	// client certificates used for mTLS with the provider
	ClientCert         string `json:"clientCert,omitempty"`
	ClientKey          string `json:"clientKey,omitempty"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty"`
	// custom CA certificate file
	CACert string `json:"caCert,omitempty"`
}

func (c SSOConfig) GetSessionExpiry() time.Duration {
	if c.SessionExpiry.Duration > 0 {
		return c.SessionExpiry.Duration
	}
	return 10 * time.Hour
}
