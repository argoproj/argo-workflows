package userinfo

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/coreos/go-oidc"
	"k8s.io/client-go/rest"
)

func UserInfoFor(restConfig *rest.Config) (*oidc.UserInfo, error) {
	username := restConfig.Username
	if username != "" {
		return &oidc.UserInfo{Subject: username}, nil
	} else if restConfig.BearerToken != "" || restConfig.BearerTokenFile != "" {
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
			return nil, fmt.Errorf("expected bearer token to be a JWT and therefore have 3 dot-delimited parts")
		}
		payload := parts[1]
		data, err := base64.RawStdEncoding.DecodeString(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to decode bearer token's JWT payload: %w", err)
		}
		claims := &oidc.UserInfo{}
		err = json.Unmarshal(data, &claims)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal bearer token's JWT payload: %w", err)
		}
		return claims, nil
	} else {
		return nil, nil
	}
}
