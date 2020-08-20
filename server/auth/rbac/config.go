package rbac

import (
	"errors"

	corev1 "k8s.io/api/core/v1"
)

type Config struct {
	// A list of rules in order of precedence that we attempt to match.
	Rules []Rule `json:"rules,omitempty"`
	// If not rules match, or there are no rules, use this account.
	DefaultServiceAccountRef *corev1.LocalObjectReference `json:"defaultServiceAccountRef,omitempty"`
}

// Get the service account to use. It maybe nil - if the config is nil.
func (c *Config) GetServiceAccount(groups []string) (*corev1.LocalObjectReference, error) {
	if c == nil {
		return nil, nil
	}
	for _, r := range c.Rules {
		if r.Matches(groups) {
			return &r.ServiceAccountRef, nil
		}
	}
	if c.DefaultServiceAccountRef != nil {
		return c.DefaultServiceAccountRef, nil
	}
	return nil, errors.New("no RBAC rules match")
}

type Rule struct {
	// Match if the user has any of these groups.
	AnyOf []string `json:"anyOf,omitempty"`
	// The service account to use.
	ServiceAccountRef corev1.LocalObjectReference `json:"serviceAccountRef"`
}

func (r Rule) Matches(groups []string) bool {
	hasGroups := make(map[string]bool)
	for _, g := range groups {
		hasGroups[g] = true
	}
	for _, g := range r.AnyOf {
		if hasGroups[g] {
			return true
		}
	}
	return false
}
