package config

import apiv1 "k8s.io/api/core/v1"

// ClaimSource specifies how to populate a claim.
type ClaimSource struct {
	// Header specifies the HTTP header containing the claim value.
	Header string `json:"header,omitempty"`

	// Value specifies a static value for the claim.
	Value string `json:"value,omitempty"`
}

// GroupClaimSource specifies how to populate the groups claim.
type GroupClaimSource struct {
	ClaimSource
}

// SharedSecretHeader configures a shared secret used to authenticate
// the trusted authentication proxy.
type SharedSecretHeader struct {
	// Header specifies the HTTP header containing the shared secret.
	Header string `json:"header,omitempty"`

	// RequiredValue references the Kubernetes Secret containing the
	// expected shared secret value.
	RequiredValue apiv1.SecretKeySelector `json:"requiredValue,omitzero"`
}

// HeaderConfig contains trusted header authentication configuration settings.
type HeaderConfig struct {
	// Issuer configures the issuer claim.
	Issuer ClaimSource `json:"iss,omitzero"`

	// Subject configures the subject claim.
	Subject ClaimSource `json:"sub,omitzero"`

	// Email configures the email claim.
	Email ClaimSource `json:"email,omitzero"`

	// PreferredUsername configures the preferred_username claim.
	PreferredUsername ClaimSource `json:"preferred_username,omitzero"`

	// Groups configures the groups claim.
	Groups GroupClaimSource `json:"groups,omitzero"`

	// SharedSecret configures authentication of the trusted proxy.
	SharedSecret *SharedSecretHeader `json:"sharedSecret,omitempty"`

	// RBAC configures role-based access controls
	RBAC *RBACConfig `json:"rbac,omitempty"`
}
