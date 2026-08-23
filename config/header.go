package config

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

	// Delimiter separates multiple groups in the header value.
	Delimiter string `json:"delimiter,omitempty"`
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
}
