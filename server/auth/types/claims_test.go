package types

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalJSON(t *testing.T) {
	testExpiry := jwt.NumericDate(1626527469)
	testissuedAt := jwt.NumericDate(1626467469)

	tests := []struct {
		description     string
		data            string
		customClaimName string
		expectedClaims  *Claims
		expectedErr     error
	}{
		{
			description:     "unmarshal valid data",
			data:            `{"user_tz":"America\/Chicago","sub":"test-user@argoproj.github.io","user_locale":"en","idp_name":"UserNamePassword","user.tenant.name":"test-user","onBehalfOfUser":true,"idp_guid":"UserNamePassword","amr":["USERNAME_PASSWORD"],"iss":"https:\/\/identity-service.argoproj.github.io","user_tenantname":"test-user","client_id":"tokenGenerator","user_isAdmin":true,"sub_type":"user","scope":"","client_tenantname":"argo-proj","region_name":"us1","user_lang":"en","userAppRoles":["Authenticated","Global Viewer","Identity Domain Administrator"],"exp":1626527469,"iat":1626467469,"client_guid":"adsf34534645654653454","client_name":"tokenGenerator","idp_type":"LOCAL","tenant":"test-user23523423","jti":"345sd435d454356","ad_groups":["argo_admin", "argo_readonly"],"gtp":"jwt","user_displayname":"Test User","sub_mappingattr":"userName","primTenant":true,"tok_type":"AT","ca_guid":"test-ca_guid","aud":["example-aud"],"user_id":"8948923893458945234","clientAppRoles":["Authenticated Client","Cross Tenant"],"tenant_iss":"https:\/\/identiy-service.argoproj.github.io"}`,
			customClaimName: "ad_groups",
			expectedErr:     nil,
			expectedClaims: &Claims{
				Claims: jwt.Claims{
					ID:        "345sd435d454356",
					Audience:  jwt.Audience{"example-aud"},
					Issuer:    "https://identity-service.argoproj.github.io",
					Subject:   "test-user@argoproj.github.io",
					Expiry:    &testExpiry,
					NotBefore: nil,
					IssuedAt:  &testissuedAt,
				},
				ServiceAccountName: "",
				RawClaim: map[string]interface{}{
					"ad_groups":         []interface{}{"argo_admin", "argo_readonly"},
					"amr":               []interface{}{"USERNAME_PASSWORD"},
					"aud":               []interface{}{"example-aud"},
					"clientAppRoles":    []interface{}{"Authenticated Client", "Cross Tenant"},
					"userAppRoles":      []interface{}{"Authenticated", "Global Viewer", "Identity Domain Administrator"},
					"ca_guid":           "test-ca_guid",
					"client_guid":       "adsf34534645654653454",
					"client_id":         "tokenGenerator",
					"client_name":       "tokenGenerator",
					"client_tenantname": "argo-proj",
					"exp":               1.626527469e+09,
					"gtp":               "jwt",
					"iat":               1.626467469e+09,
					"idp_guid":          "UserNamePassword",
					"idp_name":          "UserNamePassword",
					"idp_type":          "LOCAL",
					"iss":               "https://identity-service.argoproj.github.io",
					"jti":               "345sd435d454356",
					"onBehalfOfUser":    true,
					"primTenant":        true,
					"region_name":       "us1",
					"scope":             "",
					"sub":               "test-user@argoproj.github.io",
					"sub_mappingattr":   "userName",
					"sub_type":          "user",
					"tenant":            "test-user23523423",
					"tenant_iss":        "https://identiy-service.argoproj.github.io",
					"tok_type":          "AT",
					"user.tenant.name":  "test-user",
					"user_id":           "8948923893458945234",
					"user_isAdmin":      true,
					"user_lang":         "en",
					"user_locale":       "en",
					"user_tenantname":   "test-user",
					"user_tz":           "America/Chicago",
					"user_displayname":  "Test User",
				},
			},
		},
		{
			description: "unmarshal valid data, with default custom groups name",
			data:        `{"user_tz":"America\/Chicago","sub":"test-user@argoproj.github.io","user_locale":"en","idp_name":"UserNamePassword","user.tenant.name":"test-user","onBehalfOfUser":true,"idp_guid":"UserNamePassword","amr":["USERNAME_PASSWORD"],"iss":"https:\/\/identity-service.argoproj.github.io","user_tenantname":"test-user","client_id":"tokenGenerator","user_isAdmin":true,"sub_type":"user","scope":"","client_tenantname":"argo-proj","region_name":"us1","user_lang":"en","userAppRoles":["Authenticated","Global Viewer","Identity Domain Administrator"],"exp":1626527469,"iat":1626467469,"client_guid":"adsf34534645654653454","client_name":"tokenGenerator","idp_type":"LOCAL","tenant":"test-user23523423","jti":"345sd435d454356","groups":["argo_admin", "argo_readonly"],"gtp":"jwt","user_displayname":"Test User","sub_mappingattr":"userName","primTenant":true,"tok_type":"AT","ca_guid":"test-ca_guid","aud":["example-aud"],"user_id":"8948923893458945234","clientAppRoles":["Authenticated Client","Cross Tenant"],"tenant_iss":"https:\/\/identiy-service.argoproj.github.io"}`,
			expectedErr: nil,
			expectedClaims: &Claims{
				Claims: jwt.Claims{
					ID:        "345sd435d454356",
					Audience:  jwt.Audience{"example-aud"},
					Issuer:    "https://identity-service.argoproj.github.io",
					Subject:   "test-user@argoproj.github.io",
					Expiry:    &testExpiry,
					NotBefore: nil,
					IssuedAt:  &testissuedAt,
				},
				Groups:             []string{"argo_admin", "argo_readonly"},
				ServiceAccountName: "",
				RawClaim: map[string]interface{}{
					"groups":            []interface{}{"argo_admin", "argo_readonly"},
					"amr":               []interface{}{"USERNAME_PASSWORD"},
					"aud":               []interface{}{"example-aud"},
					"clientAppRoles":    []interface{}{"Authenticated Client", "Cross Tenant"},
					"userAppRoles":      []interface{}{"Authenticated", "Global Viewer", "Identity Domain Administrator"},
					"ca_guid":           "test-ca_guid",
					"client_guid":       "adsf34534645654653454",
					"client_id":         "tokenGenerator",
					"client_name":       "tokenGenerator",
					"client_tenantname": "argo-proj",
					"exp":               1.626527469e+09,
					"gtp":               "jwt",
					"iat":               1.626467469e+09,
					"idp_guid":          "UserNamePassword",
					"idp_name":          "UserNamePassword",
					"idp_type":          "LOCAL",
					"iss":               "https://identity-service.argoproj.github.io",
					"jti":               "345sd435d454356",
					"onBehalfOfUser":    true,
					"primTenant":        true,
					"region_name":       "us1",
					"scope":             "",
					"sub":               "test-user@argoproj.github.io",
					"sub_mappingattr":   "userName",
					"sub_type":          "user",
					"tenant":            "test-user23523423",
					"tenant_iss":        "https://identiy-service.argoproj.github.io",
					"tok_type":          "AT",
					"user.tenant.name":  "test-user",
					"user_id":           "8948923893458945234",
					"user_isAdmin":      true,
					"user_lang":         "en",
					"user_locale":       "en",
					"user_tenantname":   "test-user",
					"user_tz":           "America/Chicago",
					"user_displayname":  "Test User",
				},
			},
		},
		{
			description: "email verify field as string",
			data:        `{"email_verified":"true"}`,
			expectedErr: nil,
			expectedClaims: &Claims{
				RawClaim: map[string]interface{}{
					"email_verified": "true",
				},
				EmailVerified: true,
			},
		},
		{
			description: "email verify field as bool",
			data:        `{"email_verified":true}`,
			expectedErr: nil,
			expectedClaims: &Claims{
				RawClaim: map[string]interface{}{
					"email_verified": true,
				},
				EmailVerified: true,
			},
		},
		{
			description: "unmarshal no data",
			data:        `{}`,
			expectedErr: nil,
			expectedClaims: &Claims{
				RawClaim: map[string]interface{}{},
			},
		},
	}
	for _, test := range tests {

		claims := &Claims{}
		err := json.Unmarshal([]byte(test.data), &claims)

		assert.Equal(t, test.expectedErr, err, test.description)
		assert.Equal(t, test.expectedClaims, claims, test.description)
	}
}

func TestGetCustomGroup(t *testing.T) {

	t.Run("NoCustomGroupSet", func(t *testing.T) {
		claims := &Claims{}
		_, err := claims.GetCustomGroup(("ad_groups"))
		require.EqualError(t, err, "no claim found for key: ad_groups")
	})
	t.Run("CustomGroupSet", func(t *testing.T) {
		tGroup := []string{"my-group"}
		tGroupsIf := make([]interface{}, len(tGroup))
		for i := range tGroup {
			tGroupsIf[i] = tGroup[i]
		}
		claims := &Claims{RawClaim: map[string]interface{}{
			"ad_groups": tGroupsIf,
		}}
		groups, err := claims.GetCustomGroup(("ad_groups"))
		require.NoError(t, err)
		assert.Equal(t, []string{"my-group"}, groups)
	})
	t.Run("CustomGroupNotString", func(t *testing.T) {
		tGroup := []int{0}
		tGroupsIf := make([]interface{}, len(tGroup))
		for i := range tGroup {
			tGroupsIf[i] = tGroup[i]
		}
		claims := &Claims{RawClaim: map[string]interface{}{
			"ad_groups": tGroupsIf,
		}}
		_, err := claims.GetCustomGroup(("ad_groups"))
		require.EqualError(t, err, "group name 0 was not a string")
	})
	t.Run("CustomGroupNotSlice", func(t *testing.T) {
		tGroup := "None"
		claims := &Claims{RawClaim: map[string]interface{}{
			"ad_groups": tGroup,
		}}
		_, err := claims.GetCustomGroup(("ad_groups"))
		require.Error(t, err)
	})
}

type HTTPClientMock struct {
	StatusCode int
	Body       io.ReadCloser
}

func (c *HTTPClientMock) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: c.StatusCode,
		Body:       c.Body,
	}, nil
}

func TestGetUserInfoGroups(t *testing.T) {
	t.Run("UserInfoGroupsReturn", func(t *testing.T) {
		userInfo := UserInfo{Groups: []string{"Everyone"}}
		userInfoBytes, _ := json.Marshal(userInfo)
		body := io.NopCloser(bytes.NewReader(userInfoBytes))

		httpClient = &HTTPClientMock{StatusCode: 200, Body: body}

		claims := &Claims{}
		groups, err := claims.GetUserInfoGroups(httpClient, "Bearer fake", "https://fake.okta.com", "/user-info")
		assert.Equal(t, []string{"Everyone"}, groups)
		require.NoError(t, err)
	})
}
