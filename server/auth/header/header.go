package header

import (
	"fmt"
	"slices"
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
	config               config.HeaderConfig
	sharedSecret         string
	trustUnauthenticated bool
}

func (h *header) IsRBACEnabled() bool {
	return h.config.RBAC.IsEnabled()
}

func New(cfg config.HeaderConfig, sharedSecret string, trustUnauthenticated bool) Interface {
	return &header{
		config:               cfg,
		sharedSecret:         sharedSecret,
		trustUnauthenticated: trustUnauthenticated,
	}
}

func resolveClaim(source config.ClaimSource, md metadata.MD) string {
	if source.Value != "" {
		return source.Value
	}

	if source.Header != "" {
		values := md.Get(strings.ToLower(source.Header))
		if len(values) > 0 {
			return strings.Join(values, ",")
		}
	}

	return ""
}

func resolveGroups(source config.GroupClaimSource, md metadata.MD) []string {
	value := resolveClaim(source.ClaimSource, md)

	if value == "" {
		return nil
	}

	return strings.Split(value, ",")
}

func (h *header) Authorize(md metadata.MD) (*types.Claims, error) {
	if !h.trustUnauthenticated {
		if err := h.authenticateProxy(md); err != nil {
			return nil, err
		}
	}
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

func (h *header) authenticateProxy(md metadata.MD) error {
	if h.config.SharedSecret == nil {
		return fmt.Errorf("shared secret authentication is not configured")
	}

	values := md.Get(strings.ToLower(h.config.SharedSecret.Header))
	if len(values) == 0 {
		return fmt.Errorf("trusted proxy authentication header is missing")
	}

	if slices.Contains(values, h.sharedSecret) {
		return nil
	}

	return fmt.Errorf("trusted proxy authentication failed")
}
