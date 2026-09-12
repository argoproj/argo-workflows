package header

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"

	"google.golang.org/grpc/metadata"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

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

func getHeaderSharedSecret(
	ctx context.Context,
	secretsIf corev1.SecretInterface,
	cfg *config.SharedSecretHeader,
) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("shared secret authentication is not configured")
	}
	if cfg.Header == "" {
		return "", fmt.Errorf("shared secret header is empty")
	}
	secretRef := cfg.RequiredValue
	if secretRef.Name == "" || secretRef.Key == "" {
		return "", fmt.Errorf("shared secret reference is empty")
	}

	secret, err := secretsIf.Get(ctx, secretRef.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get shared secret: %w", err)
	}

	value, ok := secret.Data[secretRef.Key]
	if !ok {
		return "", fmt.Errorf(
			"key %s missing in secret %s",
			secretRef.Key,
			secretRef.Name,
		)
	}

	if len(value) == 0 {
		return "", fmt.Errorf("shared secret value is empty")
	}

	return string(value), nil
}

func New(ctx context.Context, cfg config.HeaderConfig, secretsIf corev1.SecretInterface, trustUnauthenticated bool) (Interface, error) {
	var sharedSecret string

	if !trustUnauthenticated {
		var err error
		sharedSecret, err = getHeaderSharedSecret(
			ctx,
			secretsIf,
			cfg.SharedSecret,
		)
		if err != nil {
			return nil, err
		}
	}
	return &header{
		config:               cfg,
		sharedSecret:         sharedSecret,
		trustUnauthenticated: trustUnauthenticated,
	}, nil
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
	if err := h.authenticateProxy(md); err != nil {
		return nil, err
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
	if h.trustUnauthenticated {
		return nil
	}

	if h.config.SharedSecret == nil {
		return fmt.Errorf("shared secret authentication is not configured")
	}

	values := md.Get(h.config.SharedSecret.Header)
	if len(values) == 0 {
		return fmt.Errorf("trusted proxy authentication header is missing")
	}

	providedSecret := strings.Join(values, ",")

	if subtle.ConstantTimeCompare(
		[]byte(providedSecret),
		[]byte(h.sharedSecret),
	) != 1 {
		return fmt.Errorf("trusted proxy authentication failed")
	}

	return nil
}
