package rbac

import "errors"

type Rule struct {
	Groups         []string `json:"groups"`
	ServiceAccount string   `json:"serviceAccount"`
}

type Config struct {
	Rules []Rule `json:"rules"`
}

func (c Config) ServiceAccount(groups []string) (string, error) {
	hasGroup := make(map[string]bool)
	for _, group := range groups {
		hasGroup[group] = true
	}
	for _, rule := range c.Rules {
		for _, g := range rule.Groups {
			if hasGroup[g] {
				if rule.ServiceAccount == "" {
					return "", errors.New("RBAC misconfigured: service account is empty")
				}
				return rule.ServiceAccount, nil
			}
		}

	}
	return "", errors.New("no RBAC rule matches the provided groups")
}

var _ Interface = Config{}
