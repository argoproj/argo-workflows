package types

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-jose/go-jose/v3/jwt"
)

type Claims struct {
	jwt.Claims
	Groups                  []string               `json:"groups,omitempty"`
	Email                   string                 `json:"email,omitempty"`
	EmailVerified           bool                   `json:"-"`
	Name                    string                 `json:"name,omitempty"`
	ServiceAccountName      string                 `json:"service_account_name,omitempty"`
	ServiceAccountNamespace string                 `json:"service_account_namespace,omitempty"`
	PreferredUsername       string                 `json:"preferred_username,omitempty"`
	RawClaim                map[string]interface{} `json:"-"`
}

type UserInfo struct {
	Groups []string `json:"groups"`
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var httpClient HTTPClient

func init() {
	httpClient = &http.Client{}
}

// UnmarshalJSON is a custom Unmarshal that overwrites
// json.Unmarshal to mash every claim into a custom map
func (c *Claims) UnmarshalJSON(data []byte) error {
	type claimAlias Claims
	var localClaim = claimAlias(*c)

	// Populate the claims struct as much as possible
	err := json.Unmarshal(data, &localClaim)
	if err != nil {
		return err
	}

	// Populate the raw data struct
	err = json.Unmarshal(data, &localClaim.RawClaim)
	if err != nil {
		return err
	}

	if localClaim.RawClaim["email_verified"] == true || localClaim.RawClaim["email_verified"] == "true" {
		localClaim.EmailVerified = true
	}

	*c = Claims(localClaim)
	return nil
}

// GetCustomGroup is responsible for extracting groups based on the
// provided custom claim key
func (c *Claims) GetCustomGroup(customKeyName string) ([]string, error) {
	groups, ok := c.RawClaim[customKeyName]
	if !ok {
		return nil, fmt.Errorf("no claim found for key: %v", customKeyName)
	}

	sliceInterface, ok := groups.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected an array, got %v", groups)
	}

	newSlice := []string{}
	for _, a := range sliceInterface {
		val, ok := a.(string)
		if !ok {
			return nil, fmt.Errorf("group name %v was not a string", a)
		}
		newSlice = append(newSlice, val)
	}

	return newSlice, nil
}

func (c *Claims) GetUserInfoGroups(ctx context.Context, httpClient HTTPClient, accessToken, issuer, userInfoPath string) ([]string, error) {
	url := fmt.Sprintf("%s%s", issuer, userInfoPath)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)

	if err != nil {
		return nil, err
	}

	bearer := fmt.Sprintf("Bearer %s", accessToken)
	request.Header.Set("Authorization", bearer)

	response, err := httpClient.Do(request)

	if err != nil {
		return nil, err
	}

	userInfo := UserInfo{}

	defer response.Body.Close()
	err = json.NewDecoder(response.Body).Decode(&userInfo)

	if err != nil {
		return nil, err
	}

	return userInfo.Groups, nil
}
