package config

import (
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SSOConfig contains single sign-on configuration settings
type SSOConfig struct {
	// Issuer is the OIDC issuer URL
	Issuer string `json:"issuer"`
	// IssuerAlias is an optional alias for the issuer
	IssuerAlias string `json:"issuerAlias,omitempty"`
	// ClientID references a secret containing the OIDC client ID
	ClientID apiv1.SecretKeySelector `json:"clientId"`
	// ClientSecret references a secret containing the OIDC client secret
	ClientSecret apiv1.SecretKeySelector `json:"clientSecret"`
	// RedirectURL is the OIDC redirect URL
	RedirectURL string `json:"redirectUrl"`
	// RBAC contains role-based access control settings
	RBAC *RBACConfig `json:"rbac,omitempty"`
	// additional scopes (on top of "openid")
	Scopes []string `json:"scopes,omitempty"`
	// SessionExpiry specifies how long user sessions last
	SessionExpiry metav1.Duration `json:"sessionExpiry,omitempty"`
	// CustomGroupClaimName will override the groups claim name
	CustomGroupClaimName string `json:"customGroupClaimName,omitempty"`
	// UserInfoPath specifies the path to user info endpoint
	UserInfoPath string `json:"userInfoPath,omitempty"`
	// InsecureSkipVerify skips TLS certificate verification
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`
	// FilterGroupsRegex filters groups using regular expressions
	FilterGroupsRegex []string `json:"filterGroupsRegex,omitempty"`
	// custom PEM encoded CA certificate file contents
	RootCA string `json:"rootCA,omitempty"`
	// custom CA certificate file name
	RootCAFile string `json:"rootCAFile,omitempty"`
}

func (c SSOConfig) GetSessionExpiry() time.Duration {
	if c.SessionExpiry.Duration > 0 {
		return c.SessionExpiry.Duration
	}
	return 10 * time.Hour
}
