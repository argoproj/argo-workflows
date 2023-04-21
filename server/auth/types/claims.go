package types

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/go-jose/go-jose/v3/jwt"
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

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var httpClient HttpClient

func init() {
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

func TransTypeToStringList(v reflect.Value) []string {
	var s []string
	for i := 0; i < v.Len(); i++ {
		str, ok := v.Index(i).Interface().(string)
		if ok {
			s = append(s, str)
		}
	}
	return s
}

func GetUserInfoStruct(userInfoGroupsField string) any {
	newType := reflect.StructOf([]reflect.StructField{
		{
			Name: "Groups",
			Type: reflect.TypeOf([]string{""}),
			Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, userInfoGroupsField)),
		},
	})
	return reflect.New(newType).Interface()
}

func (c *Claims) GetUserInfoGroups(accessToken, issuer, userInfoPath string, userInfoGroupsField string) ([]string, error) {
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

	userInfo := GetUserInfoStruct(userInfoGroupsField)

	defer response.Body.Close()
	err = json.NewDecoder(response.Body).Decode(&userInfo)

	if err != nil {
		return nil, err
	}

	return TransTypeToStringList(reflect.ValueOf(userInfo).Elem().FieldByName("Groups")), nil
}
