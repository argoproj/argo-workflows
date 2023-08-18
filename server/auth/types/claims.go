package types

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-jose/go-jose/v3/jwt"
	log "github.com/sirupsen/logrus"
)

type Claims struct {
	jwt.Claims
	Groups                  []string               `json:"groups,omitempty"`
	Email                   string                 `json:"email,omitempty"`
	EmailVerified           bool                   `json:"email_verified,omitempty"`
	Name                    string                 `json:"name,omitempty"`
	ServiceAccountName      string                 `json:"service_account_name,omitempty"`
	ServiceAccountNamespace string                 `json:"service_account_namespace,omitempty"`
	PreferredUsername       string                 `json:"preferred_username,omitempty"`
	RawClaim                map[string]interface{} `json:"-"`
}

type UserInfo struct {
	Groups []string `json:"groups"`
}

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var httpClient HttpClient

func init() {
	// Default client
	httpClient = &http.Client{}
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

func (c *Claims) SetHttpClient(newHttpClient *http.Client) {
	httpClient = newHttpClient
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

func (c *Claims) GetUserInfoGroups(
	accessToken,
	issuer,
	userInfoPath string,
	customKeyName string,
) ([]string, error) {
	url := fmt.Sprintf("%s%s", issuer, userInfoPath)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	bearer := fmt.Sprintf("Bearer %s", accessToken)
	request.Header.Set("Authorization", bearer)

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("Could not read userInfo payload: %v", err)
	}
	log.Debug("UserInfo responseBody: ", string(responseBody))

	defer response.Body.Close()

	userInfo := UserInfo{}
	err = json.Unmarshal(responseBody, &userInfo)
	if err != nil {
		return nil, err
	}
	log.Debug("userInfo: ", userInfo)

	// Default to getting groups using "groups" key
	groups := userInfo.Groups

	// If a custom group key was given, use that
	if customKeyName != "" {
		log.Info(
			"A custom group key has been supplied, trying to retrieve the groups from key: ",
			customKeyName,
		)
		var userInfoRawMap map[string]json.RawMessage

		err = json.Unmarshal(responseBody, &userInfoRawMap)
		if err != nil {
			return nil, fmt.Errorf("Could not marshall userInfo payload")
		}

		err = json.Unmarshal(userInfoRawMap[customKeyName], &groups)
		if err != nil {
			return nil, fmt.Errorf("No claim found in userInfo for key: %v", customKeyName)
		}
	}
	log.Debug("UserInfo groups: ", groups)

	return groups, nil
}
