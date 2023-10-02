package serviceaccount

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/go-jose/go-jose/v3/jwt"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v3/server/auth/types"
)

func ClaimSetFor(restConfig *rest.Config) (*types.Claims, error) {
	username := restConfig.Username
	if username != "" {
		return &types.Claims{Claims: jwt.Claims{Subject: username}}, nil
	} else if restConfig.BearerToken != "" || restConfig.BearerTokenFile != "" {
		bearerToken := restConfig.BearerToken
		if bearerToken == "" {
			// should only ever be used for service accounts
			data, err := os.ReadFile(restConfig.BearerTokenFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read bearer token file: %w", err)
			}
			bearerToken = string(data)
		}
		parts := strings.SplitN(bearerToken, ".", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("expected bearer token to be a JWT and therefore have 3 dot-delimited parts")
		}
		payload := parts[1]
		data, err := base64.RawStdEncoding.DecodeString(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to decode bearer token's JWT payload: %w", err)
		}
		claims := &types.Claims{}
		err = json.Unmarshal(data, &claims)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal bearer token's JWT payload: %w", err)
		}
		return claims, nil
	} else {
		return nil, nil
	}
}
