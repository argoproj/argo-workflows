package types

import (
	"encoding/json"
	"fmt"

	"gopkg.in/square/go-jose.v2/jwt"
)

type Claims struct {
	jwt.Claims
	Groups                    []string               `json:"groups,omitempty"`
	Email                     string                 `json:"email,omitempty"`
	EmailVerified             bool                   `json:"email_verified,omitempty"`
	CurrentServiceAccountName string                 `json:"current_service_account_name,omitempty"`
	ServiceAccountNames       []string               `json:"service_account_names,omitempty"`
	RawClaim                  map[string]interface{} `json:"-"`
}

// UnmarshalJSON is a custom Unmarshal that overwrites
// json.Unmarshal to mash every claim into a custom map
func (c *Claims) UnmarshalJSON(data []byte) error {
	type claimAlias Claims
	var localClaim claimAlias = claimAlias(*c)

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

	*c = Claims(localClaim)
	return nil
}

// GetCustomGroup is responsible for extracting groups based on the
// provided custom claim key
func (c *Claims) GetCustomGroup(customKeyName string) ([]string, error) {
	groups, ok := c.RawClaim[customKeyName]
	if !ok {
		return nil, fmt.Errorf("No claim found for key: %v", customKeyName)
	}

	sliceInterface, ok := groups.([]interface{})
	if !ok {
		return nil, fmt.Errorf("Expected an array, got %v", groups)
	}

	newSlice := []string{}
	for _, a := range sliceInterface {
		val, ok := a.(string)
		if !ok {
			return nil, fmt.Errorf("Group name %v was not a string", a)
		}
		newSlice = append(newSlice, val)
	}

	return newSlice, nil
}
