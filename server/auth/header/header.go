package header

import (
	"strings"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/server/auth/types"
	"google.golang.org/grpc/metadata"
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

	return strings.Split(value, source.Delimiter)
}

func (h *header) Authorize(md metadata.MD) (*types.Claims, error) {
	claims := &types.Claims{}

	claims.Claims.Issuer = resolveClaim(h.config.Issuer, md)
	claims.Claims.Subject = resolveClaim(h.config.Subject, md)

	claims.Email = resolveClaim(h.config.Email, md)
	claims.PreferredUsername = resolveClaim(h.config.PreferredUsername, md)
	claims.Groups = resolveGroups(h.config.Groups, md)

	return claims, nil
}
