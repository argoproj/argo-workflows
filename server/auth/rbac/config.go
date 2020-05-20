package rbac

import (
	"errors"

	corev1 "k8s.io/api/core/v1"
)

type Rule struct {
	Groups         []string                     `json:"groups"`
	ServiceAccount *corev1.LocalObjectReference `json:"serviceAccount"`
}

type Config struct {
	Rules                 []Rule                       `json:"rules"`
	DefaultServiceAccount *corev1.LocalObjectReference `json:"defaultServiceAccount"`
}

func (c Config) ServiceAccount(groups []string) (*corev1.LocalObjectReference, error) {
	hasGroup := make(map[string]bool)
	for _, group := range groups {
		hasGroup[group] = true
	}
	for _, rule := range c.Rules {
		for _, g := range rule.Groups {
			if hasGroup[g] {
				if rule.ServiceAccount == nil {
					return nil, errors.New("RBAC misconfigured: service account is empty")
				}
				return rule.ServiceAccount, nil
			}
		}
	}
	if c.DefaultServiceAccount != nil {
		return c.DefaultServiceAccount, nil
	}
	return nil, errors.New("no RBAC rule matches the provided groups")
}

var _ Interface = Config{}
