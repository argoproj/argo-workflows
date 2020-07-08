package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"golang.org/x/oauth2/jwt"
	"k8s.io/client-go/rest"
)

func getJWT(restConfig *rest.Config) (*jwt.Config, error) {
	username := restConfig.Username
	if username != "" {
		return &jwt.Config{Subject: username}, nil
	} else {
		bearerToken := restConfig.BearerToken
		if bearerToken == "" {
			data, err := ioutil.ReadFile(restConfig.BearerTokenFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read bearer token file: %w", err)
			}
			bearerToken = string(data)
		}
		parts := strings.SplitN(bearerToken, ".", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("expected 3 parts: %w", parts)
		}
		payload := parts[1]
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to get decode bearer token: %w", err)
		}
		claims := &jwt.Config{}
		err = json.Unmarshal(data, claims)
		if err != nil {
			return nil, fmt.Errorf("failed to get unmarshal JWT payload: %w", err)
		}
		return claims, nil
	}
}
