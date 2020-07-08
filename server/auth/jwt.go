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
			// should only ever be used for service accounts
			data, err := ioutil.ReadFile(restConfig.BearerTokenFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read bearer token file: %w", err)
			}
			bearerToken = string(data)
		}
		parts := strings.SplitN(bearerToken, ".", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("expected bearer token to be a JWT and have 3 dot-delimited parts")
		}
		payload := parts[1]
		data, err := base64.RawStdEncoding.DecodeString(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to get decode bearer token's JWT payload: %w", err)
		}
		claims := make(map[string]string)
		err = json.Unmarshal(data, &claims)
		if err != nil {
			return nil, fmt.Errorf("failed to get unmarshal bearer token's JWT payload: %w", err)
		}
		return &jwt.Config{Subject: claims["sub"]}, nil
	}
}
