package header

import (
	"fmt"
	"strings"

	"google.golang.org/grpc/metadata"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/server/auth/types"
)

type Interface interface {
	Authorize(md metadata.MD) (*types.Claims, error)
	IsRBACEnabled() bool
}

type header struct {
	config config.HeaderConfig
}

func (h *header) IsRBACEnabled() bool {
	return h.config.RBAC.IsEnabled()
}

func New(cfg config.HeaderConfig) Interface {
	return &header{
		config: cfg,
	}
}

func resolveClaim(source config.ClaimSource, md metadata.MD) string {
	if source.Value != "" {
		return source.Value
	}

	if source.Header != "" {
		values := md.Get(strings.ToLower(source.Header))
		if len(values) > 0 {
			return values[0]
		}
	}

	return ""
}

func resolveGroups(source config.GroupClaimSource, md metadata.MD) []string {
	value := resolveClaim(source.ClaimSource, md)

	if value == "" {
		return nil
	}

	if source.Delimiter == "" {
		return []string{value}
	}

	groups := strings.Split(value, source.Delimiter)
	result := make([]string, 0, len(groups))

	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		result = append(result, group)
	}

	return result
}

func (h *header) Authorize(md metadata.MD) (*types.Claims, error) {
	claims := &types.Claims{}

	claims.Issuer = resolveClaim(h.config.Issuer, md)
	claims.Subject = resolveClaim(h.config.Subject, md)

	if claims.Subject == "" {
		return nil, fmt.Errorf("subject claim is empty")
	}

	claims.Email = resolveClaim(h.config.Email, md)
	claims.PreferredUsername = resolveClaim(h.config.PreferredUsername, md)
	claims.Groups = resolveGroups(h.config.Groups, md)

	return claims, nil
}
